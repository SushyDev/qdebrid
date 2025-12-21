# qDebrid

A production-ready qBittorrent API proxy for Real-Debrid with intelligent rate limiting, automatic media validation, and seamless Sonarr/Radarr integration.

## Features

- **qBittorrent API Compatibility** - Drop-in replacement for qBittorrent with Sonarr/Radarr
- **Real-Debrid Integration** - Automatic torrent handling via Real-Debrid cloud service
- **Smart Rate Limiting** - Token bucket algorithm prevents API abuse (configurable req/min)
- **Media Validation** - FFprobe quality checks ensure only valid releases are imported
- **File Count Validation** - Season pack verification prevents incomplete releases
- **Bulletproof Retry Logic** - Exponential backoff with jitter, never drops operations
- **Production Ready** - Graceful shutdown, structured logging, comprehensive error handling
- **Docker Support** - Pre-built images with health checks and multi-arch support

## Quick Start with Docker

### Prerequisites

- Docker and Docker Compose
- Real-Debrid account and API token
- Sonarr/Radarr (optional, for *arr integration)

### 1. Create Configuration File

```bash
# Create config.yml
cat > config.yml << 'EOF'
server:
  host: "0.0.0.0"
  port: 8080

real_debrid:
  token: "YOUR_REAL_DEBRID_API_TOKEN"
  requests_per_minute: 20
  max_retries: 10
  additional_selectable_files:  # Files to select beyond video files
    - "srt"                     # Subtitles for Plex/Jellyfin
    - "nfo"                     # Metadata
    - "jpg"                     # Artwork

qbittorrent:
  category_name: "qdebrid"
  save_path: "/media"
  validate_paths: true

media_validation:
  enabled: true
  require_downloaded: true
  streamable_extensions:
    - "mkv"
    - "mp4"
    - "avi"
    - "m4v"
    - "mov"
    - "wmv"
    - "webm"
  min_file_size_bytes: 524288000  # 500MB
  require_video_stream: true
  require_audio_stream: true
  min_duration_seconds: 0
  ffprobe_timeout: 30
  reject_sample_files: true
  sample_min_runtime: 300
  validate_file_count: true

data:
  directory: "./data"
  cleanup_max_age: "168h"
  auto_save_interval: "5m"

logging:
  level: "info"
  output_path: "stdout"
  json: false
EOF
```

