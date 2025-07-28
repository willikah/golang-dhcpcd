# Test-specific Makefile for Docker integration testing
# This handles the test environment setup and Docker operations

.PHONY: docker-build up up-d down logs clean-docker check-conflicts

# Build Docker images for testing
docker-build:
	docker compose build

# Start the test environment
up:
	docker compose up

# Start the test environment in detached mode
up-d:
	docker compose up -d

# Stop the test environment
down:
	docker compose down --remove-orphans

# View logs from the client containers
logs:
	docker compose logs -f client-1 client-2

# View logs from a specific service
logs-client-1:
	docker compose logs -f client-1

logs-client-2:
	docker compose logs -f client-2

logs-dhcpd:
	docker compose logs -f dhcpd

# Clean up Docker resources
clean-docker:
	docker compose down --remove-orphans --volumes
	docker system prune -f

# Check for network conflicts
check-conflicts:
	@echo "Checking for network conflicts..."
	@docker network ls | grep -E "(dhcp_net|static_net)" || echo "No conflicting networks found"
	@docker ps -a | grep -E "(dhcpd|client)" || echo "No conflicting containers found"

# Restart the entire test environment
restart: down up-d

# Show status of all test containers
status:
	docker compose ps

# Execute shell in client-1 container (for debugging)
shell-client-1:
	docker compose exec client-1 /bin/sh

# Execute shell in client-2 container (for debugging)  
shell-client-2:
	docker compose exec client-2 /bin/sh

# Show help
help:
	@echo "Available test targets:"
	@echo "  docker-build     Build Docker images for testing"
	@echo "  up               Start test environment"
	@echo "  up-d             Start test environment in detached mode"
	@echo "  down             Stop test environment"
	@echo "  logs             View logs from client containers"
	@echo "  logs-client-1    View logs from client-1"
	@echo "  logs-client-2    View logs from client-2"
	@echo "  logs-dhcpd       View logs from DHCP server"
	@echo "  clean-docker     Clean up Docker resources"
	@echo "  check-conflicts  Check for network conflicts"
	@echo "  restart          Restart the entire test environment"
	@echo "  status           Show status of all test containers"
	@echo "  shell-client-1   Execute shell in client-1 container"
	@echo "  shell-client-2   Execute shell in client-2 container"
	@echo "  help             Show this help message"
