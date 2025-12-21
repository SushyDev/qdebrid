package qbittorrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/sushydev/real_debrid_go/api"
	"go.uber.org/zap"
	"qdebrid/internal/cache"
	"qdebrid/internal/config"
	"qdebrid/internal/debrid"
	"qdebrid/internal/servarr"
	"qdebrid/internal/torrent"
)

// Handler handles qBittorrent API requests
type Handler struct {
	debridClient       *debrid.Client
	servarrClient      *servarr.Client
	torrentService     *torrent.Service
	cache              *cache.Cache
	config             *config.Config
	logger             *zap.Logger
	processingTorrents *ProcessingTorrents
	shutdownCh         chan struct{}
}

// NewHandler creates a new qBittorrent API handler
func NewHandler(
	debridClient *debrid.Client,
	servarrClient *servarr.Client,
	cache *cache.Cache,
	cfg *config.Config,
	logger *zap.Logger,
) *Handler {
	// Create torrent service with validation
	torrentService := torrent.NewService(
		debridClient,
		&cfg.MediaValidation,
		logger.Named("torrent"),
	)

	handler := &Handler{
		debridClient:       debridClient,
		servarrClient:      servarrClient,
		torrentService:     torrentService,
		cache:              cache,
		config:             cfg,
		logger:             logger,
		processingTorrents: NewProcessingTorrents(),
		shutdownCh:         make(chan struct{}),
	}

	// Start background cleanup job for failed torrents
	go handler.cleanupFailedTorrents()

	return handler
}

// respondJSON writes JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Marshal for both logging and response
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("failed to marshal response", zap.Error(err))
		return
	}

	// Log response at debug level (full JSON, no truncation)
	h.logger.Debug("response json", zap.String("json", string(jsonData)))

	// Write response
	if _, err := w.Write(jsonData); err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
	}
}

// respondError writes error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.logger.Warn("handler error", zap.Int("status", status), zap.String("message", message))
	http.Error(w, message, status)
}

// respondText writes plain text response
func (h *Handler) respondText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(text))
}

// Health handles /health - returns service health status
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	// Simple health check - if we can respond, we're healthy
	health := map[string]interface{}{
		"status":  "ok",
		"service": "qdebrid",
		"version": "2.0.0",
	}
	h.respondJSON(w, http.StatusOK, health)
}

// Login handles /api/v2/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("auth/login")
	h.respondText(w, http.StatusOK, "Ok.")
}

// Version handles /api/v2/app/webapiVersion
func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("app/webapiVersion")
	h.respondText(w, http.StatusOK, "2.9.3")
}

// Preferences handles /api/v2/app/preferences
func (h *Handler) Preferences(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("app/preferences")

	prefs := Preferences{
		Dht: true, // Allow magnets without trackers
	}

	h.respondJSON(w, http.StatusOK, prefs)
}

// Categories handles /api/v2/torrents/categories
func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("torrents/categories")

	categories := map[string]Category{
		h.config.QBittorrent.CategoryName: {
			Name:     h.config.QBittorrent.CategoryName,
			SavePath: h.config.QBittorrent.SavePath,
		},
	}

	h.respondJSON(w, http.StatusOK, categories)
}