Replace `YOUR_REAL_DEBRID_API_TOKEN` with your actual Real-Debrid API token from [https://real-debrid.com/apitoken](https://real-debrid.com/apitoken).

### 2. Start qDebrid

Using Docker Compose (recommended):

```bash
docker-compose up -d
```

Or using Docker run:

```bash
docker run -d \
  --name qdebrid \
  -p 8080:8080 \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  -v /path/to/media:/media \
  --restart unless-stopped \
  ghcr.io/yourusername/qdebrid:latest
```

### 3. Verify Health

```bash
curl http://localhost:8080/health
# Should return: {"status":"ok","service":"qdebrid","version":"2.0.0"}
```

### 4. Configure Sonarr/Radarr

1. Go to **Settings → Download Clients**
2. Add **qBittorrent** client:
   - **Host**: `localhost` (or your qDebrid host)
   - **Port**: `8080`
   - **Username**: Your Sonarr/Radarr URL (e.g., `http://localhost:8989`)
   - **Password**: Your Sonarr/Radarr API Key
   - **Category**: `qdebrid`

The username/password fields pass *arr credentials to qDebrid for file count validation.

## Torrent Verification Flow

qDebrid implements a comprehensive validation pipeline to ensure only high-quality, complete releases reach your media library:

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Torrent Added (from Sonarr/Radarr)                          │
│    - Receives magnet link or torrent file via qBittorrent API  │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Add to Real-Debrid                                           │
│    - Submits torrent to Real-Debrid via API                    │
│    - Real-Debrid downloads from seeders                         │
│    - Rate-limited with token bucket algorithm                   │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Check Download Status (if require_downloaded: true)         │
│    - Checks if torrent status is 'downloaded'                  │
│    - Rejects immediately if not downloaded                      │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. File Count Validation (if validate_file_count: true)        │
│    a) Extract *arr credentials from auth header                │
│    b) Query *arr queue API with torrent hash                   │
│    c) Get expected episode/movie count                         │
│    d) Count non-sample video files in torrent                  │
│    e) Compare: actual >= expected                              │
│    → REJECT if insufficient files (incomplete pack)            │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Quality Validation (if enabled)                             │
│    For each video file:                                         │
│    a) Unrestrict download link from Real-Debrid                │
│    b) Run ffprobe to extract media info                        │
│    c) Verify video stream exists                               │
│    d) Verify audio stream exists                               │
│    e) Check duration (reject if < min_duration_seconds)        │
│    f) Detect sample files (name patterns + duration)           │
│    → FAIL-FAST: First failure rejects entire torrent           │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. Validation Result                                            │
│    ✓ SUCCESS: Torrent marked ready for *arr                    │
│    ✗ FAILURE: Torrent deleted, *arr receives rejection         │
│                *arr automatically tries next release            │
└─────────────────────────────────────────────────────────────────┘
```

### Validation Examples

**Example 1: Complete Season Pack**
```
Sonarr requests: S01E01-E10 (10 episodes)
Torrent contains: 10 MKV files + 1 sample.mkv
Quality check: All files have video/audio, duration > 5 min
File count: 10 valid files (sample excluded)
Result: ✓ ACCEPTED
```

**Example 2: Incomplete Pack**
```
Sonarr requests: S01E01-E10 (10 episodes)
Torrent contains: 8 MKV files
Result: ✗ REJECTED - "insufficient video files: expected 10, found 8"
Sonarr automatically tries next release
```

**Example 3: Fake Release (samples only)**
```
Radarr requests: Movie
Torrent contains: 3 files named "sample", all < 2 minutes
Quality check: All marked as samples, durations too short
File count: 0 valid files
Result: ✗ REJECTED - "insufficient video files: expected 1, found 0"
```

### Why This Matters

Without validation, your media library could receive:
- Sample files disguised as full releases
- Incomplete season packs missing episodes
- Corrupted files without video/audio streams
- Low-quality releases with invalid codecs

With qDebrid's validation, only verified, complete releases are imported.

## File Selection

qDebrid automatically selects files from torrents based on your configuration, optimizing for Plex/Jellyfin media libraries.

### Video File Selection

Video files are selected if they match:
- **Extension**: Matches `media_validation.streamable_extensions` (mkv, mp4, avi, etc.)
- **Size**: Greater than `media_validation.min_file_size_bytes` (default: 500MB)

This ensures only legitimate video files are selected, excluding samples and extras.

### Additional File Selection

Beyond video files, qDebrid can select additional useful files for your media server:

```yaml
real_debrid:
  additional_selectable_files:
    - "srt"   # SubRip subtitles
    - "sub"   # MicroDVD subtitles
    - "idx"   # VobSub subtitle index
    - "ass"   # Advanced SubStation Alpha subtitles
    - "ssa"   # SubStation Alpha subtitles
    - "smi"   # SAMI subtitles
    - "vtt"   # WebVTT subtitles
    - "nfo"   # Media info files
    - "jpg"   # Images (posters, fanart)
    - "jpeg"  # Images
    - "png"   # Images
    - "tbn"   # Thumbnail images
```

These files are selected **regardless of size**, as they're typically small metadata/subtitle files.

**Benefits:**
- Plex/Jellyfin automatically detect and use local subtitles
- NFO files provide additional metadata
- Local artwork improves media library appearance
- No need to manually download subtitles separately

**Note:** If both `streamable_extensions` and `additional_selectable_files` are empty, **all files** will be selected (not recommended).

## Configuration

### Minimal Configuration

```yaml
server:
  port: 8080

real_debrid:
  token: "YOUR_TOKEN"

qbittorrent:
  save_path: "/media"

logging:
  level: "info"
```

### Full Configuration Reference

See example config generation:

```bash
docker run --rm ghcr.io/yourusername/qdebrid:latest -example-config
```

#### Default Values Reference

Below is a complete configuration file showing all available options with their default values:

```yaml
# qDebrid Configuration - All Default Values
server:
  host: ""                                    # Empty = listen on all interfaces
  port: 8080                                  # HTTP server port

