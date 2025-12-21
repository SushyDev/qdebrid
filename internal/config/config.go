package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrConfigNotFound  = errors.New("config file not found")
	ErrInvalidConfig   = errors.New("invalid configuration")
	ErrMissingRequired = errors.New("missing required field")
)

// Config represents the application configuration
type Config struct {
	Server          ServerConfig          `yaml:"server"`
	RealDebrid      RealDebridConfig      `yaml:"real_debrid"`
	QBittorrent     QBittorrentConfig     `yaml:"qbittorrent"`
	MediaValidation MediaValidationConfig `yaml:"media_validation"`
	Data            DataConfig            `yaml:"data"`
	Logging         LoggingConfig         `yaml:"logging"`
	Testing         TestingConfig         `yaml:"testing"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// RealDebridConfig holds Real-Debrid API configuration
type RealDebridConfig struct {
	Token             string   `yaml:"token"`
	RequestsPerMinute int      `yaml:"requests_per_minute"`
	MaxRetries        int      `yaml:"max_retries"`
	AllowedFileTypes  []string `yaml:"allowed_file_types"`
	MinFileSizeBytes  int64    `yaml:"min_file_size_bytes"`
}

// MediaValidationConfig holds media validation configuration
type MediaValidationConfig struct {
	Enabled              bool     `yaml:"enabled"`
	RequireDownloaded    bool     `yaml:"require_downloaded"`
	StreamableExtensions []string `yaml:"streamable_extensions"`
	RequireVideoStream   bool     `yaml:"require_video_stream"`
	RequireAudioStream   bool     `yaml:"require_audio_stream"`
	MinDurationSeconds   int      `yaml:"min_duration_seconds"`
	FFProbeTimeout       int      `yaml:"ffprobe_timeout"`
	RejectSampleFiles    bool     `yaml:"reject_sample_files"`
	SampleMinRuntime     int      `yaml:"sample_min_runtime"`
}

// QBittorrentConfig holds qBittorrent emulation configuration
type QBittorrentConfig struct {
	CategoryName  string `yaml:"category_name"`
	SavePath      string `yaml:"save_path"`
	ValidatePaths bool   `yaml:"validate_paths"`
}

// DataConfig holds data persistence configuration
type DataConfig struct {
	Directory        string `yaml:"directory"`
	CleanupMaxAge    string `yaml:"cleanup_max_age"`    // e.g., "168h" for 7 days
	AutoSaveInterval string `yaml:"auto_save_interval"` // e.g., "5m" for 5 minutes
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	OutputPath string `yaml:"output_path"`
	JSON       bool   `yaml:"json"`
}

// TestingConfig holds testing configuration (not committed to git)
type TestingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	TestMagnet  string `yaml:"test_magnet"`
	TestTorrent string `yaml:"test_torrent"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate Real-Debrid token
	if strings.TrimSpace(c.RealDebrid.Token) == "" {
		return fmt.Errorf("%w: real_debrid.token", ErrMissingRequired)
	}

	// Validate server port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("%w: server.port must be between 1 and 65535", ErrInvalidConfig)
	}

	// Validate rate limiting
	if c.RealDebrid.RequestsPerMinute < 1 {
		return fmt.Errorf("%w: real_debrid.requests_per_minute must be > 0", ErrInvalidConfig)
	}

	// Validate save path
	if strings.TrimSpace(c.QBittorrent.SavePath) == "" {
		return fmt.Errorf("%w: qbittorrent.save_path", ErrMissingRequired)
	}

	// Validate save path exists if validation is enabled
	if c.QBittorrent.ValidatePaths {
		if _, err := os.Stat(c.QBittorrent.SavePath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%w: qbittorrent.save_path does not exist: %s", ErrInvalidConfig, c.QBittorrent.SavePath)
			}
			return fmt.Errorf("failed to check save_path: %w", err)
		}
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("%w: logging.level must be one of: debug, info, warn, error", ErrInvalidConfig)
	}

	return nil
}

