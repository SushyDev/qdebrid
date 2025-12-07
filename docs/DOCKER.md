# Docker Deployment Guide

## Quick Start

### Using Docker Run

```bash
# Create config file
cp config.docker.yml config.yml
# Edit config.yml and add your Real-Debrid token

# Run container
docker run -d \
  --name qdebrid \
  -p 8080:8080 \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  -v /path/to/media:/media \
  --restart unless-stopped \
  ghcr.io/yourusername/qdebrid:latest
```

### Using Docker Compose

```bash
# Copy example config
cp config.docker.yml config.yml

# Edit config.yml with your settings

# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## Configuration

### Config File

Mount your `config.yml` to `/config/config.yml` in the container:

```yaml
server:
  host: "0.0.0.0"  # Important: Must be 0.0.0.0 for Docker
  port: 8080

real_debrid:
  token: "your-token-here"
  requests_per_minute: 60
  # ... other settings

qbittorrent:
  save_path: "/media"  # Path inside container
  # ... other settings
```

### Environment Variables

You can override config values using environment variables:

```bash
docker run -d \
  -e QDEBRID_SERVER_HOST=0.0.0.0 \
  -e QDEBRID_SERVER_PORT=8080 \
  -e QDEBRID_REAL_DEBRID_TOKEN=your-token \
  -e QDEBRID_LOGGING_LEVEL=info \
  ghcr.io/yourusername/qdebrid:latest
```

## Health Check

The container includes a health check endpoint at `/health`:

```bash
# Check health
curl http://localhost:8080/health

# Response
{
  "status": "ok",
  "service": "qdebrid",
  "version": "2.0.0"
}
```

Docker will automatically monitor this endpoint:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' qdebrid
```

## Volumes

### Required

- `/config/config.yml` - Configuration file (read-only recommended)

### Optional

- `/media` - Media directory for path validation

Example with media volume:

```bash
docker run -d \
  --name qdebrid \
  -p 8080:8080 \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  -v /mnt/media:/media \
  ghcr.io/yourusername/qdebrid:latest
```

## Networking

### Ports

- `8080` - HTTP API (default, configurable)

### Network Mode

For *Arr stack integration, use a shared network:

```yaml
version: '3.8'

services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    networks:
      - arr-stack

  sonarr:
    image: linuxserver/sonarr
    networks:
      - arr-stack

networks:
  arr-stack:
    external: true
```

## Building from Source

### Local Build

```bash
# Build image
make docker-build

# Or manually
docker build -t qdebrid:latest .
```

### Multi-Architecture Build

```bash
# Build for amd64 and arm64
make docker-buildx

# Or manually
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t qdebrid:latest \
  --push .
```

## Makefile Commands

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Start with docker-compose
make docker-compose-up

# Stop docker-compose
make docker-compose-down

# View logs
make docker-compose-logs

# Push to registry
make docker-push

# Build multi-arch image
make docker-buildx
```

## Security

### Non-Root User

The container runs as a non-root user (`qdebrid:qdebrid`, UID/GID 1000) for security.

### Read-Only Config

Mount config as read-only:

```bash
-v $(pwd)/config.yml:/config/config.yml:ro
```

### Secrets

Do not include sensitive data in the image. Use:
1. Volume-mounted config file
2. Environment variables
3. Docker secrets (Swarm mode)

## Troubleshooting

### Container Won't Start

Check logs:

```bash
docker logs qdebrid
```

Common issues:
- Missing or invalid `config.yml`
- Wrong `server.host` (must be `0.0.0.0` for Docker)
- Port already in use

### Health Check Failing

```bash
# Check health endpoint manually
docker exec qdebrid wget -q -O- http://localhost:8080/health

# View container health status
docker inspect --format='{{json .State.Health}}' qdebrid | jq
```

### Permission Issues

Ensure volumes have correct permissions:

```bash
# If using a specific UID/GID
docker run -d \
  --user 1000:1000 \
  -v $(pwd)/config.yml:/config/config.yml:ro \
  qdebrid:latest
```

### Cannot Connect to *Arr Apps

1. Check network configuration
2. Ensure containers are on the same network
3. Use container name as hostname: `http://qdebrid:8080`

## Integration with *Arr Stack

### Sonarr/Radarr Configuration

1. **Add Download Client**:
   - Type: qBittorrent
   - Host: `qdebrid` (container name) or `localhost` (if same host)
   - Port: `8080`
   - Username: `<servarr-url>`
   - Password: `<servarr-api-key>`

2. **Docker Compose Example**:

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
    networks:
      - media

  sonarr:
    image: linuxserver/sonarr
    container_name: sonarr
    restart: unless-stopped
    ports:
      - "8989:8989"
    volumes:
      - ./sonarr:/config
      - /mnt/media:/media
    networks:
      - media

networks:
  media:
    driver: bridge
```

## GitHub Container Registry

### Pull Image

```bash
# Latest release
docker pull ghcr.io/yourusername/qdebrid:latest

# Specific version
docker pull ghcr.io/yourusername/qdebrid:2.0.0

# Development build
docker pull ghcr.io/yourusername/qdebrid:development
```

### Authentication

For private repositories:

```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Pull image
docker pull ghcr.io/yourusername/qdebrid:latest
```

## Advanced Configuration

### Custom Logging

```bash
docker run -d \
  -e QDEBRID_LOGGING_LEVEL=debug \
  -e QDEBRID_LOGGING_JSON=true \
  ghcr.io/yourusername/qdebrid:latest
```

### Resource Limits

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
          cpus: '0.5'
          memory: 256M
```

### Health Check Customization

```yaml
services:
  qdebrid:
    image: ghcr.io/yourusername/qdebrid:latest
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 60s
      timeout: 10s
      retries: 3
      start_period: 10s
```

## Support

- **Issues**: https://github.com/yourusername/qdebrid/issues
- **Documentation**: https://github.com/yourusername/qdebrid/blob/main/README.md
- **Health Check**: `GET /health`
- **Logs**: `docker logs qdebrid -f`
