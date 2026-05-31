# SmartTicket Simple Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean

# Binary info
BINARY_NAME=smartticket
BUILD_DIR=build
CONFIG_FILE=configs/config.dev.yaml

# Default target
.PHONY: help
help: ## Show help
	@echo "SmartTicket - Simple Commands:"
	@echo "  make dev       - Start development server (API only)"
	@echo "  make build     - Build API-only binary"
	@echo "  make build-all - Build single binary with embedded web console"
	@echo "  make test      - Run tests"
	@echo "  make run      - Run without building"
	@echo "  make clean    - Clean build files"
	@echo "  make deps     - Install dependencies"

# Development
.PHONY: dev
dev: ## Start development server
	$(GOCMD) run cmd/server/main.go serve --config $(CONFIG_FILE)

.PHONY: run
run: dev ## Alias for dev

# Building
.PHONY: build
build: ## Build API-only binary (no embedded web console)
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go

.PHONY: build-web
build-web: ## Build the web console (web/dist)
	cd web && pnpm install --frozen-lockfile && pnpm build

.PHONY: build-all
build-all: build-web ## Build single static binary with the web console embedded
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -tags embedui -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) — serves API + console on :6533"

# Testing
.PHONY: test
test: ## Run tests
	$(GOTEST) -v ./...

# Dependencies
.PHONY: deps
deps: ## Install dependencies
	$(GOCMD) mod download
	$(GOCMD) mod tidy

# Database
.PHONY: migrate
migrate: ## Run database migrations
	$(GOCMD) run cmd/server/main.go migrate --config $(CONFIG_FILE)

# Cleaning
.PHONY: clean
clean: ## Clean build files
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

# Code quality
.PHONY: fmt
fmt: ## Format code
	$(GOCMD) fmt ./...

.PHONY: lint
lint: ## Run linter (if available)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed - skipping lint"; \
	fi

# Quick setup
.PHONY: setup
setup: ## Quick setup for new developers
	deps
	@mkdir -p data
	migrate
	@echo "Setup complete! Run 'make dev' to start"

# Default
.DEFAULT_GOAL := help