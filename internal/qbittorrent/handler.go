package qbittorrent

import (
	"context"
	"encoding/json"
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
	debridClient   *debrid.Client
	servarrClient  *servarr.Client
	torrentService *torrent.Service
	cache          *cache.Cache
	config         *config.Config
	logger         *zap.Logger
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

	return &Handler{
		debridClient:   debridClient,
		servarrClient:  servarrClient,
		torrentService: torrentService,
		cache:          cache,
		config:         cfg,
		logger:         logger,
	}
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

	// Log response at debug level (truncate if too large)
	if len(jsonData) > 1000 {
		h.logger.Debug("response json", zap.String("json", string(jsonData[:1000])+"... (truncated)"))
	} else {
		h.logger.Debug("response json", zap.String("json", string(jsonData)))
	}

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

	// Parse form based on Content-Type
	if err := h.parseForm(r); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
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

	// Add torrents from URLs
	for _, url := range urls {
		torrentID, err := h.debridClient.AddTorrentByURL(ctx, url)
		if err != nil {
			h.logger.Error("failed to add torrent by URL", zap.String("url", url), zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add torrent: %v", err))
			return
		}

		// Get torrent info first (needed for both validations)
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

	// Add torrents from files
	for _, file := range files {
		torrentID, err := h.debridClient.AddTorrentByFile(ctx, file)
		if err != nil {
			h.logger.Error("failed to add torrent by file", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add torrent: %v", err))
			return
		}

		// Get torrent info first (needed for both validations)
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

	// Clear cache after adding
	h.cache.Clear()

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

	var result []TorrentInfo
	if ok, _ := h.cache.GetJSON(cacheKey, &result); ok {
		h.logger.Debug("returning cached torrents info")

		// Cache hole punching: Re-validate paths for missingFiles entries
		// This allows status to update quickly when files appear without full cache invalidation
		if h.config.QBittorrent.ValidatePaths {
			needsUpdate := false
			for i := range result {
				if result[i].State == "missingFiles" {
					// Extract torrent hash from the Hash field
					torrentHash := result[i].Hash
					if ValidatePath(h.config.QBittorrent.SavePath, torrentHash) {
						// Path now exists! Update status to pausedUP
						result[i].State = "pausedUP"
						needsUpdate = true
						h.logger.Debug("cache hole punch: path now exists",
							zap.String("torrent_hash", torrentHash),
							zap.String("name", result[i].Name))
					}
				}
			}

			// If we updated any entries, save back to cache
			if needsUpdate {
				h.cache.SetJSON(cacheKey, result, 15*time.Minute)
			}
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

		h.logger.Debug("checking torrent",
			zap.String("id", torrent.ID),
			zap.String("hash", torrent.Hash),
			zap.String("filename", torrent.Filename))

		// Only include torrents that are in Servarr history
		// Compare against torrent.Hash (actual infohash), not torrent.ID (Real-Debrid ID)
		// This matches the old implementation behavior
		inHistory := historyHashes[strings.ToLower(torrent.Hash)] || historyHashes[strings.ToLower(torrent.ID)]
		if !inHistory {
			h.logger.Debug("torrent not in history, skipping",
				zap.String("id", torrent.ID),
				zap.String("hash", torrent.Hash))
			continue
		}

		h.logger.Debug("torrent in history, including", zap.String("id", torrent.ID))

		// Map Real-Debrid status to qBittorrent state
		state := debrid.MapStatus(torrent.Status)

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

	h.logger.Debug("torrents to return", zap.Int("count", len(torrentInfos)))

	// Cache the result with longer TTL
	// Cache hole punching handles missingFiles status updates
	h.cache.SetJSON(cacheKey, torrentInfos, 15*time.Minute)

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

	// Query *arr queue API for this download
	queueRecords, err := h.servarrClient.GetQueueByDownloadID(ctx, servarrHost, servarrAPIKey, downloadID)
	if err != nil {
		h.logger.Warn("skipping file count validation: failed to query queue",
			zap.Error(err),
			zap.String("downloadId", downloadID))
		return nil // Don't fail if queue query fails - service might be unreachable
	}

	if len(queueRecords) == 0 {
		h.logger.Debug("no queue records found for download", zap.String("downloadId", downloadID))
		return nil // No queue entry found - maybe it's not tracked yet
	}

	// Use the first queue record
	record := queueRecords[0]

	// Determine expected file count based on *arr type
	var expectedCount int
	if record.SeriesID != nil {
		// Sonarr - count episodes
		expectedCount = len(record.EpisodeIDs)
		h.logger.Info("validating Sonarr file count",
			zap.String("torrent_id", torrentInfo.ID),
			zap.String("downloadId", downloadID),
			zap.Int("expected_episodes", expectedCount))
	} else if record.MovieID != nil {
		// Radarr - always 1 movie
		expectedCount = 1
		h.logger.Info("validating Radarr file count",
			zap.String("torrent_id", torrentInfo.ID),
			zap.String("downloadId", downloadID),
			zap.Int("expected_movies", expectedCount))
	} else {
		h.logger.Debug("queue record has no seriesId or movieId, skipping validation",
			zap.String("downloadId", downloadID))
		return nil
	}

	// Validate file count
	return h.torrentService.ValidateFileCount(ctx, torrentInfo, expectedCount)
}
