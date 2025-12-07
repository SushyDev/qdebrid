package debrid

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"qdebrid/internal/config"
	"qdebrid/pkg/retry"
	"strconv"
	"strings"

	real_debrid "github.com/sushydev/real_debrid_go"
	"github.com/sushydev/real_debrid_go/api"
	"go.uber.org/zap"
)

// Client wraps the Real-Debrid client with rate limiting and retry logic
type Client struct {
	client *real_debrid.Client
	queue  *retry.Queue
	config *config.RealDebridConfig
	logger *zap.Logger
}

// NewClient creates a new Real-Debrid client with rate limiting and retry
func NewClient(cfg *config.RealDebridConfig, logger *zap.Logger) *Client {
	httpClient := &http.Client{
		Timeout: 30 * 1000000000, // 30 seconds
	}

	rdClient := real_debrid.NewClient(cfg.Token, httpClient)

	retryConfig := retry.Config{
		MaxRetries:     cfg.MaxRetries,
		InitialBackoff: 2 * 1000000000,      // 2 seconds
		MaxBackoff:     5 * 60 * 1000000000, // 5 minutes
		Multiplier:     2.0,
		Jitter:         true,
	}

	// Create queue with rate limiting
	// Use conservative burst of 3 to handle occasional spikes
	queue := retry.NewQueue(cfg.RequestsPerMinute, 3, retryConfig, logger)

	return &Client{
		client: rdClient,
		queue:  queue,
		config: cfg,
		logger: logger,
	}
}

// Shutdown gracefully shuts down the client
func (c *Client) Shutdown() {
	c.queue.Shutdown()
}

// AddTorrentByURL adds a torrent from a URL (magnet or http)
func (c *Client) AddTorrentByURL(ctx context.Context, url string) (string, error) {
	c.logger.Info("adding torrent by URL", zap.String("url_type", getURLType(url)))

	var torrentID string
	var addErr error

	err := c.queue.Submit(ctx, "add_torrent", func(ctx context.Context) error {
		if strings.HasPrefix(url, "magnet") {
			response, err := api.AddMagnet(c.client, url)
			if err != nil {
				return c.wrapHTTPError(err, "add_magnet")
			}
			torrentID = response.Id
			return nil
		}

		if strings.HasPrefix(url, "http") {
			file, err := c.fetchTorrentFile(ctx, url)
			if err != nil {
				return fmt.Errorf("fetch torrent file: %w", err)
			}
			defer file.Close()

			response, err := api.AddTorrent(c.client, file)
			if err != nil {
				return c.wrapHTTPError(err, "add_torrent")
			}
			torrentID = response.Id
			return nil
		}

		return fmt.Errorf("unsupported URL type: %s", url)
	})

	if err != nil {
		return "", err
	}

	// Select files after adding
	if err := c.SelectFiles(ctx, torrentID); err != nil {
		// Log but don't fail - file selection can be retried later
		c.logger.Warn("file selection failed after adding torrent",
			zap.String("torrent_id", torrentID),
			zap.Error(err))
		addErr = fmt.Errorf("torrent added but file selection failed: %w", err)
	}

	c.logger.Info("torrent added successfully", zap.String("torrent_id", torrentID))
	return torrentID, addErr
}

// AddTorrentByFile adds a torrent from a file
func (c *Client) AddTorrentByFile(ctx context.Context, file io.ReadCloser) (string, error) {
	c.logger.Info("adding torrent by file")

	var torrentID string
	var addErr error

	err := c.queue.Submit(ctx, "add_torrent_file", func(ctx context.Context) error {
		response, err := api.AddTorrent(c.client, file)
		if err != nil {
			return c.wrapHTTPError(err, "add_torrent")
		}
		torrentID = response.Id
		return nil
	})

	if err != nil {
		return "", err
	}

	// Select files after adding
	if err := c.SelectFiles(ctx, torrentID); err != nil {
		c.logger.Warn("file selection failed after adding torrent",
			zap.String("torrent_id", torrentID),
			zap.Error(err))
		addErr = fmt.Errorf("torrent added but file selection failed: %w", err)
	}

	c.logger.Info("torrent added successfully", zap.String("torrent_id", torrentID))
	return torrentID, addErr
}

// SelectFiles selects files for a torrent based on configuration
func (c *Client) SelectFiles(ctx context.Context, torrentID string) error {
	c.logger.Debug("selecting files for torrent", zap.String("torrent_id", torrentID))

	return c.queue.Submit(ctx, "select_files", func(ctx context.Context) error {
		// Get torrent info
		info, err := api.GetTorrentInfo(c.client, torrentID)
		if err != nil {
			return c.wrapHTTPError(err, "get_torrent_info")
		}

		// Filter files
		fileIDs := c.filterFiles(info.Files)
		if len(fileIDs) == 0 {
			return fmt.Errorf("no files match criteria (types: %v, min size: %d bytes)",
				c.config.AllowedFileTypes, c.config.MinFileSizeBytes)
		}

		c.logger.Debug("selecting files",
			zap.String("torrent_id", torrentID),
			zap.Strings("file_ids", fileIDs))

		// Select files
		fileIDsStr := strings.Join(fileIDs, ",")
		return c.wrapHTTPError(
			api.SelectFiles(c.client, torrentID, fileIDsStr),
			"select_files",
		)
	})
}

