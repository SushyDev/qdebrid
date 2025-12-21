# Media Validation Guide

qDebrid v2.1+ includes powerful media validation features to ensure only high-quality, complete releases are imported into your media library.

## Table of Contents

- [Overview](#overview)
- [Quality Validation](#quality-validation)
- [File Count Validation](#file-count-validation)
- [Configuration](#configuration)
- [How It Works](#how-it-works)
- [Troubleshooting](#troubleshooting)

## Overview

Media validation happens automatically when torrents are added through the qBittorrent API. There are two types of validation:

1. **Quality Validation** (v2.1): Uses ffprobe to verify media file quality
2. **File Count Validation** (v2.2): Ensures torrents contain the expected number of episodes/movies

Both validations are optional and can be enabled independently.

## Quality Validation

Quality validation uses ffprobe to inspect video files and reject problematic releases.

### Features

- **Video Stream Detection**: Ensures file has a valid video stream
- **Audio Stream Detection**: Ensures file has a valid audio stream
- **Duration Checks**: Reject files shorter than a minimum duration
- **Sample File Detection**: Automatically reject sample files based on duration
- **Codec Validation**: Extracts codec information for logging
- **Fail-Fast**: First validation failure immediately rejects the torrent

### Configuration

```yaml
media_validation:
  enabled: true                       # Enable quality validation
  require_downloaded: true            # Wait for torrent to finish downloading
  streamable_extensions:              # File types to validate
    - "mkv"
    - "mp4"
    - "avi"
    - "m4v"
    - "mov"
    - "wmv"
    - "webm"
  require_video_stream: true          # Reject if no video stream
  require_audio_stream: true          # Reject if no audio stream
  min_duration_seconds: 0             # Minimum duration (0 = disabled)
  ffprobe_timeout: 30                 # Timeout per file
  reject_sample_files: true           # Reject samples
  sample_min_runtime: 300             # Min runtime for non-samples (5 min)
```

### How It Works

1. **Torrent Added**: *arr sends torrent to qDebrid via qBittorrent API
2. **Wait for Download**: If `require_downloaded: true`, waits for Real-Debrid to finish
3. **File Selection**: Filters to only streamable file extensions
4. **FFprobe Analysis**: For each file:
   - Unrestricts download link from Real-Debrid
   - Runs ffprobe to analyze media streams
   - Extracts video/audio codec, duration, resolution
5. **Validation Checks**: Applies all configured rules
6. **Rejection**: If any file fails, entire torrent is deleted and *arr is notified

### Sample File Detection

Sample files are detected in two ways:

1. **By Name**: Files containing "sample", "trailer", "preview", or "demo"
2. **By Duration**: Files shorter than `sample_min_runtime` seconds

### Requirements

- **ffprobe** must be installed (included in Docker images)
- **Internet access** to unrestrict Real-Debrid links
- Sufficient **timeout** settings for large files

### Example Logs

```
INFO  validating torrent media torrent_id=ABC123
INFO  found streamable files to validate count=2
INFO  file validation passed file=episode1.mkv duration=2400s video_codec=h264 audio_codec=aac
INFO  file validation passed file=episode2.mkv duration=2350s video_codec=h264 audio_codec=aac
INFO  torrent validation completed successfully torrent_id=ABC123 validated_files=2
```

## File Count Validation

File count validation ensures torrents contain the expected number of video files, preventing incomplete releases.

### Features

- **Sonarr Integration**: Compares file count against expected episodes
- **Radarr Integration**: Ensures at least 1 movie file exists
- **Season Pack Validation**: Perfect for validating multi-episode torrents
- **Sample Exclusion**: Automatically excludes sample files from count
- **Queue API Integration**: Queries *arr for expected file counts
- **Graceful Fallback**: Skips validation if *arr is unreachable

### Configuration

```yaml
media_validation:
  enabled: true                       # Required for file count validation
  validate_file_count: true           # Enable file count validation
  reject_sample_files: true           # Recommended: exclude samples from count
  sample_min_runtime: 300             # Files < 5 min are samples
```

### How It Works

1. **Torrent Added**: *arr sends torrent with hash via qBittorrent API
2. **Extract Credentials**: Parse *arr host and API key from auth header
3. **Query Queue API**: GET `/api/v3/queue?downloadId={hash}`
4. **Determine Expected Count**:
   - **Sonarr**: `len(EpisodeIDs)` from queue entry
   - **Radarr**: Always `1` (single movie)
5. **Count Video Files**: Count non-sample video files in torrent
6. **Validate**: Reject if `actual < expected`

### Correlation Mechanism

qDebrid correlates torrents with *arr requests using:

1. **Torrent Hash**: Real-Debrid provides the infohash
2. **Download ID**: *arr records this hash in its queue
3. **Queue Query**: Match by `downloadId` parameter

### Sample Exclusion

Files are excluded from count if they match:

- **Name patterns**: "sample", "trailer", "preview", "demo"
- **Duration** (if enabled): Shorter than `sample_min_runtime`

### Example Scenarios

#### Scenario 1: Complete Season Pack
```
Sonarr requests: S01E01-E10 (10 episodes)
Torrent contains: 10 MKV files + 1 sample
Expected: 10
Actual: 10 (sample excluded)
Result: ✓ ACCEPTED
```

#### Scenario 2: Incomplete Season Pack
```
Sonarr requests: S01E01-E10 (10 episodes)
Torrent contains: 8 MKV files
Expected: 10
Actual: 8
Result: ✗ REJECTED - "insufficient video files: expected at least 10, found 8"
```

#### Scenario 3: Movie with Extras
```
Radarr requests: Movie (1 file)
Torrent contains: movie.mkv + trailer.mkv + sample.mkv
Expected: 1
Actual: 1 (trailer and sample excluded)
Result: ✓ ACCEPTED
```

#### Scenario 4: Fake Release
```
Sonarr requests: S01E01-E05 (5 episodes)
Torrent contains: 3 sample files only
Expected: 5
Actual: 0 (all samples)
Result: ✗ REJECTED - "insufficient video files: expected at least 5, found 0"
```

### Requirements

- `validate_file_count: true` in configuration
- *arr application must be accessible from qDebrid
- Proper authentication setup (see below)

### Authentication Setup

File count validation requires qDebrid to access your *arr API:

1. In *arr, go to **Settings → Download Clients**
2. Add qBittorrent client with:
   - **Host**: qDebrid host
   - **Port**: qDebrid port
   - **Username**: Your *arr URL (e.g., `http://localhost:8989`)
   - **Password**: Your *arr API Key
   - **Category**: `qdebrid`

The username/password fields are repurposed to pass *arr credentials to qDebrid.

### Graceful Degradation

File count validation fails gracefully:

- **Auth parse fails**: Validation skipped (maybe not from *arr)
- **Queue query fails**: Validation skipped (service unreachable)
- **No queue entry**: Validation skipped (not tracked yet)
- **No seriesId/movieId**: Validation skipped (unknown type)

This ensures the feature doesn't break existing functionality.

### Example Logs

```
INFO  validating Sonarr file count torrent_id=ABC123 downloadId=1234ABCD expected_episodes=10
DEBUG counting valid video files torrent_id=ABC123
DEBUG skipping sample file torrent_id=ABC123 file=sample-video.mkv
DEBUG counted valid video files torrent_id=ABC123 count=10
INFO  torrent file count validation passed torrent_id=ABC123 expected=10 actual=10
```

## Combined Validation

Both validations can work together for maximum quality assurance:

```yaml
media_validation:
  # Quality validation
  enabled: true
  require_downloaded: true
  require_video_stream: true
  require_audio_stream: true
  reject_sample_files: true
  sample_min_runtime: 300
  
  # File count validation
  validate_file_count: true
```

**Validation Order:**
1. Quality validation (ffprobe checks each file)
2. File count validation (compares count vs expected)

If either fails, the torrent is rejected.

## Configuration Examples

### Minimal Quality Validation
```yaml
media_validation:
  enabled: true
  require_video_stream: true
  require_audio_stream: true
```

### Aggressive Quality + File Count
```yaml
media_validation:
  enabled: true
  require_downloaded: true
  require_video_stream: true
  require_audio_stream: true
  min_duration_seconds: 120           # At least 2 minutes
  reject_sample_files: true
  sample_min_runtime: 300
  validate_file_count: true
```

### File Count Only (No Quality Checks)
```yaml
media_validation:
  enabled: true
  validate_file_count: true
  require_video_stream: false
  require_audio_stream: false
```

## Troubleshooting

### Quality Validation Issues

#### "ffprobe failed: executable file not found"
- **Solution**: Install ffmpeg/ffprobe on your system
- **Docker**: Included in official images

#### "ffprobe timeout"
- **Cause**: Large files or slow connection
- **Solution**: Increase `ffprobe_timeout` to 60+ seconds

#### "no video stream found"
- **Cause**: File is corrupt or not a video
- **Solution**: This is expected behavior - torrent is correctly rejected

### File Count Validation Issues

#### "skipping file count validation: failed to parse auth header"
- **Cause**: *arr credentials not in qBittorrent username/password fields
- **Solution**: Configure download client correctly (see Authentication Setup)

#### "skipping file count validation: failed to query queue"
- **Cause**: *arr API unreachable
- **Solution**: Check *arr is running and URL/API key are correct

#### "no queue records found for download"
- **Cause**: Torrent hash not found in *arr queue
- **Possible reasons**:
  - Torrent was added manually (not from *arr)
  - Queue entry was already processed/removed
  - Hash mismatch between Real-Debrid and *arr
- **Solution**: This is normal for manual adds; validation is skipped

#### False rejections (count too low)
- **Cause**: Sample files being counted as real files
- **Solution**: Enable `reject_sample_files: true`

#### False acceptances (count too high)
- **Cause**: Torrent has extra episodes not in request
- **Note**: This is expected - we check for minimum count, not exact match

## Performance Impact

### Quality Validation
- **Time**: ~2-10 seconds per file (depends on file size and connection)
- **Network**: Downloads first few MB of each file for analysis
- **CPU**: Minimal (ffprobe is efficient)

### File Count Validation
- **Time**: <1 second (simple API calls)
- **Network**: Minimal (small JSON responses)
- **CPU**: Negligible

### Recommendations
- Enable `require_downloaded: true` to avoid re-validating during download
- Set reasonable `ffprobe_timeout` (30s default is good)
- Use `reject_sample_files` to avoid validating sample files

## API Endpoints Used

### Sonarr Queue API
```
GET /api/v3/queue?downloadId={hash}&includeEpisode=true
```

**Response:**
```json
{
  "records": [{
    "downloadId": "1234ABCD...",
    "seriesId": 1,
    "episodeIds": [1, 2, 3],
    "episodes": [...]
  }]
}
```

### Radarr Queue API
```
GET /api/v3/queue?downloadId={hash}&includeMovie=true
```

**Response:**
```json
{
  "records": [{
    "downloadId": "1234ABCD...",
    "movieId": 1,
    "movie": {...}
  }]
}
```

## Contributing

Found a bug or have a suggestion? Please open an issue on GitHub!

## License

GNU GPLv3 - See LICENSE file
