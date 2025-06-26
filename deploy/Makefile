.PHONY: help build run dev stop clean certs logs test ghcr

# Default target
help: ## Show this help message
	@echo "GrpcHub Deployment Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Production deployment
build: ## Build the Docker image
	docker-compose build

run: certs ## Run the production server
	docker-compose up -d

stop: ## Stop all services
	docker-compose down

restart: ## Restart services
	docker-compose restart

# GHCR deployment
ghcr: certs ## Run using GitHub Container Registry image
	docker-compose -f docker-compose.ghcr.yaml up -d

ghcr-stop: ## Stop GHCR deployment
	docker-compose -f docker-compose.ghcr.yaml down

ghcr-logs: ## Show GHCR deployment logs
	docker-compose -f docker-compose.ghcr.yaml logs -f

ghcr-restart: ## Restart GHCR deployment
	docker-compose -f docker-compose.ghcr.yaml restart

ghcr-pull: ## Pull latest image from GHCR
	docker pull ghcr.io/lisoboss/grpchub:latest

# Development
dev: certs ## Run development environment
	docker-compose -f docker-compose.dev.yaml up -d

dev-stop: ## Stop development environment
	docker-compose -f docker-compose.dev.yaml down

dev-logs: ## Show development logs
	docker-compose -f docker-compose.dev.yaml logs -f

dev-shell: ## Open shell in development container
	docker-compose -f docker-compose.dev.yaml exec dev-tools bash

# Certificate management
certs: ## Generate TLS certificates using Docker
	@if [ ! -f certs/server.pem ]; then \
		echo "🔐 Generating TLS certificates..."; \
		docker-compose --profile tools run --rm cert-generator; \
	else \
		echo "✅ Certificates already exist"; \
	fi

certs-standalone: ## Generate TLS certificates using standalone script
	@if [ ! -f certs/server.pem ]; then \
		echo "🔐 Downloading and running certificate generator..."; \
		curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash; \
	else \
		echo "✅ Certificates already exist"; \
	fi

certs-clean: ## Remove all certificates
	rm -rf certs/*.pem certs/*.crt certs/*.key
	@echo "🧹 Certificates cleaned"

certs-refresh: certs-clean certs ## Regenerate certificates

# Logs and monitoring
logs: ## Show production logs
	docker-compose logs -f

logs-server: ## Show server logs only
	docker-compose logs -f grpchub-server

# Testing
test: ## Run tests
	cd .. && cargo test

test-integration: ## Run integration tests
	cd .. && cargo test --test integration

# Cleanup
clean: ## Clean up containers and volumes
	docker-compose down -v
	docker-compose -f docker-compose.dev.yaml down -v
	docker-compose -f docker-compose.ghcr.yaml down -v
	docker system prune -f

clean-all: clean certs-clean ## Clean everything including certificates

# Health check
health: ## Check service health
	@echo "🏥 Checking GrpcHub server health..."
	@if docker-compose ps grpchub-server | grep -q "Up"; then \
		echo "✅ Server is running"; \
	else \
		echo "❌ Server is not running"; \
		exit 1; \
	fi

health-ghcr: ## Check GHCR deployment health
	@echo "🏥 Checking GrpcHub GHCR server health..."
	@if docker-compose -f docker-compose.ghcr.yaml ps grpchub-server | grep -q "Up"; then \
		echo "✅ GHCR Server is running"; \
	else \
		echo "❌ GHCR Server is not running"; \
		exit 1; \
	fi

# Quick start
quick-start: build run ## Quick start (build and run - requires existing certs)
	@if [ ! -f certs/server.pem ]; then \
		echo "❌ Certificates not found! Generate them first with: make certs"; \
		exit 1; \
	fi
	@echo "🚀 GrpcHub is starting..."
	@echo "📡 Server will be available at: localhost:50055"
	@echo "📋 Check logs with: make logs"

quick-start-ghcr: ghcr-pull ghcr ## Quick start with GHCR image (requires existing certs)
	@if [ ! -f certs/server.pem ]; then \
		echo "❌ Certificates not found! Generate them first with: make certs-standalone"; \
		exit 1; \
	fi
	@echo "🚀 GrpcHub (GHCR) is starting..."
	@echo "📡 Server will be available at: localhost:50055"
	@echo "📋 Check logs with: make ghcr-logs"

# Complete quick start with certificate generation
quick-start-full: certs build run ## Complete quick start (generate certs, build, and run)
	@echo "🚀 GrpcHub is starting..."
	@echo "📡 Server will be available at: localhost:50055"
	@echo "📋 Check logs with: make logs"

quick-start-ghcr-full: certs-standalone ghcr-pull ghcr ## Complete GHCR quick start
	@echo "🚀 GrpcHub (GHCR) is starting..."
	@echo "📡 Server will be available at: localhost:50055"
	@echo "📋 Check logs with: make ghcr-logs"

# Development quick start
dev-start: dev ## Quick development start (requires existing certs)
	@if [ ! -f certs/server.pem ]; then \
		echo "❌ Certificates not found! Generate them first with: make certs"; \
		exit 1; \
	fi
	@echo "🛠️  GrpcHub development environment starting..."
	@echo "📡 Server will be available at: localhost:50055"
	@echo "📋 Check logs with: make dev-logs"

dev-start-full: certs dev ## Complete development start with certificate generation
	@echo "🛠️  GrpcHub development environment starting..."
	@echo "📡 Server will be available at: localhost:50055"
	@echo "📋 Check logs with: make dev-logs"