// Add handles /api/v2/torrents/add
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Info("torrents/add")

	// Log all request details for debugging
	h.logger.Debug("request details",
		zap.String("method", r.Method),
		zap.String("content_type", r.Header.Get("Content-Type")),
		zap.String("user_agent", r.Header.Get("User-Agent")))

	// Parse form based on Content-Type
	if err := h.parseForm(r); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	// Log all form values
	if r.Form != nil && len(r.Form) > 0 {
		h.logger.Debug("form data received")
		for key, values := range r.Form {
			for _, value := range values {
				// Don't log the full URL for magnets, just a snippet
				logValue := value
				if key == "urls" && len(value) > 100 {
					logValue = value[:100] + "..."
				}
				h.logger.Debug("form field",
					zap.String("key", key),
					zap.String("value", logValue))
			}
		}
	}

	// Parse auth to get Servarr credentials for async validation
	servarrHost, servarrAPIKey, err := ParseAuthHeader(r)
	if err != nil {
		h.logger.Error("failed to parse auth header", zap.Error(err))
		h.respondError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	// Extract URLs
	urls := h.extractURLs(r)

	// Extract files
	files := h.extractFiles(r)

	if len(urls) == 0 && len(files) == 0 {
		h.respondError(w, http.StatusBadRequest, "no torrents provided")
		return
	}

	// Extract category for async processing
	category := r.FormValue("category")

	// Process torrents from URLs (magnets) - async
	for _, magnetURL := range urls {
		// Extract hash from magnet IMMEDIATELY
		hash := extractInfoHashFromMagnet(magnetURL)
		if hash == "" {
			h.logger.Error("failed to extract hash from magnet", zap.String("magnet", magnetURL[:100]+"..."))
			h.respondError(w, http.StatusBadRequest, "invalid magnet URL")
			return
		}

		// Create processing torrent entry BEFORE doing anything else
		processing := &ProcessingTorrent{
			Hash:          hash,
			MagnetURL:     magnetURL,
			Category:      category,
			AddedAt:       time.Now(),
			Status:        "checkingUP", // Show as checking/validating
			ServarrHost:   servarrHost,
			ServarrAPIKey: servarrAPIKey,
		}

		// Add to processingTorrents so it shows up in /torrents/info
		h.processingTorrents.Add(hash, processing)
		h.logger.Info("torrent marked as processing, starting async validation",
			zap.String("hash", hash))

		// Spawn goroutine for async processing
		go h.processTorrentAsync(magnetURL, category, servarrHost, servarrAPIKey)
	}

	// Process torrents from files - keep synchronous for now (less common)
	for _, file := range files {
		torrentID, err := h.debridClient.AddTorrentByFile(ctx, file)
		if err != nil {
			h.logger.Error("failed to add torrent by file", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add torrent: %v", err))
			return
		}

		// Get torrent info first (needed for validation)
		torrentInfo, err := h.torrentService.GetInfo(ctx, torrentID)
		if err != nil {
			h.logger.Error("failed to get torrent info", zap.String("torrent_id", torrentID), zap.Error(err))
			// Delete the torrent since we can't get its info
			if delErr := h.debridClient.DeleteTorrent(ctx, torrentID); delErr != nil {
				h.logger.Error("failed to delete torrent", zap.String("torrent_id", torrentID), zap.Error(delErr))
			}
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get torrent info: %v", err))
			return
		}

		// Validate expected file count first if enabled (cheap - just counts files)
		if h.config.MediaValidation.ValidateFileCount {
			if err := h.validateExpectedFileCount(ctx, r, torrentInfo); err != nil {
				h.logger.Error("file count validation failed", zap.String("torrent_id", torrentID), zap.Error(err))
				// Delete the torrent since it failed validation
				if delErr := h.debridClient.DeleteTorrent(ctx, torrentID); delErr != nil {
					h.logger.Error("failed to delete invalid torrent", zap.String("torrent_id", torrentID), zap.Error(delErr))
				}
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("file count validation failed: %v", err))
				return
			}
		}

		// Now validate media quality if enabled (expensive - ffprobe on each file)
		if h.config.MediaValidation.Enabled {
			torrentInfo, err = h.torrentService.AddAndValidate(ctx, torrentID, torrentInfo)
			if err != nil {
				h.logger.Error("media validation failed", zap.String("torrent_id", torrentID), zap.Error(err))
				// Delete the torrent since it failed validation
				if delErr := h.debridClient.DeleteTorrent(ctx, torrentID); delErr != nil {
					h.logger.Error("failed to delete invalid torrent", zap.String("torrent_id", torrentID), zap.Error(delErr))
				}
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("media validation failed: %v", err))
				return
			}
		}
	}

	// Clear cache after adding
	h.cache.Clear()

	// Return 200 OK immediately - this triggers Sonarr to create history entry and start polling
	h.respondText(w, http.StatusOK, "Ok.")
}

