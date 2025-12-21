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
	ID          int    `json:"id"`
	DownloadID  string `json:"downloadId"`
	EventType   string `json:"eventType"`
	Date        string `json:"date"`
	SourceTitle string `json:"sourceTitle"`

	// Radarr fields
	MovieID *int `json:"movieId,omitempty"`

	// Sonarr fields
	SeriesID  *int `json:"seriesId,omitempty"`
	EpisodeID *int `json:"episodeId,omitempty"`
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

	// Log the full queue response JSON for debugging
	if queueJSON, err := json.Marshal(queueResp); err == nil {
		c.logger.Info("servarr queue API response",
			zap.String("downloadId", downloadID),
			zap.String("response_json", string(queueJSON)))
	}

	// Ensure we return an empty array instead of nil
	if queueResp.Records == nil {
		queueResp.Records = make([]QueueRecord, 0)
	}

	// Filter records by downloadId (case-insensitive comparison)
	// The server-side filter might not work properly depending on case
	var filteredRecords []QueueRecord

	// Log all queue record downloadIds for debugging
	if len(queueResp.Records) > 0 {
		c.logger.Debug("queue records details",
			zap.Int("total", len(queueResp.Records)))
		for i, record := range queueResp.Records {
			c.logger.Debug("queue record",
				zap.Int("index", i),
				zap.String("downloadId", record.DownloadID),
				zap.String("title", record.Title),
				zap.String("status", record.Status))
		}
	}

	for _, record := range queueResp.Records {
		if strings.EqualFold(record.DownloadID, downloadID) {
			filteredRecords = append(filteredRecords, record)
		}
	}

	c.logger.Debug("fetched servarr queue",
		zap.Int("total_records", len(queueResp.Records)),
		zap.Int("filtered_records", len(filteredRecords)),
		zap.String("downloadId", downloadID))

	return filteredRecords, nil
}

// GetHistoryByDownloadID retrieves history entries for a specific download ID
func (c *Client) GetHistoryByDownloadID(ctx context.Context, baseURL string, apiKey string, downloadID string) ([]HistoryRecord, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL.Path = parsedURL.Path + "/api/v3/history"
	query := parsedURL.Query()
	// Use downloadId filter directly - API supports it
	query.Add("downloadId", strings.ToUpper(downloadID))
	// Add pagination parameters to ensure we get results
	query.Add("page", "1")
	query.Add("pageSize", "100") // Should be enough for season packs
	// Include episode/movie details for proper counting
	query.Add("includeEpisode", "true")
	query.Add("includeMovie", "true")
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Api-Key", apiKey)

	c.logger.Debug("fetching servarr history by downloadId",
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

	var historyResp struct {
		Records []HistoryRecord `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Log the full history response JSON for debugging
	if historyJSON, err := json.Marshal(historyResp); err == nil {
		c.logger.Info("servarr history API response",
			zap.String("downloadId", downloadID),
			zap.String("response_json", string(historyJSON)))
	}

	// Ensure we return an empty array instead of nil
	if historyResp.Records == nil {
		historyResp.Records = make([]HistoryRecord, 0)
	}

	// Filter records by eventType "grabbed" since API might return other event types
	var grabbedRecords []HistoryRecord
	for _, record := range historyResp.Records {
		if record.EventType == "grabbed" {
			grabbedRecords = append(grabbedRecords, record)
		}
	}

	// Log details about what we found
	if len(historyResp.Records) > 0 {
		c.logger.Debug("history records details",
			zap.Int("total", len(historyResp.Records)),
			zap.Int("grabbed", len(grabbedRecords)))
		for i, record := range historyResp.Records {
			c.logger.Debug("history record",
				zap.Int("index", i),
				zap.String("downloadId", record.DownloadID),
				zap.String("eventType", record.EventType),
				zap.String("sourceTitle", record.SourceTitle))
		}
	}

	c.logger.Debug("fetched servarr history by downloadId",
		zap.Int("total_records", len(historyResp.Records)),
		zap.Int("grabbed_records", len(grabbedRecords)),
		zap.String("downloadId", downloadID))

	return grabbedRecords, nil
}

// CommandResponse represents the response from a Servarr command API call
type CommandResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RefreshMonitoredDownloads triggers Sonarr/Radarr to refresh the download queue
func (c *Client) RefreshMonitoredDownloads(ctx context.Context, baseURL string, apiKey string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL.Path = parsedURL.Path + "/api/v3/command"

	// Command payload
	payload := map[string]string{
		"name": "RefreshMonitoredDownloads",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", parsedURL.String(), strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("triggering RefreshMonitoredDownloads",
		zap.String("url", parsedURL.Host))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: check API key")
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var cmdResp CommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	c.logger.Info("RefreshMonitoredDownloads command response",
		zap.Int("commandId", cmdResp.ID),
		zap.String("name", cmdResp.Name),
		zap.String("status", cmdResp.Status))

	return nil
}