real_debrid:
  token: ""                                   # Required: Your Real-Debrid API token
  requests_per_minute: 20                     # Rate limit (Real-Debrid allows ~60/min)
  max_retries: 10                             # Maximum retry attempts for failed requests
  additional_selectable_files:                # Additional file extensions to select (beyond video files)
    - "srt"                                   # SubRip subtitles
    - "sub"                                   # MicroDVD subtitles
    - "idx"                                   # VobSub subtitle index
    - "ass"                                   # Advanced SubStation Alpha subtitles
    - "ssa"                                   # SubStation Alpha subtitles
    - "smi"                                   # SAMI subtitles
    - "vtt"                                   # WebVTT subtitles
    - "nfo"                                   # Media info files
    - "jpg"                                   # Images (posters, fanart)
    - "jpeg"                                  # Images
    - "png"                                   # Images
    - "tbn"                                   # Thumbnail images

qbittorrent:
  category_name: "qdebrid"                    # Category name shown in *Arr apps
  save_path: ""                               # Required: Path where media is saved
  validate_paths: false                       # Verify files exist on disk

media_validation:
  enabled: false                              # Enable media validation with ffprobe
  require_downloaded: false                   # Reject if torrent status is not 'downloaded'
  streamable_extensions:                      # File extensions to select and validate
    - "mkv"
    - "mp4"
    - "avi"
    - "m4v"
    - "mov"
    - "wmv"
    - "webm"
  min_file_size_bytes: 524288000              # Minimum file size (500MB) for video selection
  require_video_stream: false                 # Reject if no video stream found
  require_audio_stream: false                 # Reject if no audio stream found
  min_duration_seconds: 0                     # Minimum video duration (0 = disabled)
  ffprobe_timeout: 30                         # FFprobe timeout in seconds
  reject_sample_files: false                  # Reject sample files based on duration
  sample_min_runtime: 300                     # Minimum runtime in seconds (5 minutes)
  validate_file_count: false                  # Validate expected number of video files from *arr

data:
  directory: "./data"                         # Directory for persistent data (history, state)
  cleanup_max_age: "168h"                     # Remove old history entries after this duration (7 days)
  auto_save_interval: "5m"                    # How often to auto-save history to disk

logging:
  level: "info"                               # Log level: debug, info, warn, error
  output_path: "stdout"                       # Output: stdout, stderr, or file path
  json: false                                 # Output logs in JSON format
```

**Note**: Only `real_debrid.token` and `qbittorrent.save_path` are required. All other fields will use the defaults shown above if not specified.

Key settings:

- `real_debrid.requests_per_minute`: Rate limit (default: 20, RD allows ~60)
- `real_debrid.additional_selectable_files`: Extra file types to select (subtitles, nfo, images)
- `media_validation.enabled`: Enable quality validation with ffprobe
- `media_validation.validate_file_count`: Enable episode/movie count validation
- `qbittorrent.validate_paths`: Check if files exist at save_path

For detailed validation configuration, see [docs/VALIDATION.md](docs/VALIDATION.md).

## Docker Configuration

### Environment Variables

Override config values using environment variables:

```bash
docker run -d \
  -e QDEBRID_REAL_DEBRID_TOKEN=your_token \
  -e QDEBRID_LOGGING_LEVEL=debug \
  -e QDEBRID_SERVER_PORT=8080 \
  ghcr.io/yourusername/qdebrid:latest
```

### Volumes

- `/config/config.yml` - Configuration file (required)
- `/media` - Media directory for path validation (optional)

### Health Check

The container includes automatic health monitoring:

```bash
docker inspect --format='{{.State.Health.Status}}' qdebrid
```

Health endpoint: `GET /health`

### Docker Compose Example

```yaml
version: '3.8'

services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    container_name: qdebrid
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config.yml:/config/config.yml:ro
      - /mnt/media:/media
    environment:
      - TZ=UTC
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - media-stack

  sonarr:
    image: linuxserver/sonarr:4.0.11
    container_name: sonarr
    ports:
      - "8989:8989"
    volumes:
      - ./sonarr:/config
      - /mnt/media:/media
    networks:
      - media-stack