// Info handles /api/v2/torrents/info
func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Debug("torrents/info")

	// Try cache first
	cacheKey := cache.KeyBuilder{}.FromRequest(r.Method, r.URL.Path, map[string]string{
		"auth": r.Header.Get("Authorization"),
	})

	var cachedResult []TorrentInfo
	if ok, _ := h.cache.GetJSON(cacheKey, &cachedResult); ok {
		h.logger.Debug("returning cached torrents info")

		// Make a copy of the cached result to avoid modifying the cached data
		result := make([]TorrentInfo, len(cachedResult))
		copy(result, cachedResult)

		// Cache hole punching: Re-validate paths for missingFiles entries FIRST
		// This allows status to update quickly when files appear without full cache invalidation
		needsCacheUpdate := false
		if h.config.QBittorrent.ValidatePaths {
			for i := range result {
				if result[i].State == "missingFiles" {
					// Extract torrent hash from the Hash field
					torrentHash := result[i].Hash
					if ValidatePath(h.config.QBittorrent.SavePath, torrentHash) {
						// Path now exists! Update status to pausedUP
						result[i].State = "pausedUP"
						needsCacheUpdate = true
						h.logger.Debug("cache hole punch: path now exists",
							zap.String("torrent_hash", torrentHash),
							zap.String("name", result[i].Name))
					}
				}
			}
		}

		// Override state for torrents that are currently being processed
		for i := range result {
			if processing, exists := h.processingTorrents.Get(result[i].Hash); exists {
				// Check if validation failed
				if processing.Failed {
					h.logger.Debug("torrent validation failed, setting state to error",
						zap.String("hash", result[i].Hash),
						zap.String("name", result[i].Name),
						zap.String("error", processing.ErrorMessage))
					result[i].State = "error" // Show as error/failed - triggers Sonarr re-search
					// Prepend error message to torrent name for visibility
					if processing.ErrorMessage != "" {
						result[i].Name = fmt.Sprintf("[FAILED] %s: %s", result[i].Name, processing.ErrorMessage)
					}
				} else {
					h.logger.Debug("torrent is being processed, setting state to checkingUP",
						zap.String("hash", result[i].Hash),
						zap.String("name", result[i].Name))
					result[i].State = "checkingUP" // Show as checking/queued during validation
				}
			}
		}

		// Add processing torrents that aren't in Real-Debrid yet (very early in the async process)
		processingTorrents := h.processingTorrents.GetAll()
		for _, processing := range processingTorrents {
			// Check if this torrent is already in the result
			found := false
			for _, existing := range result {
				if strings.EqualFold(existing.Hash, processing.Hash) {
					found = true
					break
				}
			}

			// If not found in Real-Debrid results, add it as a processing entry
			if !found && processing.TorrentInfo == nil {
				h.logger.Debug("adding processing torrent to info response",
					zap.String("hash", processing.Hash))

				// Determine state and name based on failed status
				state := "checkingUP"
				name := "Validating..."
				if processing.Failed {
					state = "error"
					name = fmt.Sprintf("[FAILED] Validation failed: %s", processing.ErrorMessage)
				}

				result = append(result, TorrentInfo{
					Hash:     processing.Hash,
					Name:     name,
					State:    state,
					Category: processing.Category,
					SavePath: h.config.QBittorrent.SavePath,
					AddedOn:  processing.AddedAt.Unix(),
				})
			}
		}

		// If we updated the cache data (path validation), save the ORIGINAL result back
		// (before we added processing torrents)
		if needsCacheUpdate {
			// Save only the base torrents, not the ones we appended from processing
			baseResult := make([]TorrentInfo, len(cachedResult))
			copy(baseResult, cachedResult)
			// Apply only the path validation updates to the base result
			for i := range baseResult {
				if baseResult[i].State == "missingFiles" {
					torrentHash := baseResult[i].Hash
					if ValidatePath(h.config.QBittorrent.SavePath, torrentHash) {
						baseResult[i].State = "pausedUP"
					}
				}
			}
			h.cache.SetJSON(cacheKey, baseResult, 15*time.Minute)
		}

		// Ensure we return an empty array instead of null if result is nil
		if result == nil {
			result = make([]TorrentInfo, 0)
		}
		h.respondJSON(w, http.StatusOK, result)
		return
	}

	// Parse auth to get Servarr credentials
	servarrHost, servarrAPIKey, err := ParseAuthHeader(r)
	if err != nil {
		h.logger.Error("failed to parse auth header", zap.Error(err))
		h.respondError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	// Get all torrents from Real-Debrid
	torrents, err := h.debridClient.GetTorrents(ctx)
	if err != nil {
		h.logger.Error("failed to get torrents", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get torrents: %v", err))
		return
	}

	h.logger.Debug("real-debrid torrents", zap.Int("count", len(*torrents)))

	// Get Servarr history to filter
	servarrHistory, err := h.servarrClient.GetHistory(ctx, servarrHost, servarrAPIKey)
	if err != nil {
		h.logger.Error("failed to get servarr history", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get history: %v", err))
		return
	}

	h.logger.Debug("servarr history", zap.Int("count", len(servarrHistory)))

	// Build hash map for quick lookup from Servarr history
	// Note: Servarr records the actual torrent hash (infohash), not Real-Debrid's ID
	historyHashes := make(map[string]bool)

	// Add from Servarr history (this is the source of truth)
	for _, record := range servarrHistory {
		// Skip records with empty downloadId
		if record.DownloadID == "" {
			continue
		}
		h.logger.Debug("history record", zap.String("downloadId", record.DownloadID))
		historyHashes[strings.ToLower(record.DownloadID)] = true
	}

	h.logger.Debug("history hashes to match", zap.Int("count", len(historyHashes)))

	// Filter torrents to only those in Servarr history (like the old implementation)
	torrentInfos := make([]TorrentInfo, 0) // Initialize to empty slice, not nil
	for _, torrent := range *torrents {
		// Skip torrents with empty ID (shouldn't happen but defensive check)
		if torrent.ID == "" {
			h.logger.Warn("skipping torrent with empty ID", zap.String("filename", torrent.Filename))
			continue
		}

		// Only include torrents that are in Servarr history
		// Compare against torrent.Hash (actual infohash), not torrent.ID (Real-Debrid ID)
		// This matches the old implementation behavior
		inHistory := historyHashes[strings.ToLower(torrent.Hash)] || historyHashes[strings.ToLower(torrent.ID)]
		if !inHistory {
			continue
		}

		h.logger.Debug("torrent in history, including", zap.String("id", torrent.ID))

		// Map Real-Debrid status to qBittorrent state
		state := debrid.MapStatus(torrent.Status)

		// Override state for torrents that are currently being processed
		// Use torrent hash for checking, fallback to ID if hash is empty
		hashToCheck := torrent.Hash
		if hashToCheck == "" {
			hashToCheck = torrent.ID
		}
		if processing, exists := h.processingTorrents.Get(hashToCheck); exists {
			// Check if validation failed
			if processing.Failed {
				h.logger.Debug("torrent validation failed, setting state to error",
					zap.String("id", torrent.ID),
					zap.String("hash", torrent.Hash),
					zap.String("error", processing.ErrorMessage))
				state = "error" // Show as error/failed - triggers Sonarr re-search
			} else {
				h.logger.Debug("torrent is being processed, setting state to checkingUP",
					zap.String("id", torrent.ID),
					zap.String("hash", torrent.Hash))
				state = "checkingUP" // Show as checking/queued during validation
			}
		}

		// Validate path if enabled
		if h.config.QBittorrent.ValidatePaths && state == "pausedUP" {
			// Use torrent hash for path validation, fallback to ID if hash is empty
			hashOrID := torrent.Hash
			if hashOrID == "" {
				hashOrID = torrent.ID
			}
			if !ValidatePath(h.config.QBittorrent.SavePath, hashOrID) {
				state = "missingFiles"
			}
		}

		info := ConvertRealDebridToTorrentInfo(torrent, &h.config.QBittorrent, state)
		torrentInfos = append(torrentInfos, info)
	}

	h.logger.Debug("base torrents from Real-Debrid", zap.Int("count", len(torrentInfos)))

	// Cache the BASE result (without processing torrents) with longer TTL
	// Cache hole punching handles missingFiles status updates
	// Processing torrents are added dynamically on each request
	h.cache.SetJSON(cacheKey, torrentInfos, 15*time.Minute)

	// Add processing torrents that aren't in Real-Debrid yet (very early in the async process)
	// This is done AFTER caching so processing torrents aren't persisted to cache
	processingTorrents := h.processingTorrents.GetAll()
	for _, processing := range processingTorrents {
		// Check if this torrent is already in the result
		found := false
		for _, existing := range torrentInfos {
			if strings.EqualFold(existing.Hash, processing.Hash) {
				found = true
				break
			}
		}

		// If not found in Real-Debrid results, add it as a processing entry
		// This happens when the torrent was just accepted but hasn't been added to RD yet
		if !found && processing.TorrentInfo == nil {
			h.logger.Debug("adding processing torrent to info response",
				zap.String("hash", processing.Hash))
			torrentInfos = append(torrentInfos, TorrentInfo{
				Hash:     processing.Hash,
				Name:     "Validating...",
				State:    "checkingUP",
				Category: processing.Category,
				SavePath: h.config.QBittorrent.SavePath,
				AddedOn:  processing.AddedAt.Unix(),
			})
		}
	}

	h.logger.Debug("torrents to return (with processing)", zap.Int("count", len(torrentInfos)))

	h.respondJSON(w, http.StatusOK, torrentInfos)
}

// Properties handles /api/v2/torrents/properties
func (h *Handler) Properties(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hash := r.URL.Query().Get("hash")

	h.logger.Debug("torrents/properties", zap.String("hash", hash))

	if hash == "" {
		h.respondError(w, http.StatusBadRequest, "hash parameter required")
		return
	}

	// Check if this is a processing torrent (not yet in Real-Debrid or still validating)
	if processing, exists := h.processingTorrents.Get(hash); exists {
		h.logger.Debug("torrent is processing, returning placeholder properties",
			zap.String("hash", hash))

		// Return placeholder properties for processing torrents
		props := TorrentProperties{
			SavePath:   h.config.QBittorrent.SavePath,
			AddedOn:    processing.AddedAt.Unix(),
			TotalSize:  0,
			ShareRatio: 0,
			PieceSize:  16384,
		}
		h.respondJSON(w, http.StatusOK, props)
		return
	}

	// Try cache
	cacheKey := cache.KeyBuilder{}.FromString("properties", hash)
	var result TorrentProperties
	if ok, _ := h.cache.GetJSON(cacheKey, &result); ok {
		h.respondJSON(w, http.StatusOK, result)
		return
	}

	// Get torrent info
	torrentInfo, err := h.debridClient.GetTorrentInfoByHash(ctx, hash)
	if err != nil {
		h.logger.Error("failed to get torrent info", zap.String("hash", hash), zap.Error(err))
		h.respondError(w, http.StatusNotFound, "torrent not found")
		return
	}

	// Convert to qBittorrent format
	props := ConvertTorrentInfoToProperties(torrentInfo, &h.config.QBittorrent)

	// Cache result
	h.cache.SetJSON(cacheKey, props, 5*time.Minute)

	h.respondJSON(w, http.StatusOK, props)
}

// Files handles /api/v2/torrents/files
func (h *Handler) Files(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hash := r.URL.Query().Get("hash")

	h.logger.Debug("torrents/files", zap.String("hash", hash))

	if hash == "" {
		h.respondError(w, http.StatusBadRequest, "hash parameter required")
		return
	}

	// Check if this is a processing torrent (not yet in Real-Debrid or still validating)
	if _, exists := h.processingTorrents.Get(hash); exists {
		h.logger.Debug("torrent is processing, returning empty files list",
			zap.String("hash", hash))

		// Return empty files list for processing torrents
		h.respondJSON(w, http.StatusOK, []TorrentFile{})
		return
	}

	// Try cache
	cacheKey := cache.KeyBuilder{}.FromString("files", hash)
	var result []TorrentFile
	if ok, _ := h.cache.GetJSON(cacheKey, &result); ok {
		h.respondJSON(w, http.StatusOK, result)
		return
	}

	// Get torrent info
	torrentInfo, err := h.debridClient.GetTorrentInfoByHash(ctx, hash)
	if err != nil {
		h.logger.Error("failed to get torrent info", zap.String("hash", hash), zap.Error(err))
		h.respondError(w, http.StatusNotFound, "torrent not found")
		return
	}

	// Convert files
	files := ConvertRealDebridFiles(torrentInfo)

	// Cache result
	h.cache.SetJSON(cacheKey, files, 10*time.Minute)

	h.respondJSON(w, http.StatusOK, files)
}

// Delete handles /api/v2/torrents/delete
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	hashes := r.FormValue("hashes")
	h.logger.Info("torrents/delete", zap.String("hashes", hashes))

	if hashes == "" {
		h.respondError(w, http.StatusBadRequest, "hashes parameter required")
		return
	}

	// Split hashes by pipe
	hashList := strings.Split(hashes, "|")

	// Delete each torrent
	for _, hash := range hashList {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}

		if err := h.debridClient.DeleteTorrentByHash(ctx, hash); err != nil {
			// If torrent is already deleted (not found), log as info and continue
			if errors.Is(err, debrid.ErrTorrentNotFound) {
				h.logger.Info("torrent already deleted or not found", zap.String("hash", hash))
				continue
			}
			h.logger.Error("failed to delete torrent", zap.String("hash", hash), zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete torrent: %v", err))
			return
		}
	}

	// Clear cache after deletion
	h.cache.Clear()

	h.respondText(w, http.StatusOK, "Ok.")
}

// parseForm parses the request form based on Content-Type
func (h *Handler) parseForm(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")

	// Extract the base content type (before any parameters like boundary)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	switch contentType {
	case "multipart/form-data":
		return r.ParseMultipartForm(32 << 20) // 32 MB max
	case "application/x-www-form-urlencoded", "":
		return r.ParseForm()
	default:
		return fmt.Errorf("unsupported Content-Type: %s", contentType)
	}
}

