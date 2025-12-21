package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"qdebrid/internal/cache"
	"qdebrid/internal/config"
	"qdebrid/internal/debrid"
)

// findAndLoadConfig tries to load config from various locations
func findAndLoadConfig() (*config.Config, error) {
	locations := []string{
		"config.yml",
		"../../config.yml",
		os.Getenv("HOME") + "/.config/qdebrid/config.yml",
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return config.Load(path)
		}
	}

	return nil, config.ErrConfigNotFound
}

// TestIntegrationRealDebridAPI tests the Real-Debrid API integration
// This test requires a valid Real-Debrid API token in config.yml
// Run with: go test -v ./internal/integration/...
func TestIntegrationRealDebridAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Try to load config from default locations
	cfg, err := findAndLoadConfig()
	if err != nil {
		t.Skipf("skipping integration test: could not load config: %v", err)
	}

	// Check if we have a real token (not the fake one)
	if cfg.RealDebrid.Token == "your-real-debrid-token-here" || cfg.RealDebrid.Token == "" {
		t.Skip("skipping integration test: no real API token configured")
	}

	// Check if we have a test magnet
	if cfg.Testing.TestMagnet == "" {
		t.Skip("skipping integration test: no test magnet configured in testing.test_magnet")
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create clients
	debridClient := debrid.NewClient(&cfg.RealDebrid, &cfg.MediaValidation, logger)
	defer debridClient.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("AddMagnet", func(t *testing.T) {
		// Add test magnet
		torrentID, err := debridClient.AddTorrentByURL(ctx, cfg.Testing.TestMagnet)
		if err != nil {
			t.Fatalf("failed to add magnet: %v", err)
		}

		if torrentID == "" {
			t.Fatal("torrent ID is empty")
		}

		t.Logf("Successfully added magnet, torrent ID: %s", torrentID)

		// Clean up - delete the torrent
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()

			if err := debridClient.DeleteTorrent(cleanupCtx, torrentID); err != nil {
				t.Logf("failed to delete test torrent: %v", err)
			} else {
				t.Logf("Test torrent cleaned up successfully")
			}
		}()
	})

	t.Run("GetTorrents", func(t *testing.T) {
		torrents, err := debridClient.GetTorrents(ctx)
		if err != nil {
			t.Fatalf("failed to get torrents: %v", err)
		}

		if torrents == nil {
			t.Fatal("torrents list is nil")
		}

		t.Logf("Found %d torrents", len(*torrents))
	})

	t.Run("RateLimiting", func(t *testing.T) {
		// Make multiple rapid requests to test rate limiting
		start := time.Now()

		for i := 0; i < 3; i++ {
			_, err := debridClient.GetTorrents(ctx)
			if err != nil {
				t.Fatalf("request %d failed: %v", i, err)
			}
		}

		elapsed := time.Since(start)

		// With default 20 requests per minute (3 seconds per request), 3 requests should take ~6 seconds
		// (first is immediate, second waits 3s, third waits 3s)
		minExpected := 5 * time.Second
		if elapsed < minExpected {
			t.Logf("Warning: rate limiting might not be working correctly. Expected >= %v, got %v", minExpected, elapsed)
		} else {
			t.Logf("Rate limiting working correctly: %v elapsed for 3 requests", elapsed)
		}
	})
}