networks:
  media-stack:
    driver: bridge
```

## Building from Source

### Local Development

```bash
# Install dependencies
make install

# Build binary
make build

# Generate example config
./build/qdebrid -example-config > config.yml

# Edit config with your token
nano config.yml

# Run
make run
```

### Build Docker Image

```bash
# Build for local architecture
make docker-build

# Or manually
docker build -t qdebrid:latest .

# Run
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  qdebrid:latest
```

## Architecture

```
cmd/qdebrid/              # Application entry point
internal/
  cache/                  # In-memory cache with TTL
  config/                 # Configuration management
  debrid/                 # Real-Debrid client wrapper
  history/                # Download history tracking
  logger/                 # Structured logging setup
  mediavalidation/        # FFprobe media validation
  qbittorrent/            # qBittorrent API handlers
  servarr/                # Sonarr/Radarr integration
  torrent/                # Torrent service & validation
pkg/
  retry/                  # Retry & rate limiting logic
```

### Key Components

- **Retry Package**: Exponential backoff with jitter, token bucket rate limiting
- **Debrid Client**: Wraps Real-Debrid API with retry queue
- **Media Validator**: FFprobe integration for quality checks
- **Torrent Service**: Orchestrates validation workflows
- **qBittorrent Handler**: HTTP API compatible with qBittorrent

## Rate Limiting

- **Token Bucket Algorithm**: Refills at configured rate
- **Request Queue**: All operations wait for token availability
- **Burst Handling**: Allows brief spikes (3 requests)
- **Automatic Retry on 429**: Exponential backoff on rate limits
- **Conservative Default**: 20 req/min (RD allows ~60)

## Logging

Log levels: `debug`, `info`, `warn`, `error`

Example logs:
```
INFO  qDebrid starting version=2.0.0
INFO  starting server addr=:8080
INFO  http request method=POST path=/api/v2/torrents/add status=200
INFO  validating torrent media torrent_id=ABC123
INFO  file validation passed file=episode1.mkv duration=2400s
INFO  torrent validation completed successfully validated_files=2
```

## API Endpoints

Compatible with qBittorrent Web API v2:

- `POST /api/v2/auth/login` - Authentication
- `GET /api/v2/app/webapiVersion` - API version
- `GET /api/v2/app/preferences` - App preferences
- `GET /api/v2/torrents/categories` - List categories
- `POST /api/v2/torrents/add` - Add torrent
- `GET /api/v2/torrents/info` - List torrents
- `POST /api/v2/torrents/delete` - Delete torrent
- `GET /health` - Health check

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs qdebrid

# Common issues:
# - Missing config.yml
# - Invalid Real-Debrid token
# - server.host must be "0.0.0.0" in Docker
```

### Media validation fails

```bash
# Check ffprobe is available (included in Docker image)
docker exec qdebrid which ffprobe

# Increase timeout for large files
# In config.yml:
media_validation:
  ffprobe_timeout: 60
```

### File count validation skipped

```bash
# Ensure *arr credentials are configured in download client:
# Username: http://localhost:8989 (your Sonarr URL)
# Password: your_sonarr_api_key
```

### Rate limiting errors

```bash
# Reduce request rate in config.yml:
real_debrid:
  requests_per_minute: 15
```

## Documentation

- [Docker Deployment Guide](docs/DOCKER.md) - Advanced Docker configuration
- [Media Validation Guide](docs/VALIDATION.md) - Complete validation documentation

## Development

```bash
make help           # Show all commands
make test           # Run tests
make fmt            # Format code
make lint           # Run linter
make clean          # Clean build artifacts
```

## License

GNU GPLv3 - See LICENSE file

## Acknowledgments

- [real_debrid_go](https://github.com/sushydev/real_debrid_go) - Real-Debrid API client
- [zap](https://github.com/uber-go/zap) - Structured logging
- [qBittorrent Web API](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))

## Links

- [Real-Debrid API](https://api.real-debrid.com)
- [Sonarr API](https://sonarr.tv/docs/api)
- [Radarr API](https://radarr.video/docs/api)
