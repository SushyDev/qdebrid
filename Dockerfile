# Stage 1: Build Stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o qdebrid ./cmd/qdebrid

# Stage 2: Runtime Stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 qdebrid && \
    adduser -D -u 1000 -G qdebrid qdebrid

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=qdebrid:qdebrid /app/qdebrid /app/qdebrid

# Create config directory
RUN mkdir -p /config && chown qdebrid:qdebrid /config

# Switch to non-root user
USER qdebrid

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/qdebrid"]
CMD ["--config", "/config/config.yml"]