// extractURLs extracts torrent URLs from the request
func (h *Handler) extractURLs(r *http.Request) []string {
	var urls []string

	urlsParam := r.FormValue("urls")
	if urlsParam != "" {
		lines := strings.Split(urlsParam, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				urls = append(urls, line)
			}
		}
	}

	return urls
}

// extractFiles extracts torrent files from the request
func (h *Handler) extractFiles(r *http.Request) []io.ReadCloser {
	var files []io.ReadCloser

	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		fileHeaders := r.MultipartForm.File["torrents"]
		for _, fileHeader := range fileHeaders {
			file, err := h.openFile(fileHeader)
			if err != nil {
				h.logger.Warn("failed to open uploaded file",
					zap.String("filename", fileHeader.Filename),
					zap.Error(err))
				continue
			}
			files = append(files, file)
		}
	}

	return files
}

// openFile opens a multipart file
func (h *Handler) openFile(fileHeader *multipart.FileHeader) (io.ReadCloser, error) {
	return fileHeader.Open()
}

// validateExpectedFileCount validates that a torrent has the expected number of files from *arr
func (h *Handler) validateExpectedFileCount(ctx context.Context, r *http.Request, torrentInfo *api.TorrentInfo) error {
	// Parse auth to get Servarr credentials
	servarrHost, servarrAPIKey, err := ParseAuthHeader(r)
	if err != nil {
		h.logger.Warn("skipping file count validation: failed to parse auth header", zap.Error(err))
		return nil // Don't fail if we can't parse auth - maybe it's not from *arr
	}

	// Use torrent hash as downloadId
	downloadID := torrentInfo.Hash
	if downloadID == "" {
		h.logger.Warn("skipping file count validation: torrent has no hash", zap.String("torrent_id", torrentInfo.ID))
		return nil
	}

	h.logger.Debug("validating file count",
		zap.String("torrent_id", torrentInfo.ID),
		zap.String("downloadId", downloadID))

	// Trigger Sonarr/Radarr to refresh the monitored downloads queue
	// This ensures the queue has fresh data before we query it
	h.logger.Debug("triggering RefreshMonitoredDownloads before validation",
		zap.String("downloadId", downloadID))
	if err := h.servarrClient.RefreshMonitoredDownloadsAndWait(ctx, servarrHost, servarrAPIKey, 15); err != nil {
		h.logger.Warn("failed to wait for RefreshMonitoredDownloads completion", zap.Error(err))
		// Don't fail - continue with validation even if refresh times out
	}

	// Query *arr queue API - this is populated as soon as Servarr sends the torrent
	queueRecords, err := h.servarrClient.GetQueueByDownloadID(ctx, servarrHost, servarrAPIKey, downloadID)
	if err != nil {
		return fmt.Errorf("failed to query servarr queue: %w", err)
	}

	if len(queueRecords) == 0 {
		return fmt.Errorf("no queue records found for download %s - torrent was not requested by Servarr", downloadID)
	}

	h.logger.Debug("found records in queue",
		zap.Int("queue_records", len(queueRecords)),
		zap.String("downloadId", downloadID))

	// Use queue records to determine expected file count and validate
	return h.validateFromQueueRecords(ctx, torrentInfo, queueRecords)
}

