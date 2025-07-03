# Makefile for SoulHound Discord Music Bot

.PHONY: help build test deploy clean docker-build docker-test docker-deploy podman-build podman-test podman-deploy

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Go targets
build: ## Build the Go binary
	go build -o soulhound cmd/main.go

test: ## Run Go tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f soulhound-bin ./soulhound
	go clean

# Docker targets
docker-build: ## Build Docker image
	./scripts/build.sh --docker

docker-test: ## Test Docker image
	./scripts/test.sh --docker

docker-deploy: ## Deploy with Docker
	./scripts/deploy.sh --docker

docker-logs: ## Show Docker container logs
	./scripts/deploy.sh --docker --logs

docker-stop: ## Stop Docker container
	./scripts/deploy.sh --docker --stop

# Podman targets
podman-build: ## Build Podman image
	./scripts/build.sh --podman

podman-test: ## Test Podman image
	./scripts/test.sh --podman

podman-deploy: ## Deploy with Podman
	./scripts/deploy.sh --podman

podman-logs: ## Show Podman container logs
	./scripts/deploy.sh --podman --logs

podman-stop: ## Stop Podman container
	./scripts/deploy.sh --podman --stop

# Compose targets
compose-up: ## Start with docker-compose
	docker compose up -d

compose-down: ## Stop docker-compose
	docker compose down

compose-logs: ## Show docker-compose logs
	docker compose logs -f

# Development targets
dev-build: ## Build development image
	./scripts/build.sh -t soulhound:dev

dev-shell: ## Run development shell
	docker run -it --rm -v $(PWD):/app -w /app golang:1.21-alpine sh

# Setup targets
setup: ## Setup development environment
	cp .env.example .env
	@echo "Please edit .env file with your tokens"

# All-in-one targets
all-docker: docker-build docker-test docker-deploy ## Build, test, and deploy with Docker

all-podman: podman-build podman-test podman-deploy ## Build, test, and deploy with Podman