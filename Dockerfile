# Stage 1: Build Stage
FROM golang:latest AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Search and replace any "replace" lines in the go mod for local development
RUN grep -v '^replace' go.mod > go.mod.tmp && mv go.mod.tmp go.mod

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Search and replace any "replace" lines in the go mod for local development
RUN grep -v '^replace' go.mod > go.mod.tmp && mv go.mod.tmp go.mod

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

# Stage 2: Final Stage
FROM alpine:latest

# Install ca-certificates to handle HTTPS requests
RUN apk add --no-cache ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Run the main binary
ENTRYPOINT ["/app/main"]
