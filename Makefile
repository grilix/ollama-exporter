# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary names
BINARY_NAME=ollama-exporter
BINARY_UNIX=$(BINARY_NAME)_unix

# Build information
VERSION ?= dev
SHA ?= unknown
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS=-ldflags "-X ollama-exporter/cmd.version=$(VERSION) -X ollama-exporter/cmd.sha=$(SHA) -X ollama-exporter/cmd.buildTm=$(BUILD_TIME)"

# Directories
CMD_DIR=.
INTERNAL_DIR=./internal
PKG_DIR=./pkg
TEST_DIR=./test

# Docker parameters
DOCKER_IMAGE=ollama-exporter
DOCKER_TAG=latest

.PHONY: help build clean test coverage fmt lint vet deps docker docker-run docker-build docker-push run install

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  coverage       - Run tests with coverage"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run linter"
	@echo "  vet            - Run go vet"
	@echo "  deps           - Download dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-push    - Push Docker image"
	@echo "  run            - Run the application"
	@echo "  install        - Install the binary"

# Build targets
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_DIR)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) $(CMD_DIR)

build-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Clean targets
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out

# Test targets
test:
	$(GOTEST) ./...

test-integration:
	$(GOTEST) -tags=integration ./test/integration/...

test-unit:
	$(GOTEST) ./internal/... ./pkg/...

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Code quality targets
fmt:
	$(GOFMT) -s -w .

lint:
	$(GOLINT) run ./...

vet:
	$(GOCMD) vet ./...

check: fmt vet lint test

# Dependency targets
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker targets
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run:
	docker run --rm -p 8000:8000 -e OLLAMA_HOST=$(OLLAMA_HOST) $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Development targets
run:
	$(GOCMD) run $(CMD_DIR)

run-dev:
	OLLAMA_HOST=localhost:11434 EXPORTER_PORT=8000 $(GOCMD) run $(CMD_DIR)

install: build
	cp $(BINARY_NAME) /usr/local/bin/

# Development setup
dev-setup:
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Release targets
release: clean test build build-linux

# CI targets
ci: deps fmt vet lint test

# Local development
dev: fmt vet test run

# Generate mocks (if using mockgen)
mocks:
	mockgen -source=internal/interfaces/interfaces.go -destination=test/mocks/interfaces.go

# Security scan
security:
	gosec ./...

# Performance profiling
profile:
	$(GOTEST) -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...

# Documentation
docs:
	godoc -http=:6060

# Version info
version:
	@echo "Version: $(VERSION)"
	@echo "SHA: $(SHA)"
	@echo "Build Time: $(BUILD_TIME)"