// validateFromQueueRecords validates file count using queue records
func (h *Handler) validateFromQueueRecords(ctx context.Context, torrentInfo *api.TorrentInfo, queueRecords []servarr.QueueRecord) error {
	var expectedCount int

	// Check if this is Sonarr (has seriesId) or Radarr (has movieId)
	if queueRecords[0].MovieID != nil {
		// Radarr - always 1 movie
		expectedCount = 1
		h.logger.Info("validating Radarr file count from queue",
			zap.String("torrent_id", torrentInfo.ID),
			zap.Int("expected_movies", expectedCount))
	} else if queueRecords[0].SeriesID != nil {
		// Sonarr - count queue records
		// Each queue record represents one episode
		// This is MORE RELIABLE than counting episodeIds which may be empty/unpopulated
		expectedCount = len(queueRecords)
		h.logger.Info("validating Sonarr file count from queue (counting records)",
			zap.String("torrent_id", torrentInfo.ID),
			zap.Int("expected_episodes", expectedCount),
			zap.Int("queue_records", len(queueRecords)))
	} else {
		h.logger.Warn("queue record has no movieId or seriesId, skipping validation",
			zap.String("torrent_id", torrentInfo.ID))
		return nil
	}

	// Validate file count
	return h.torrentService.ValidateFileCount(ctx, torrentInfo, expectedCount)
}

