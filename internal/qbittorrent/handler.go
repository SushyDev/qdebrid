package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"qdebrid/internal/cache"
	"qdebrid/internal/config"
	"qdebrid/internal/debrid"
	"qdebrid/internal/servarr"
)

// Handler handles qBittorrent API requests
type Handler struct {
	debridClient  *debrid.Client
	servarrClient *servarr.Client
	cache         *cache.Cache
	config        *config.Config
	logger        *zap.Logger
}

// NewHandler creates a new qBittorrent API handler
func NewHandler(
	debridClient *debrid.Client,
	servarrClient *servarr.Client,
	cache *cache.Cache,
	cfg *config.Config,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		debridClient:  debridClient,
		servarrClient: servarrClient,
		cache:         cache,
		config:        cfg,
		logger:        logger,
	}
}

// respondJSON writes JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.logger.Error("failed to encode response", zap.Error(err))
		}
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
		SavePath: h.config.QBittorrent.SavePath,
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

	// Parse form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB max
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
		if _, err := h.debridClient.AddTorrentByURL(ctx, url); err != nil {
			h.logger.Error("failed to add torrent by URL", zap.String("url", url), zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add torrent: %v", err))
			return
		}
	}

	// Add torrents from files
	for _, file := range files {
		if _, err := h.debridClient.AddTorrentByFile(ctx, file); err != nil {
			h.logger.Error("failed to add torrent by file", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add torrent: %v", err))
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

	// Get Servarr history to filter
	history, err := h.servarrClient.GetHistory(ctx, servarrHost, servarrAPIKey)
	if err != nil {
		h.logger.Error("failed to get servarr history", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get history: %v", err))
		return
	}

	// Build hash map for quick lookup
	historyHashes := make(map[string]bool)
	for _, record := range history {
		historyHashes[strings.ToLower(record.DownloadID)] = true
	}

	// Filter and convert torrents
	var torrentInfos []TorrentInfo
	for _, torrent := range *torrents {
		// Only include torrents that are in Servarr history
		if !historyHashes[strings.ToLower(torrent.Hash)] {
			continue
		}

		// Map Real-Debrid status to qBittorrent state
		state := debrid.MapStatus(torrent.Status)

		// Validate path if enabled
		if h.config.QBittorrent.ValidatePaths && state == "pausedUP" {
			if !ValidatePath(h.config.QBittorrent.SavePath, torrent.ID) {
				state = "missingFiles"
			}
		}

		info := ConvertRealDebridToTorrentInfo(torrent, &h.config.QBittorrent, state)
		torrentInfos = append(torrentInfos, info)
	}

	// Cache the result
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
	files := ConvertRealDebridFiles(torrentInfo.Files)

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