// TestCacheIntegration tests cache behavior
func TestCacheIntegration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cleanupInterval := 100 * time.Millisecond
	c := cache.New(cleanupInterval, logger)
	defer c.Close()

	t.Run("BasicOperations", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")

		// Set value
		c.Set(key, value, 1*time.Minute)

		// Get value
		result, ok := c.Get(key)
		if !ok {
			t.Fatal("expected to find value in cache")
		}

		if string(result) != string(value) {
			t.Errorf("expected %q, got %q", value, result)
		}

		// Get stats
		stats := c.Stats()
		if stats["total"] < 1 {
			t.Errorf("expected at least 1 item, got %d", stats["total"])
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		key := "expire-test"
		value := []byte("will-expire")

		// Set with short TTL
		c.Set(key, value, 100*time.Millisecond)

		// Verify it exists
		if _, ok := c.Get(key); !ok {
			t.Fatal("value should exist immediately after set")
		}

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Verify it expired
		if _, ok := c.Get(key); ok {
			t.Fatal("value should have expired")
		}
	})

	t.Run("JSON", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Value int
		}

		key := "json-test"
		original := TestStruct{Name: "test", Value: 42}

		// Set JSON
		if err := c.SetJSON(key, original, 1*time.Minute); err != nil {
			t.Fatalf("failed to set JSON: %v", err)
		}

		// Get JSON
		var retrieved TestStruct
		ok, err := c.GetJSON(key, &retrieved)
		if err != nil {
			t.Fatalf("failed to get JSON: %v", err)
		}

		if !ok {
			t.Fatal("expected to find JSON value in cache")
		}

		if retrieved.Name != original.Name || retrieved.Value != original.Value {
			t.Errorf("expected %+v, got %+v", original, retrieved)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		// Set some values
		c.Set("key1", []byte("value1"), 1*time.Minute)
		c.Set("key2", []byte("value2"), 1*time.Minute)

		// Verify they exist
		stats := c.Stats()
		if stats["total"] < 2 {
			t.Fatalf("expected at least 2 items, got %d", stats["total"])
		}

		// Clear cache
		c.Clear()

		// Wait a bit for cleanup
		time.Sleep(150 * time.Millisecond)

		// Verify cache is empty
		stats = c.Stats()
		if stats["total"] != 0 {
			t.Errorf("expected 0 items after clear, got %d", stats["total"])
		}
	})
}

// TestConfigIntegration tests configuration loading and validation
func TestConfigIntegration(t *testing.T) {
	t.Run("LoadConfig", func(t *testing.T) {
		cfg, err := findAndLoadConfig()
		if err != nil {
			t.Skipf("skipping test: could not load config: %v", err)
		}

		// Validate basic fields
		if cfg.Server.Host == "" {
			t.Error("server host should not be empty")
		}

		if cfg.Server.Port <= 0 {
			t.Error("server port should be positive")
		}

		if cfg.RealDebrid.RequestsPerMinute <= 0 {
			t.Error("requests per minute should be positive")
		}

		t.Logf("Config loaded successfully:")
		t.Logf("  Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
		t.Logf("  Real-Debrid rate limit: %d req/min", cfg.RealDebrid.RequestsPerMinute)
		t.Logf("  qBittorrent save path: %s", cfg.QBittorrent.SavePath)
	})

	t.Run("ValidationRules", func(t *testing.T) {
		tests := []struct {
			name      string
			cfg       config.Config
			shouldErr bool
		}{
			{
				name: "valid config",
				cfg: config.Config{
					Server: config.ServerConfig{
						Host: "0.0.0.0",
						Port: 8080,
					},
					QBittorrent: config.QBittorrentConfig{
						SavePath:     "/tmp/test",
						CategoryName: "test",
					},
					RealDebrid: config.RealDebridConfig{
						Token:             "test-token",
						RequestsPerMinute: 60,
						MaxRetries:        10,
					},
					MediaValidation: config.MediaValidationConfig{
						MinFileSizeBytes:     1024,
						StreamableExtensions: []string{"mkv", "mp4"},
					},
					Logging: config.LoggingConfig{
						Level: "info",
					},
				},
				shouldErr: false,
			},
			{
				name: "missing token",
				cfg: config.Config{
					Server: config.ServerConfig{
						Host: "0.0.0.0",
						Port: 8080,
					},
					QBittorrent: config.QBittorrentConfig{
						SavePath: "/tmp/test",
					},
					RealDebrid: config.RealDebridConfig{
						RequestsPerMinute: 60,
					},
					Logging: config.LoggingConfig{
						Level: "info",
					},
				},
				shouldErr: true,
			},
			{
				name: "invalid port",
				cfg: config.Config{
					Server: config.ServerConfig{
						Host: "0.0.0.0",
						Port: -1,
					},
					QBittorrent: config.QBittorrentConfig{
						SavePath: "/tmp/test",
					},
					RealDebrid: config.RealDebridConfig{
						Token:             "test-token",
						RequestsPerMinute: 60,
					},
					Logging: config.LoggingConfig{
						Level: "info",
					},
				},
				shouldErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.cfg.Validate()
				if tt.shouldErr && err == nil {
					t.Error("expected validation error, got nil")
				}
				if !tt.shouldErr && err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			})
		}
	})
}
