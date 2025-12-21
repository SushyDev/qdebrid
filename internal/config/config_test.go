package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   Config
	}{
		{
			name:   "empty config gets all defaults",
			config: Config{},
			want: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				RealDebrid: RealDebridConfig{
					RequestsPerMinute: 20,
					MaxRetries:        10,
				},
				MediaValidation: MediaValidationConfig{
					StreamableExtensions: []string{"mkv", "mp4", "avi", "m4v", "mov", "wmv", "webm"},
					MinFileSizeBytes:     500 * 1024 * 1024,
					FFProbeTimeout:       30,
					SampleMinRuntime:     300,
				},
				QBittorrent: QBittorrentConfig{
					CategoryName: "qdebrid",
				},
				Data: DataConfig{
					Directory:        "./data",
					CleanupMaxAge:    "168h",
					AutoSaveInterval: "5m",
				},
				Logging: LoggingConfig{
					Level:      "info",
					OutputPath: "stdout",
				},
			},
		},
		{
			name: "partial config preserves values",
			config: Config{
				Server: ServerConfig{
					Port: 9090,
				},
				RealDebrid: RealDebridConfig{
					RequestsPerMinute: 30,
				},
			},
			want: Config{
				Server: ServerConfig{
					Port: 9090, // preserved
				},
				RealDebrid: RealDebridConfig{
					RequestsPerMinute: 30, // preserved
					MaxRetries:        10,
				},
				MediaValidation: MediaValidationConfig{
					StreamableExtensions: []string{"mkv", "mp4", "avi", "m4v", "mov", "wmv", "webm"},
					MinFileSizeBytes:     500 * 1024 * 1024,
					FFProbeTimeout:       30,
					SampleMinRuntime:     300,
				},
				QBittorrent: QBittorrentConfig{
					CategoryName: "qdebrid",
				},
				Data: DataConfig{
					Directory:        "./data",
					CleanupMaxAge:    "168h",
					AutoSaveInterval: "5m",
				},
				Logging: LoggingConfig{
					Level:      "info",
					OutputPath: "stdout",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config
			got.SetDefaults()

			// Check server
			if got.Server.Port != tt.want.Server.Port {
				t.Errorf("Server.Port = %v, want %v", got.Server.Port, tt.want.Server.Port)
			}

			// Check real debrid
			if got.RealDebrid.RequestsPerMinute != tt.want.RealDebrid.RequestsPerMinute {
				t.Errorf("RealDebrid.RequestsPerMinute = %v, want %v",
					got.RealDebrid.RequestsPerMinute, tt.want.RealDebrid.RequestsPerMinute)
			}
			if got.RealDebrid.MaxRetries != tt.want.RealDebrid.MaxRetries {
				t.Errorf("RealDebrid.MaxRetries = %v, want %v",
					got.RealDebrid.MaxRetries, tt.want.RealDebrid.MaxRetries)
			}
			if got.MediaValidation.MinFileSizeBytes != tt.want.MediaValidation.MinFileSizeBytes {
				t.Errorf("MediaValidation.MinFileSizeBytes = %v, want %v",
					got.MediaValidation.MinFileSizeBytes, tt.want.MediaValidation.MinFileSizeBytes)
			}

			// Check qbittorrent
			if got.QBittorrent.CategoryName != tt.want.QBittorrent.CategoryName {
				t.Errorf("QBittorrent.CategoryName = %v, want %v",
					got.QBittorrent.CategoryName, tt.want.QBittorrent.CategoryName)
			}

			// Check logging
			if got.Logging.Level != tt.want.Logging.Level {
				t.Errorf("Logging.Level = %v, want %v", got.Logging.Level, tt.want.Logging.Level)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				RealDebrid: RealDebridConfig{
					Token:             "test_token",
					RequestsPerMinute: 20,
				},
				QBittorrent: QBittorrentConfig{
					SavePath: "/tmp",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "missing token",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				RealDebrid: RealDebridConfig{
					RequestsPerMinute: 20,
				},
				QBittorrent: QBittorrentConfig{
					SavePath: "/tmp",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "real_debrid.token",
		},
		{
			name: "invalid port",
			config: Config{
				Server: ServerConfig{
					Port: 99999,
				},
				RealDebrid: RealDebridConfig{
					Token:             "test_token",
					RequestsPerMinute: 20,
				},
				QBittorrent: QBittorrentConfig{
					SavePath: "/tmp",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "server.port",
		},
		{
			name: "invalid requests per minute",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				RealDebrid: RealDebridConfig{
					Token:             "test_token",
					RequestsPerMinute: 0,
				},
				QBittorrent: QBittorrentConfig{
					SavePath: "/tmp",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "requests_per_minute",
		},
		{
			name: "missing save path",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				RealDebrid: RealDebridConfig{
					Token:             "test_token",
					RequestsPerMinute: 20,
				},
				QBittorrent: QBittorrentConfig{},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "save_path",
		},
		{
			name: "invalid log level",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				RealDebrid: RealDebridConfig{
					Token:             "test_token",
					RequestsPerMinute: 20,
				},
				QBittorrent: QBittorrentConfig{
					SavePath: "/tmp",
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "logging.level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, should contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name:     "valid config file",
			filename: "valid.yml",
			content: `
server:
  port: 9090
real_debrid:
  token: "test_token_123"
  requests_per_minute: 30
qbittorrent:
  save_path: "/tmp/test"
  category_name: "mycat"
logging:
  level: "debug"
`,
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				if c.Server.Port != 9090 {
					t.Errorf("Port = %v, want 9090", c.Server.Port)
				}
				if c.RealDebrid.Token != "test_token_123" {
					t.Errorf("Token = %v, want test_token_123", c.RealDebrid.Token)
				}
				if c.RealDebrid.RequestsPerMinute != 30 {
					t.Errorf("RequestsPerMinute = %v, want 30", c.RealDebrid.RequestsPerMinute)
				}
				if c.QBittorrent.SavePath != "/tmp/test" {
					t.Errorf("SavePath = %v, want /tmp/test", c.QBittorrent.SavePath)
				}
				if c.Logging.Level != "debug" {
					t.Errorf("Level = %v, want debug", c.Logging.Level)
				}
			},
		},
		{
			name:     "invalid yaml",
			filename: "invalid.yml",
			content:  "this is not: valid: yaml:",
			wantErr:  true,
		},
		{
			name:     "missing required field",
			filename: "missing.yml",
			content: `
server:
  port: 8080
qbittorrent:
  save_path: "/tmp"
logging:
  level: "info"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Load config
			got, err := Load(filePath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if tt.validate != nil && got != nil {
					tt.validate(t, got)
				}
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Load() error should mention 'not found', got: %v", err)
	}
}

func TestExampleConfig(t *testing.T) {
	example := ExampleConfig()

	if example == "" {
		t.Error("ExampleConfig() returned empty string")
	}

	if !contains(example, "real_debrid") {
		t.Error("ExampleConfig() should contain 'real_debrid'")
	}

	if !contains(example, "token") {
		t.Error("ExampleConfig() should contain 'token'")
	}

	if !contains(example, "qbittorrent") {
		t.Error("ExampleConfig() should contain 'qbittorrent'")
	}
}

func TestWriteExample(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "example.yml")

	err := WriteExample(filePath)
	if err != nil {
		t.Fatalf("WriteExample() error = %v", err)
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("WriteExample() did not create file")
	}

	// Check file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if len(content) == 0 {
		t.Error("WriteExample() wrote empty file")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
