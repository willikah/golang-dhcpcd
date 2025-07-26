# Full-featured Makefile for Go projects
# Usage:
#   make build      - Build the binary
#   make run        - Run the application
#   make test       - Run tests
#   make lint       - Run golangci-lint
#   make clean      - Remove build artifacts
#   make fmt        - Format code
#   make vet        - Run go vet
#   make deps       - Download dependencies
#   make install    - Install the binary
#   make help       - Show help

BINARY_NAME := golang-dhcpcd
SRC := $(shell find . -type f -name '*.go' -not -path './vendor/*')
VERSION := $(shell git describe --tags --always --dirty)
BUILD_DIR := build

.PHONY: all build run test test-integration test-all lint clean fmt vet deps install generate help

all: build

generate:
	go generate ./...

build: generate
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test ./...

test-integration: docker-build
	@echo "Running integration tests..."
	go test -v ./test -tags=integration
	@echo "Integration tests completed"

test-all: test test-integration
	@echo "All tests completed"

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)/*

.PHONY: up logs down docker-build up-d

docker-build:
	cd test && docker compose build

up:
	cd test && docker compose up

up-d:
	cd test && docker compose up -d

down:
	cd test && docker compose down --remove-orphans

logs:
	cd test && docker compose logs -f client

fmt:
	go fmt ./...

vet: generate
	go vet ./...

deps:
	go mod tidy

install: build
	go install ./...

help:
	@echo "Available targets:"
	@echo "  build           Build the binary"
	@echo "  run             Run the application"
	@echo "  test            Run unit tests"
	@echo "  test-integration Run integration tests (requires Docker)"
	@echo "  test-all        Run all tests (unit + integration)"
	@echo "  lint            Run golangci-lint (auto-installs if missing)"
	@echo "  clean           Remove build artifacts"
	@echo "  fmt             Format code"
	@echo "  vet             Run go vet"
	@echo "  deps            Download dependencies"
	@echo "  install         Install the binary"
	@echo "  help            Show this help message"
	@echo "  docker-build    Build Docker images"
	@echo "  up              Start docker-compose"
	@echo "  down            Stop docker-compose"
	@echo "  logs            View client logs"
