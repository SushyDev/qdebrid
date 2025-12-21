# qDebrid v2 - Complete Rewrite

A production-ready qBittorrent API proxy for Real-Debrid with bulletproof rate limiting and retry logic.

## 🎯 What's New in v2

### Major Improvements

1. **Bulletproof Rate Limiting**
   - Token bucket algorithm prevents API abuse
   - Configurable requests/minute (default: 20 req/min)
   - Burst capacity for handling spikes
   - Minimum interval enforcement

2. **Robust Retry System**
   - Exponential backoff with jitter
   - Never drops operations - queues and retries
   - Automatic retry on transient errors (429, 500, 503)
   - Context-aware cancellation

3. **Clean Architecture**
   - Dependency injection throughout
   - No global state
   - Fully testable components
   - Clear separation of concerns

4. **Production Ready**
   - Graceful shutdown (respects ongoing operations)
   - Structured logging with zap
   - HTTP middleware (logging, recovery, timeout)
   - Comprehensive error handling

5. **Developer Friendly**
   - Type-safe configuration
   - CLI flags for all operations
   - Example config generation
   - Clear error messages

## 🚀 Quick Start

### Prerequisites

- Go 1.23.2 or later
- Real-Debrid account with API token

### Installation

```bash
cd v2

# Install dependencies
make install

# Build
make build

# Generate config file
make write-config

# Edit config.yml with your Real-Debrid token
nano config.yml

# Run
make run
```

### Configuration

Generate an example configuration:

```bash
./build/qdebrid -example-config > config.yml
```

Minimal required configuration:

```yaml
server:
  port: 8080

real_debrid:
  token: "YOUR_REAL_DEBRID_TOKEN"

qbittorrent:
  save_path: "/mnt/debrid/media"

logging:
  level: "info"
```

## 📖 Configuration Reference

### Server Settings

```yaml
server:
  host: ""        # Bind address (empty = all interfaces)
  port: 8080      # HTTP port
```

### Real-Debrid Settings

```yaml
real_debrid:
  token: ""                      # Required: Your RD API token
  requests_per_minute: 20        # Rate limit (RD allows ~60)
  max_retries: 10                # Max retry attempts
  allowed_file_types:            # File types to download
    - "mkv"
    - "mp4"
    - "avi"
  min_file_size_bytes: 524288000 # Min file size (500MB)
```

### qBittorrent Settings

```yaml
qbittorrent:
  category_name: "qdebrid"       # Category shown in *Arr
  save_path: "/mnt/debrid/media" # Media storage path
  validate_paths: true            # Check if files exist
```

### Media Validation Settings

**New in v2.1**: Automatic media validation with ffprobe

```yaml
media_validation:
  enabled: false                      # Enable media validation with ffprobe
  require_downloaded: true            # Reject if torrent status is not 'downloaded'
  streamable_extensions:              # File extensions to validate
    - "mkv"
    - "mp4"
    - "avi"
    - "m4v"
    - "mov"
    - "wmv"
    - "webm"
  require_video_stream: true          # Reject if no video stream found
  require_audio_stream: true          # Reject if no audio stream found
  min_duration_seconds: 0             # Minimum video duration (0 = disabled)
  ffprobe_timeout: 30                 # FFprobe timeout in seconds
  reject_sample_files: true           # Reject sample files based on duration
  sample_min_runtime: 300             # Minimum runtime in seconds (5 minutes)
```

**How it works:**
1. When a torrent is added, qDebrid validates media files using ffprobe
2. Each streamable file is checked for video/audio streams, duration, and quality
3. If validation fails, the torrent is automatically deleted and rejected
4. Radarr/Sonarr receives the rejection and can try another release

**Use cases:**
- Automatically reject sample files that sneak into releases
- Ensure media files have both video and audio streams
- Reject corrupted or incomplete downloads
- Filter out non-playable files before they reach your media library

**Requirements:**
- ffprobe must be installed (included in Docker image)
- Torrent must reach 'downloaded' status first (if `require_downloaded: true`)
- Internet connection to unrestrict Real-Debrid links for validation

### Logging Settings

```yaml
logging:
  level: "info"         # debug, info, warn, error
  output_path: "stdout" # stdout, stderr, or file path
  json: false           # JSON format output
```

## 🔧 Usage

### Command Line Options

```bash
# Use specific config file
./qdebrid -config /path/to/config.yml

# Generate example config
./qdebrid -example-config

# Write example config to file
./qdebrid -write-config config.yml
```

### Environment Variables

Set `QDEBRID_CONFIG` to specify config file path:

```bash
export QDEBRID_CONFIG=/etc/qdebrid/config.yml
./qdebrid
```

### Using with *Arr Applications

1. In your *Arr app (Sonarr/Radarr), go to Settings → Download Clients
2. Add qBittorrent client with these settings:
   - **Host**: `localhost` (or qDebrid host)
   - **Port**: `8080` (or configured port)
   - **Username**: Your *Arr URL (e.g., `http://localhost:8989`)
   - **Password**: Your *Arr API Key
   - **Category**: `qdebrid` (or configured category)

