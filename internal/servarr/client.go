package servarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	c.logger.Debug("fetched servarr history", zap.Int("records", len(records)))

	return records, nil
}
