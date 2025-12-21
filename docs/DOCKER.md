# Docker Deployment Guide

This guide covers advanced Docker deployment scenarios. For quick start instructions, see the main [README.md](../README.md#quick-start-with-docker).

## Table of Contents

- [Environment Variables](#environment-variables)
- [Volumes and Persistence](#volumes-and-persistence)
- [Networking](#networking)
- [Building from Source](#building-from-source)
- [Security Best Practices](#security-best-practices)
- [Advanced Configuration](#advanced-configuration)
- [Troubleshooting](#troubleshooting)

## Environment Variables

Override any configuration value using environment variables with the `QDEBRID_` prefix:

```bash
docker run -d \
  -e QDEBRID_SERVER_HOST=0.0.0.0 \
  -e QDEBRID_SERVER_PORT=8080 \
  -e QDEBRID_REAL_DEBRID_TOKEN=your_token \
  -e QDEBRID_REAL_DEBRID_REQUESTS_PER_MINUTE=20 \
  -e QDEBRID_LOGGING_LEVEL=info \
  -e QDEBRID_LOGGING_JSON=true \
  -e TZ=America/New_York \
  ghcr.io/yourusername/qdebrid:latest
```

### Environment Variable Format

Configuration hierarchy is flattened using underscores:

```yaml
# config.yml
real_debrid:
  token: "value"
  
# Environment variable
QDEBRID_REAL_DEBRID_TOKEN=value
```

## Volumes and Persistence

### Required Volumes

- `/config/config.yml` - Configuration file (mount as read-only for security)

```bash
-v $(pwd)/config.yml:/config/config.yml:ro
```

### Optional Volumes

- `/media` - Media directory for path validation (if `qbittorrent.validate_paths: true`)

```bash
-v /mnt/media:/media
```

### Volume Permissions

The container runs as user `qdebrid` (UID/GID 1000 by default). Ensure volumes have appropriate permissions:

```bash
# Check permissions
ls -la config.yml

# Fix permissions if needed
chown 1000:1000 config.yml
chmod 644 config.yml
```

To use a different UID/GID:

```bash
docker run -d \
  --user 1001:1001 \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  ghcr.io/yourusername/qdebrid:latest
```

## Networking

### Port Mapping

Default port is `8080`. Map to a different host port:

```bash
-p 9090:8080  # Host port 9090 → Container port 8080
```

### Shared Networks for *Arr Stack

Create a dedicated network for your media stack:

```bash
docker network create media-stack
```

```yaml
version: '3.8'

services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    networks:
      - media-stack

  sonarr:
    image: linuxserver/sonarr:4.0.11
    networks:
      - media-stack

  radarr:
    image: linuxserver/radarr:5.14.0
    networks:
      - media-stack

networks:
  media-stack:
    external: true
```

Use container names as hostnames: `http://qdebrid:8080`

### Host Networking

For direct host network access (not recommended for security):

```bash
docker run -d \
  --network host \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  ghcr.io/yourusername/qdebrid:latest
```

**Note**: With host networking, set `server.host: "0.0.0.0"` and `server.port` as desired in config.yml.

## Building from Source

### Local Build

```bash
# Using Makefile
make docker-build

# Manual build
docker build -t qdebrid:latest .

# Build with custom Dockerfile
docker build -f Dockerfile.custom -t qdebrid:custom .
```

### Multi-Architecture Build

Build for multiple platforms using Docker Buildx:

```bash
# Setup buildx (one time)
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap

# Build for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t ghcr.io/yourusername/qdebrid:latest \
  --push .
```

### Build Arguments

Pass build-time variables:

```bash
docker build \
  --build-arg GO_VERSION=1.23.2 \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t qdebrid:latest .
```

## Security Best Practices

### 1. Read-Only Root Filesystem

Run with read-only root filesystem:

```bash
docker run -d \
  --read-only \
  --tmpfs /tmp \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  ghcr.io/yourusername/qdebrid:latest
```

### 2. Drop Capabilities

Remove unnecessary Linux capabilities:

```bash
docker run -d \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  ghcr.io/yourusername/qdebrid:latest
```

### 3. Use Docker Secrets

For Docker Swarm or Compose with secrets:

```yaml
version: '3.8'

services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    secrets:
      - rd_token
    environment:
      - QDEBRID_REAL_DEBRID_TOKEN_FILE=/run/secrets/rd_token

secrets:
  rd_token:
    external: true
```

### 4. Non-Root User

Container already runs as non-root user (UID 1000). To verify:

```bash
docker exec qdebrid id
# Output: uid=1000(qdebrid) gid=1000(qdebrid)
```

### 5. Security Scanning

Scan images for vulnerabilities:

```bash
# Using Docker Scout
docker scout cve ghcr.io/yourusername/qdebrid:latest

# Using Trivy
trivy image ghcr.io/yourusername/qdebrid:latest
```

## Advanced Configuration

### Resource Limits

Limit CPU and memory usage:

```yaml
services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 128M
```

Or with Docker run:

```bash
docker run -d \
  --cpus="1.0" \
  --memory="512m" \
  --memory-reservation="128m" \
  ghcr.io/yourusername/qdebrid:latest
```

### Custom Health Check

Override default health check:

```yaml
services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

Disable health check:

```yaml
healthcheck:
  disable: true
```

### Restart Policies

```yaml
services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    restart: unless-stopped  # Restart unless manually stopped
```

Options:
- `no` - Never restart
- `always` - Always restart
- `on-failure` - Restart on non-zero exit
- `unless-stopped` - Restart unless manually stopped (recommended)

### Logging Configuration

Configure Docker logging driver:

```yaml
services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

Or use syslog:

```yaml
logging:
  driver: "syslog"
  options:
    syslog-address: "tcp://192.168.1.100:514"
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs qdebrid

# Common issues:
# 1. Missing config.yml
ls -la $(pwd)/config.yml

# 2. Wrong server.host (must be 0.0.0.0 for Docker)
grep "host:" config.yml

# 3. Port already in use
ss -tlnp | grep 8080

# 4. Permission denied
docker run --rm -v $(pwd)/config.yml:/tmp/test.yml:ro alpine cat /tmp/test.yml
```

### Health Check Failing

```bash
# Test health endpoint manually
docker exec qdebrid wget -q -O- http://localhost:8080/health

# Check container health status
docker inspect --format='{{json .State.Health}}' qdebrid | jq

# View health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' qdebrid
```

### Cannot Connect to *Arr Apps

```bash
# 1. Check if containers are on same network
docker network inspect media-stack

# 2. Test connectivity from qDebrid to Sonarr
docker exec qdebrid wget -q -O- http://sonarr:8989/api/v3/system/status?apikey=YOUR_API_KEY

# 3. Verify *arr credentials in qBittorrent download client config
# Username should be: http://sonarr:8989
# Password should be: your_sonarr_api_key
```

### Media Validation Fails

```bash
# Check if ffprobe is available
docker exec qdebrid which ffprobe
docker exec qdebrid ffprobe -version

# Test ffprobe with a file
docker exec qdebrid ffprobe -v error -show_format -show_streams "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4"

# Increase timeout in config.yml
# media_validation:
#   ffprobe_timeout: 60
```

### High Memory Usage

```bash
# Check container stats
docker stats qdebrid

# If cache is too large, reduce TTL in config
# Or restart container to clear cache
docker restart qdebrid
```

### Rate Limiting Errors

```bash
# Check logs for 429 errors
docker logs qdebrid | grep "429"

# Reduce rate in config.yml:
# real_debrid:
#   requests_per_minute: 15
```

## Container Registry

### GitHub Container Registry (GHCR)

Pull from GHCR:

```bash
# Latest stable release
docker pull ghcr.io/yourusername/qdebrid:latest

# Specific version
docker pull ghcr.io/yourusername/qdebrid:v2.0.0

# Development/nightly builds
docker pull ghcr.io/yourusername/qdebrid:development
```

### Authenticate with GHCR

For private repositories:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

### Image Tags

- `latest` - Latest stable release (recommended for production)
- `v2.0.0` - Specific version tag
- `development` - Latest development build (unstable)
- `sha-abc1234` - Specific commit SHA

## Docker Compose Full Example

Complete docker-compose.yml with all services:

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
      - TZ=America/New_York
      - QDEBRID_LOGGING_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - media-stack
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  sonarr:
    image: linuxserver/sonarr:4.0.11
    container_name: sonarr
    restart: unless-stopped
    ports:
      - "8989:8989"
    volumes:
      - ./sonarr:/config
      - /mnt/media:/media
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=America/New_York
    networks:
      - media-stack

  radarr:
    image: linuxserver/radarr:5.14.0
    container_name: radarr
    restart: unless-stopped
    ports:
      - "7878:7878"
    volumes:
      - ./radarr:/config
      - /mnt/media:/media
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=America/New_York
    networks:
      - media-stack

networks:
  media-stack:
    driver: bridge
```

Start the stack:

```bash
docker-compose up -d
docker-compose logs -f
```

## Makefile Commands

If building from source, use these Makefile targets:

```bash
make docker-build         # Build Docker image
make docker-run           # Run container locally
make docker-compose-up    # Start with docker-compose
make docker-compose-down  # Stop docker-compose
make docker-compose-logs  # View logs
make docker-push          # Push to registry
make docker-buildx        # Multi-arch build
```

## Additional Resources

- [Main README](../README.md) - Quick start and configuration
- [Validation Guide](VALIDATION.md) - Media validation documentation
- [Dockerfile](../Dockerfile) - Container build configuration
- [docker-compose.yml](../docker-compose.yml) - Compose example

## Support

For issues and questions:
- GitHub Issues: https://github.com/yourusername/qdebrid/issues
- Logs: `docker logs qdebrid -f`
- Health: `curl http://localhost:8080/health`
