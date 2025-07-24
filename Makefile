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

.PHONY: all build run test lint clean fmt vet deps install generate help

all: build

generate:
	go generate ./...

build: generate
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)/*
.PHONY: up logs down docker-build check-conflicts

docker-build:
	cd test && docker compose build

up:
	cd test && docker compose up

down:
	cd test && docker compose down --remove-orphans

check-conflicts:
	@echo "=== Checking for Network Address Conflicts ==="
	@echo "\n1. Current System Networks:"
	@ip route show | grep -E "172\.|192\.168\.|10\." || echo "No conflicting routes found"
	@echo "\n2. Current Docker Networks:"
	@docker network ls --format "table {{.Name}}\t{{.Driver}}\t{{.Scope}}"
	@echo "\n3. Docker Network Details:"
	@docker network inspect bridge --format '{{range .IPAM.Config}}Subnet: {{.Subnet}}{{end}}' 2>/dev/null || echo "No bridge network details"
	@echo "\n4. Planned Docker Compose Networks:"
	@echo "   dhcp_net: 192.168.100.0/24 (Gateway: 192.168.100.1)"
	@echo "   static_net: 192.168.101.0/24 (Gateway: 192.168.101.1)"
	@echo "\n5. Conflict Analysis:"
	@if ip route show | grep -q "192\.168\.100\.\|192\.168\.101\."; then \
		echo "⚠️  CONFLICT DETECTED: 192.168.100.x or 192.168.101.x networks already exist!"; \
		ip route show | grep "192\.168\.100\.\|192\.168\.101\."; \
	else \
		echo "✅ No conflicts detected - 192.168.100.x and 192.168.101.x networks are available"; \
	fi
	@echo "\n6. System Interface Check:"
	@ip addr show | grep -E "inet 192\.168\.(100|101)\." && echo "⚠️  Interface conflict detected!" || echo "✅ No interface conflicts"

logs:
	cd test && docker compose logs -f client

fmt:
	go fmt ./...

vet:
	go vet ./...

deps:
	go mod tidy

install: build
	go install ./...

help:
	@echo "Available targets:"
	@echo "  build           Build the binary"
	@echo "  run             Run the application"
	@echo "  test            Run tests"
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
	@echo "  check-conflicts Check for network address conflicts"