// SetDefaults sets default values for optional fields
func (c *Config) SetDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

	if c.RealDebrid.RequestsPerMinute == 0 {
		c.RealDebrid.RequestsPerMinute = 20 // Conservative default
	}

	if c.RealDebrid.MaxRetries == 0 {
		c.RealDebrid.MaxRetries = 10
	}

	if len(c.RealDebrid.AllowedFileTypes) == 0 {
		c.RealDebrid.AllowedFileTypes = []string{"mkv", "mp4", "avi"}
	}

	if c.RealDebrid.MinFileSizeBytes == 0 {
		c.RealDebrid.MinFileSizeBytes = 500 * 1024 * 1024 // 500MB
	}

	// Media validation defaults
	if len(c.MediaValidation.StreamableExtensions) == 0 {
		c.MediaValidation.StreamableExtensions = []string{"mkv", "mp4", "avi", "m4v", "mov", "wmv", "webm"}
	}
	if c.MediaValidation.FFProbeTimeout == 0 {
		c.MediaValidation.FFProbeTimeout = 30 // 30 seconds
	}
	if c.MediaValidation.SampleMinRuntime == 0 {
		c.MediaValidation.SampleMinRuntime = 300 // 5 minutes
	}

	if c.QBittorrent.CategoryName == "" {
		c.QBittorrent.CategoryName = "qdebrid"
	}

	if c.Data.Directory == "" {
		c.Data.Directory = "./data"
	}

	if c.Data.CleanupMaxAge == "" {
		c.Data.CleanupMaxAge = "168h" // 7 days
	}

	if c.Data.AutoSaveInterval == "" {
		c.Data.AutoSaveInterval = "5m" // 5 minutes
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}

	if c.Logging.OutputPath == "" {
		c.Logging.OutputPath = "stdout"
	}
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.SetDefaults()

	// Validate
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadFromEnv attempts to load config from environment variable or default locations
func LoadFromEnv() (*Config, error) {
	// Check environment variable
	if path := os.Getenv("QDEBRID_CONFIG"); path != "" {
		return Load(path)
	}

	// Try default locations
	locations := []string{
		"config.yml",
		"config.yaml",
		"/etc/qdebrid/config.yml",
		filepath.Join(os.Getenv("HOME"), ".config", "qdebrid", "config.yml"),
	}

	var lastErr error
	for _, location := range locations {
		config, err := Load(location)
		if err == nil {
			return config, nil
		}
		if !errors.Is(err, ErrConfigNotFound) {
			return nil, err // Return actual parsing errors immediately
		}
		lastErr = err
	}

	return nil, fmt.Errorf("no config file found in default locations: %w", lastErr)
}

// ExampleConfig returns an example configuration as a string
func ExampleConfig() string {
	return `# qDebrid Configuration
server:
  host: ""        # Leave empty to listen on all interfaces
  port: 8080      # HTTP server port

real_debrid:
  token: "YOUR_REAL_DEBRID_TOKEN_HERE"  # Required: Your Real-Debrid API token
  requests_per_minute: 20                # Rate limit (Real-Debrid allows ~60/min, we use 20 to be safe)
  max_retries: 10                        # Maximum retry attempts for failed requests
  allowed_file_types:                    # Only download these file types
    - "mkv"
    - "mp4"
    - "avi"
  min_file_size_bytes: 524288000         # Minimum file size (500MB default)

qbittorrent:
  category_name: "qdebrid"                           # Category name shown in *Arr apps
  save_path: "/mnt/debrid/media"                     # Path where media is saved (must exist)
  validate_paths: true                                # Verify files exist on disk

media_validation:
  enabled: false                                      # Enable media validation with ffprobe
  require_downloaded: true                            # Only validate if torrent status is 'downloaded'
  streamable_extensions:                              # File extensions to validate
    - "mkv"
    - "mp4"
    - "avi"
    - "m4v"
    - "mov"
    - "wmv"
    - "webm"
  require_video_stream: true                          # Reject if no video stream found
  require_audio_stream: true                          # Reject if no audio stream found
  min_duration_seconds: 0                             # Minimum video duration (0 = disabled)
  ffprobe_timeout: 30                                 # FFprobe timeout in seconds
  reject_sample_files: true                           # Reject sample files based on duration
  sample_min_runtime: 300                             # Minimum runtime in seconds (5 minutes)

data:
  directory: "./data"                                 # Directory for persistent data (history, state)
  cleanup_max_age: "168h"                             # Remove old history entries after this duration (168h = 7 days)
  auto_save_interval: "5m"                            # How often to auto-save history to disk

logging:
  level: "info"                # Log level: debug, info, warn, error
  output_path: "stdout"        # Output: stdout, stderr, or file path
  json: false                  # Output logs in JSON format
`
}

// WriteExample writes an example config file to the specified path
func WriteExample(path string) error {
	return os.WriteFile(path, []byte(ExampleConfig()), 0644)
}
