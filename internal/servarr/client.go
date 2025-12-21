package servarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client handles communication with Servarr applications (*Arr)
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// HistoryRecord represents a grab event from Servarr history
type HistoryRecord struct {
	DownloadID string `json:"downloadId"`
	EventType  string `json:"eventType"`
	Date       string `json:"date"`
}

// QueueRecord represents a queue entry from Servarr
type QueueRecord struct {
	ID         int     `json:"id"`
	DownloadID string  `json:"downloadId"`
	Title      string  `json:"title"`
	Size       float64 `json:"size"`
	Status     string  `json:"status"`

	// Sonarr-specific fields
	SeriesID   *int      `json:"seriesId,omitempty"`
	EpisodeIDs []int     `json:"episodeIds,omitempty"`
	Episodes   []Episode `json:"episodes,omitempty"`

	// Radarr-specific fields
	MovieID *int `json:"movieId,omitempty"`
}

// Episode represents a Sonarr episode
type Episode struct {
	ID            int    `json:"id"`
	EpisodeNumber int    `json:"episodeNumber"`
	SeasonNumber  int    `json:"seasonNumber"`
	Title         string `json:"title"`
}

// QueueResponse represents the paginated queue response
type QueueResponse struct {
	Records []QueueRecord `json:"records"`
}

// NewClient creates a new Servarr client
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// GetHistory retrieves the grab history from a Servarr application
func (c *Client) GetHistory(ctx context.Context, baseURL string, apiKey string) ([]HistoryRecord, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL.Path = parsedURL.Path + "/api/v3/history/since"
	query := parsedURL.Query()
	query.Add("eventType", "grabbed")
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Api-Key", apiKey)

	c.logger.Debug("fetching servarr history", zap.String("url", parsedURL.Host))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check API key")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var records []HistoryRecord
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Ensure we return an empty array instead of nil
	if records == nil {
		records = make([]HistoryRecord, 0)
	}

	c.logger.Debug("fetched servarr history", zap.Int("records", len(records)))

	return records, nil
}

// GetQueueByDownloadID retrieves queue entries for a specific download ID
func (c *Client) GetQueueByDownloadID(ctx context.Context, baseURL string, apiKey string, downloadID string) ([]QueueRecord, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL.Path = parsedURL.Path + "/api/v3/queue"
	query := parsedURL.Query()
	// Servarrs store downloadId in uppercase, so convert it
	query.Add("downloadId", strings.ToUpper(downloadID))
	query.Add("includeEpisode", "true") // For Sonarr - includes episode details
	query.Add("includeMovie", "true")   // For Radarr - includes movie details
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Api-Key", apiKey)

	c.logger.Debug("fetching servarr queue",
		zap.String("url", parsedURL.Host),
		zap.String("downloadId", downloadID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check API key")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var queueResp QueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&queueResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Ensure we return an empty array instead of nil
	if queueResp.Records == nil {
		queueResp.Records = make([]QueueRecord, 0)
	}

	// Filter records by downloadId (case-insensitive comparison)
	// The server-side filter might not work properly depending on case
	var filteredRecords []QueueRecord
	downloadIDUpper := strings.ToUpper(downloadID)
	for _, record := range queueResp.Records {
		if strings.EqualFold(record.DownloadID, downloadIDUpper) {
			filteredRecords = append(filteredRecords, record)
		}
	}

	c.logger.Debug("fetched servarr queue",
		zap.Int("total_records", len(queueResp.Records)),
		zap.Int("filtered_records", len(filteredRecords)),
		zap.String("downloadId", downloadID))

	return filteredRecords, nil
}