// extractInfoHashFromMagnet extracts the infohash from a magnet URL
func extractInfoHashFromMagnet(magnetURL string) string {
	// magnet:?xt=urn:btih:HASH
	if !strings.HasPrefix(magnetURL, "magnet:?xt=urn:btih:") {
		return ""
	}
	hash := strings.TrimPrefix(magnetURL, "magnet:?xt=urn:btih:")
	// Remove any additional parameters after the hash
	if idx := strings.Index(hash, "&"); idx != -1 {
		hash = hash[:idx]
	}
	return strings.ToLower(hash)
}

// processTorrentAsync asynchronously handles adding and validating a torrent
func (h *Handler) processTorrentAsync(magnetURL, category string, servarrHost, servarrAPIKey string) {
	ctx := context.Background()

	// Extract hash from magnet
	hash := extractInfoHashFromMagnet(magnetURL)
	if hash == "" {
		h.logger.Error("failed to extract hash from magnet", zap.String("magnet", magnetURL))
		h.processingTorrents.Remove(hash)
		return
	}

	h.logger.Info("starting async torrent processing",
		zap.String("hash", hash),
		zap.String("category", category))

	// Add to Real-Debrid
	h.logger.Info("adding torrent to Real-Debrid", zap.String("hash", hash))
	torrentID, err := h.debridClient.AddTorrentByURL(ctx, magnetURL)
	if err != nil {
		h.logger.Error("failed to add torrent to Real-Debrid", zap.String("hash", hash), zap.Error(err))
		h.processingTorrents.MarkFailed(hash, fmt.Sprintf("Real-Debrid add failed: %v", err))
		return
	}
	h.logger.Info("torrent added to Real-Debrid",
		zap.String("hash", hash),
		zap.String("torrent_id", torrentID))

	// Get torrent info
	h.logger.Info("fetching torrent info from Real-Debrid", zap.String("torrent_id", torrentID))
	torrentInfo, err := h.torrentService.GetInfo(ctx, torrentID)
	if err != nil {
		h.logger.Error("failed to get torrent info", zap.String("torrent_id", torrentID), zap.Error(err))
		// Don't delete yet - mark as failed so Sonarr can detect it
		h.processingTorrents.MarkFailed(hash, fmt.Sprintf("Failed to get torrent info: %v", err))
		return
	}

	// Update processing torrent with Real-Debrid info and torrent ID
	if processing, exists := h.processingTorrents.Get(hash); exists {
		processing.TorrentID = torrentID
		h.processingTorrents.UpdateTorrentInfo(hash, torrentInfo)
	}

	// Trigger Sonarr/Radarr to refresh the monitored downloads queue
	// This tells Sonarr to update its queue with the new download
	h.logger.Info("triggering RefreshMonitoredDownloads",
		zap.String("hash", hash),
		zap.String("servarr_host", servarrHost))
	if err := h.servarrClient.RefreshMonitoredDownloadsAndWait(ctx, servarrHost, servarrAPIKey, 15); err != nil {
		h.logger.Warn("failed to wait for RefreshMonitoredDownloads completion", zap.Error(err))
		// Don't fail - continue with validation even if refresh times out
	} else {
		h.logger.Info("RefreshMonitoredDownloads completed successfully", zap.String("hash", hash))
	}

	// Wait a brief moment for the queue to be fully updated
	h.logger.Info("waiting for queue to be fully updated", zap.String("hash", hash))
	time.Sleep(2 * time.Second)

	// Validate file count if enabled
	if h.config.MediaValidation.ValidateFileCount {
		h.logger.Info("starting file count validation", zap.String("hash", hash))
		if err := h.validateFileCountFromQueue(ctx, torrentInfo, servarrHost, servarrAPIKey); err != nil {
			h.logger.Error("file count validation failed", zap.String("torrent_id", torrentID), zap.Error(err))
			// Mark as failed instead of deleting - Sonarr needs to see the failure
			h.processingTorrents.MarkFailed(hash, fmt.Sprintf("File count validation failed: %v", err))
			return
		}
		h.logger.Info("file count validation passed", zap.String("hash", hash))
	} else {
		h.logger.Info("file count validation disabled, skipping", zap.String("hash", hash))
	}

	// Validate media quality if enabled
	if h.config.MediaValidation.Enabled {
		h.logger.Info("starting media quality validation", zap.String("hash", hash))
		torrentInfo, err = h.torrentService.AddAndValidate(ctx, torrentID, torrentInfo)
		if err != nil {
			h.logger.Error("media validation failed", zap.String("torrent_id", torrentID), zap.Error(err))
			// Mark as failed instead of deleting - Sonarr needs to see the failure
			h.processingTorrents.MarkFailed(hash, fmt.Sprintf("Media validation failed: %v", err))
			return
		}
		// Update torrent info after validation
		h.processingTorrents.UpdateTorrentInfo(hash, torrentInfo)
		h.logger.Info("media quality validation passed", zap.String("hash", hash))
	} else {
		h.logger.Info("media quality validation disabled, skipping", zap.String("hash", hash))
	}

	// Validation completed successfully!
	h.logger.Info("torrent validation completed successfully", zap.String("hash", hash))

	// Trigger RefreshMonitoredDownloads after successful validation
	h.logger.Info("triggering RefreshMonitoredDownloads after successful validation", zap.String("hash", hash))
	if err := h.servarrClient.RefreshMonitoredDownloadsAndWait(ctx, servarrHost, servarrAPIKey, 15); err != nil {
		h.logger.Warn("failed to wait for RefreshMonitoredDownloads completion after validation", zap.Error(err))
	}

	// Clear cache so Info() picks up the new torrent from Real-Debrid
	h.cache.Clear()

	// Remove from processing immediately - torrent is now in Real-Debrid with proper status
	// Sonarr will now see the actual Real-Debrid status (e.g., pausedUP = Completed)
	h.processingTorrents.Remove(hash)
	h.logger.Info("removed torrent from processing, now using Real-Debrid status", zap.String("hash", hash))
}