// GetTorrents retrieves all torrents
func (c *Client) GetTorrents(ctx context.Context) (*api.Torrents, error) {
	var torrents *api.Torrents

	err := c.queue.Submit(ctx, "get_torrents", func(ctx context.Context) error {
		result, err := api.GetTorrents(c.client, 1000, 1)
		if err != nil {
			return c.wrapHTTPError(err, "get_torrents")
		}
		torrents = result
		return nil
	})

	return torrents, err
}

// GetTorrentInfo retrieves info for a specific torrent
func (c *Client) GetTorrentInfo(ctx context.Context, torrentID string) (*api.TorrentInfo, error) {
	var info *api.TorrentInfo

	err := c.queue.Submit(ctx, "get_torrent_info", func(ctx context.Context) error {
		result, err := api.GetTorrentInfo(c.client, torrentID)
		if err != nil {
			return c.wrapHTTPError(err, "get_torrent_info")
		}
		info = result
		return nil
	})

	return info, err
}

// GetTorrentInfoByHash retrieves info by torrent hash
func (c *Client) GetTorrentInfoByHash(ctx context.Context, hash string) (*api.TorrentInfo, error) {
	torrents, err := c.GetTorrents(ctx)
	if err != nil {
		return nil, err
	}

	torrentID := findTorrentIDByHash(torrents, hash)
	if torrentID == "" {
		return nil, fmt.Errorf("torrent not found with hash: %s", hash)
	}

	return c.GetTorrentInfo(ctx, torrentID)
}

// DeleteTorrent deletes a torrent by ID
func (c *Client) DeleteTorrent(ctx context.Context, torrentID string) error {
	c.logger.Info("deleting torrent", zap.String("torrent_id", torrentID))

	return c.queue.Submit(ctx, "delete_torrent", func(ctx context.Context) error {
		return c.wrapHTTPError(
			api.Delete(c.client, torrentID),
			"delete",
		)
	})
}

// DeleteTorrentByHash deletes a torrent by hash
func (c *Client) DeleteTorrentByHash(ctx context.Context, hash string) error {
	torrents, err := c.GetTorrents(ctx)
	if err != nil {
		return err
	}

	torrentID := findTorrentIDByHash(torrents, hash)
	if torrentID == "" {
		return fmt.Errorf("torrent not found with hash: %s", hash)
	}

	return c.DeleteTorrent(ctx, torrentID)
}

// filterFiles filters files based on configuration
func (c *Client) filterFiles(files []api.TorrentFile) []string {
	if len(c.config.AllowedFileTypes) == 0 {
		// No filter, select all files
		return []string{"all"}
	}

	var fileIDs []string
	for _, file := range files {
		// Check minimum size
		if int64(file.Bytes) < c.config.MinFileSizeBytes {
			continue
		}

		// Check file extension
		for _, ext := range c.config.AllowedFileTypes {
			if strings.HasSuffix(strings.ToLower(file.Path), "."+strings.ToLower(ext)) {
				fileIDs = append(fileIDs, strconv.Itoa(file.ID))
				break
			}
		}
	}

	return fileIDs
}

// fetchTorrentFile downloads a torrent file from a URL
func (c *Client) fetchTorrentFile(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch torrent: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, retry.NewHTTPError(resp.StatusCode, "failed to fetch torrent file")
	}

	return resp.Body, nil
}

// wrapHTTPError wraps errors to provide retry hints
func (c *Client) wrapHTTPError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Try to extract status code from error message
	// The real_debrid_go library doesn't expose structured errors, so we parse the message
	errMsg := err.Error()

	if strings.Contains(errMsg, "429") || strings.Contains(strings.ToLower(errMsg), "rate limit") {
		return retry.NewHTTPError(http.StatusTooManyRequests, fmt.Sprintf("%s: %v", operation, err))
	}

	if strings.Contains(errMsg, "500") {
		return retry.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s: %v", operation, err))
	}

	if strings.Contains(errMsg, "503") {
		return retry.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("%s: %v", operation, err))
	}

	// Return as-is if we can't determine the status
	return fmt.Errorf("%s: %w", operation, err)
}

// findTorrentIDByHash finds a torrent ID by its hash
func findTorrentIDByHash(torrents *api.Torrents, hash string) string {
	if torrents == nil {
		return ""
	}

	for _, torrent := range *torrents {
		if strings.EqualFold(torrent.Hash, hash) {
			return torrent.ID
		}
	}

	return ""
}

// getURLType returns a friendly name for the URL type
func getURLType(url string) string {
	if strings.HasPrefix(url, "magnet") {
		return "magnet"
	}
	if strings.HasPrefix(url, "http") {
		return "http"
	}
	return "unknown"
}

// MapStatus maps Real-Debrid status to qBittorrent status
func MapStatus(status string) string {
	switch status {
	case "magnet_error":
		return "error"
	case "magnet_conversion":
		return "checkingUP"
	case "waiting_files_selection":
		return "checkingUP"
	case "queued":
		return "checkingUP"
	case "downloading":
		return "downloading"
	case "downloaded":
		return "pausedUP"
	case "error":
		return "error"
	case "virus":
		return "error"
	case "compressing":
		return "checkingUP"
	case "uploading":
		return "uploading"
	case "dead":
		return "error"
	default:
		return "unknown"
	}
}
