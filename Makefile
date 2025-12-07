# Makefile for qDebrid v2

.PHONY: all build clean test run dev install help docker-build docker-run docker-compose-up docker-compose-down docker-push

# Build variables
BINARY_NAME=qdebrid
BUILD_DIR=build
CMD_DIR=cmd/qdebrid
VERSION?=2.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Docker variables
DOCKER_IMAGE=qdebrid
DOCKER_TAG=$(VERSION)
DOCKER_REGISTRY=ghcr.io
DOCKER_REPO=$(DOCKER_REGISTRY)/$(shell echo $(GITHUB_REPOSITORY) | tr '[:upper:]' '[:lower:]')

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

all: clean build

## build: Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Clean build files
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

## coverage: Show test coverage
coverage: test
	$(GOCMD) tool cover -html=coverage.out

## run: Run the application (requires config.yml)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run with example config
dev:
	@echo "Running in development mode..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	./$(BUILD_DIR)/$(BINARY_NAME) -config=config.example.yml

## install: Install dependencies
install:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies installed"

## example-config: Generate example configuration
example-config: build
	./$(BUILD_DIR)/$(BINARY_NAME) -example-config

## write-config: Write example config to config.yml
write-config: build
	./$(BUILD_DIR)/$(BINARY_NAME) -write-config config.yml

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 \
		-v $(PWD)/config.yml:/config/config.yml:ro \
		--name $(DOCKER_IMAGE) \
		$(DOCKER_IMAGE):latest -config /config/config.yml

## docker-test: Run all tests including integration tests in Docker
docker-test:
	@echo "Running tests in Docker container..."
	docker run --rm \
		-v $(PWD):/src \
		-w /src \
		golang:1.25.4-alpine \
		sh -c "apk add --no-cache git make gcc musl-dev && go test -v -race -coverprofile=coverage.out ./..."
	@echo "Tests complete"

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d
	@echo "Services started. View logs with: docker-compose logs -f"

## docker-compose-down: Stop services
docker-compose-down:
	@echo "Stopping services..."
	docker-compose down

## docker-compose-logs: View docker-compose logs
docker-compose-logs:
	docker-compose logs -f

## docker-push: Push Docker image to registry
docker-push:
	@echo "Pushing Docker image to registry..."
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REPO):$(DOCKER_TAG)
	docker tag $(DOCKER_IMAGE):latest $(DOCKER_REPO):latest
	docker push $(DOCKER_REPO):$(DOCKER_TAG)
	docker push $(DOCKER_REPO):latest
	@echo "Docker image pushed to $(DOCKER_REPO)"

## docker-buildx: Build multi-arch Docker image
docker-buildx:
	@echo "Building multi-arch Docker image..."
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		--push .

## help: Show this help
help:
	@echo "qDebrid v2 - Makefile commands:"
	@echo ""
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'
