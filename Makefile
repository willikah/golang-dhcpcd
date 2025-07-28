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
DOCKER_IMAGE := golang-dhcpcd
DOCKER_TAG := latest

.PHONY: all build run test test-unit test-integration test-all lint clean fmt vet deps install generate help docker-build docker-run docker-push

all: build

generate:
	go generate ./...

build: generate
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test ./...

test-unit:
	go test -tags=unit ./...

test-integration: test-docker-build
	@echo "Running integration tests..."
	go test -v ./test -tags=integration
	@echo "Integration tests completed"

test-all: test test-integration
	@echo "All tests completed"

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)/*

# Docker targets
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	docker run --rm -it $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: docker-build
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Test environment targets (delegates to test/Makefile)
.PHONY: test-docker-build test-up test-up-d test-down test-logs test-clean-docker

test-docker-build:
	$(MAKE) -C test docker-build

test-up:
	$(MAKE) -C test up

test-up-d:
	$(MAKE) -C test up-d

test-down:
	$(MAKE) -C test down

test-logs:
	$(MAKE) -C test logs

test-clean-docker:
	$(MAKE) -C test clean-docker

test-status:
	$(MAKE) -C test status

test-restart:
	$(MAKE) -C test restart

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
	@echo "  build               Build the binary"
	@echo "  run                 Run the application"
	@echo "  test                Run all tests"
	@echo "  test-unit           Run unit tests only (tags=unit)"
	@echo "  test-integration    Run integration tests (requires Docker)"
	@echo "  test-all            Run all tests (unit + integration)"
	@echo "  lint                Run golangci-lint"
	@echo "  clean               Remove build artifacts"
	@echo "  fmt                 Format code"
	@echo "  vet                 Run go vet"
	@echo "  deps                Download dependencies"
	@echo "  install             Install the binary"
	@echo "  generate            Generate embedded files"
	@echo "  help                Show this help message"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build        Build Docker image"
	@echo "  docker-run          Build and run Docker image"
	@echo "  docker-push         Build and push Docker image"
	@echo ""
	@echo "Test environment targets:"
	@echo "  test-docker-build   Build Docker images for testing"
	@echo "  test-up             Start test environment"
	@echo "  test-up-d           Start test environment in detached mode"
	@echo "  test-down           Stop test environment"
	@echo "  test-logs           View test container logs"
	@echo "  test-status         Show test container status"
	@echo "  test-restart        Restart test environment"
	@echo "  test-clean-docker   Clean up test Docker resources"
