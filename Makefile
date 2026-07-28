.PHONY: dev dev-infra dev-backend dev-frontend migrate seed test lint build clean help

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development
dev: dev-infra ## Start full development stack
	@echo "Starting backend, worker, and frontend..."
	@make -j3 dev-backend dev-worker dev-frontend

dev-infra: ## Start infrastructure (Postgres, Redis, MinIO)
	docker compose up -d postgres redis minio

dev-backend: ## Start Go backend with hot reload
	cd backend && go run ./cmd/api

dev-frontend: ## Start Next.js frontend
	cd frontend && npm run dev

dev-worker: ## Start background worker
	cd backend && go run ./cmd/worker

# Database
migrate: ## Run database migrations
	cd backend && go run ./cmd/migrate

migrate-create: ## Create a new migration (usage: make migrate-create NAME=add_feature)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	touch backend/internal/database/migrations/$$(date +%Y%m%d%H%M%S)_$(NAME).up.sql
	touch backend/internal/database/migrations/$$(date +%Y%m%d%H%M%S)_$(NAME).down.sql

seed: ## Seed the database with sample data
	cd backend && go run ./cmd/migrate --seed

resetdev: ## Delete all users and onboarding state from the DEV database (refuses to run outside development)
	cd backend && go run ./cmd/resetdev

# Testing
test: ## Run all tests
	cd backend && go test ./...
	cd frontend && npm test

test-backend: ## Run backend tests
	cd backend && go test ./... -v

test-frontend: ## Run frontend tests
	cd frontend && npm test

# Code quality
lint: ## Run linters
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint

fmt: ## Format code
	cd backend && gofmt -w .
	cd frontend && npx prettier --write .

# Build
build: ## Build production images
	docker compose -f docker-compose.yml build

build-backend: ## Build backend binary
	cd backend && go build -o ../bin/api ./cmd/api
	cd backend && go build -o ../bin/worker ./cmd/worker

build-frontend: ## Build frontend
	cd frontend && npm run build

# Cleanup
clean: ## Remove build artifacts and volumes
	rm -rf bin/ frontend/.next frontend/node_modules
	docker compose down -v

# Docker
up: ## Start all services via Docker Compose
	docker compose up -d

down: ## Stop all services
	docker compose down

logs: ## View service logs
	docker compose logs -f

# Utilities
generate-openapi: ## Generate OpenAPI documentation
	cd backend && go run ./cmd/api --generate-openapi > docs/openapi.yaml