// validateFileCountFromQueue validates file count using Sonarr/Radarr queue API
func (h *Handler) validateFileCountFromQueue(ctx context.Context, torrentInfo *api.TorrentInfo, servarrHost, servarrAPIKey string) error {
	downloadID := torrentInfo.Hash
	if downloadID == "" {
		h.logger.Warn("skipping file count validation: torrent has no hash", zap.String("torrent_id", torrentInfo.ID))
		return nil
	}

	h.logger.Debug("validating file count from queue",
		zap.String("torrent_id", torrentInfo.ID),
		zap.String("downloadId", downloadID))

	// Query *arr queue API
	queueRecords, err := h.servarrClient.GetQueueByDownloadID(ctx, servarrHost, servarrAPIKey, downloadID)
	if err != nil {
		return fmt.Errorf("failed to query servarr queue: %w", err)
	}

	if len(queueRecords) == 0 {
		return fmt.Errorf("no queue records found for download %s - torrent was not requested by Servarr", downloadID)
	}

	h.logger.Debug("found records in queue",
		zap.Int("queue_records", len(queueRecords)),
		zap.String("downloadId", downloadID))

	// Use queue records to determine expected file count
	return h.validateFromQueueRecords(ctx, torrentInfo, queueRecords)
}

// cleanupFailedTorrents runs a background job to clean up failed torrents after 5 minutes
func (h *Handler) cleanupFailedTorrents() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	const failedRetentionPeriod = 5 * time.Minute

	h.logger.Info("started background cleanup job for failed torrents")

	for {
		select {
		case <-ticker.C:
			failedTorrents := h.processingTorrents.GetFailedTorrents()

			for _, failed := range failedTorrents {
				age := time.Since(failed.FailedAt)

				if age >= failedRetentionPeriod {
					h.logger.Info("cleaning up expired failed torrent",
						zap.String("hash", failed.Hash),
						zap.String("error", failed.ErrorMessage),
						zap.Duration("age", age))

					// Delete from Real-Debrid if present
					if failed.TorrentID != "" {
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						if err := h.debridClient.DeleteTorrent(ctx, failed.TorrentID); err != nil {
							h.logger.Warn("failed to delete expired torrent from Real-Debrid",
								zap.String("torrent_id", failed.TorrentID),
								zap.String("hash", failed.Hash),
								zap.Error(err))
						} else {
							h.logger.Info("deleted expired failed torrent from Real-Debrid",
								zap.String("torrent_id", failed.TorrentID),
								zap.String("hash", failed.Hash))
						}
						cancel()
					}

					// Remove from processing queue
					h.processingTorrents.Remove(failed.Hash)

					// Clear cache to update torrent list
					h.cache.Clear()
				} else {
					h.logger.Debug("failed torrent still in retention period",
						zap.String("hash", failed.Hash),
						zap.Duration("age", age),
						zap.Duration("remaining", failedRetentionPeriod-age))
				}
			}

		case <-h.shutdownCh:
			h.logger.Info("stopping cleanup job")
			return
		}
	}
}

// Shutdown gracefully stops the handler
func (h *Handler) Shutdown() {
	h.logger.Info("shutting down handler")
	close(h.shutdownCh)
}