The username/password hack allows qDebrid to filter torrents per *Arr app.

## 🏗️ Architecture

### Project Structure

```
v2/
├── cmd/qdebrid/              # Application entry point
├── internal/
│   ├── cache/                # In-memory cache with TTL
│   ├── config/               # Configuration management
│   ├── debrid/               # Real-Debrid client wrapper
│   ├── logger/               # Structured logging setup
│   ├── qbittorrent/          # qBittorrent API handlers
│   │   ├── handler.go        # HTTP request handlers
│   │   ├── models.go         # Data models & converters
│   │   └── server.go         # HTTP server & middleware
│   └── servarr/              # Servarr integration
└── pkg/
    └── retry/                # Retry & rate limiting logic
```

### Component Overview

#### Retry Package (`pkg/retry/`)
- **Retryer**: Exponential backoff with jitter
- **RateLimiter**: Token bucket rate limiting
- **Queue**: Operation queue with retry & rate limiting

#### Debrid Client (`internal/debrid/`)
- Wraps `real_debrid_go` library
- All operations go through retry queue
- Smart file selection based on config
- Context propagation for cancellation

#### qBittorrent Handler (`internal/qbittorrent/`)
- HTTP handlers for qBittorrent API
- Dependency injection for testability
- Cache integration with configurable TTL
- Proper error responses

#### HTTP Server (`internal/qbittorrent/server.go`)
- Standard library `net/http` 
- Middleware chain: logging → recovery → timeout
- Graceful shutdown support

## 🔄 API Rate Limiting

### How It Works

1. **Token Bucket**: Refills at configured rate
2. **Request Queue**: All operations wait for token
3. **Minimum Interval**: Enforces spacing between requests
4. **Retry on 429**: Automatic backoff on rate limit errors

### Configuration

```yaml
real_debrid:
  requests_per_minute: 20  # Conservative default
```

Real-Debrid allows ~60 requests/minute. We default to 20 for safety margin.

### Burst Handling

The system allows bursts of 3 requests to handle occasional spikes, then enforces the configured rate.

## 🔁 Retry Strategy

### Automatic Retry

Retries these errors automatically:
- HTTP 429 (Rate Limit)
- HTTP 500 (Internal Server Error)
- HTTP 502 (Bad Gateway)
- HTTP 503 (Service Unavailable)
- HTTP 504 (Gateway Timeout)
- Network timeouts
- Temporary network errors

### No Retry

Does not retry:
- HTTP 400 (Bad Request)
- HTTP 401 (Unauthorized)
- HTTP 404 (Not Found)
- Invalid configuration
- User cancellation

### Backoff Schedule

Default backoff with jitter:
- Attempt 1: 2s
- Attempt 2: 4s
- Attempt 3: 8s
- Attempt 4: 16s
- Attempt 5: 32s
- Attempts 6-10: 5min (max)

## 📊 Logging

### Log Levels

- **debug**: Verbose, shows cache hits/misses, all operations
- **info**: Normal operation, requests, important events
- **warn**: Warnings, retries, non-fatal errors
- **error**: Errors requiring attention

### Example Logs

```
2025-01-07T10:30:45Z INFO  qDebrid starting version=2.0.0
2025-01-07T10:30:45Z INFO  starting server addr=:8080
2025-01-07T10:31:12Z INFO  http request method=POST path=/api/v2/torrents/add status=200 duration=1.2s
2025-01-07T10:31:13Z WARN  operation failed, retrying attempt=1 backoff=2s error="HTTP 429"
2025-01-07T10:31:15Z INFO  operation succeeded after retries attempts=1
```

## 🐋 Docker Support

Coming soon - Dockerfile and docker-compose.yml will be added.

## 🛠️ Development

### Building

```bash
make build          # Build binary
make clean          # Clean build files
make install        # Install dependencies
```

### Testing

```bash
make test           # Run tests
make coverage       # Show coverage report
make lint           # Run linter
make vet            # Run go vet
make fmt            # Format code
```

### Running

```bash
make run            # Run with config.yml
make dev            # Run with config.example.yml
```

## 📝 Makefile Targets

```bash
make help           # Show all available commands
```

## 🤝 Contributing

1. Use `make fmt` before committing
2. Ensure `make test` passes
3. Add tests for new features
4. Update documentation

## 📄 License

GNU GPLv3 - See LICENSE file

## 🙏 Acknowledgments

- Uses [real_debrid_go](https://github.com/sushydev/real_debrid_go) library
- Built with [zap](https://github.com/uber-go/zap) for logging
- Inspired by the original qDebrid implementation

## 🔗 Links

- [Real-Debrid API Documentation](https://api.real-debrid.com)
- [qBittorrent Web API](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))
