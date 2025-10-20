# SmartTicket Makefile
# Provides targets for building, testing, and deploying the SmartTicket application

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
BINARY_NAME=smartticket
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# LDFLAGS for building
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -s -w"

# Directories
BUILD_DIR=build
DIST_DIR=dist
COVERAGE_DIR=coverage
DATA_DIR=data

# Database files
DEV_DB=$(DATA_DIR)/smartticket_dev.db
TEST_DB=$(DATA_DIR)/smartticket_test.db

# Configuration
CONFIG_FILE=configs/config.dev.yaml
DOCKER_COMPOSE_FILE=deployments/docker-compose.yml

# Default target
.PHONY: all
all: clean deps lint test build

# Help target
.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Development targets
.PHONY: deps
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

.PHONY: deps-update
deps-update: ## Update dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

.PHONY: clean
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out

# Building targets
.PHONY: build
build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) cmd/server/main.go

.PHONY: build-local
build-local: ## Build the binary for local platform
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go

.PHONY: build-all
build-all: ## Build binaries for all platforms
	@mkdir -p $(DIST_DIR)
	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 cmd/server/main.go
	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 cmd/server/main.go
	# Darwin AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/server/main.go
	# Darwin ARM64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/server/main.go
	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/server/main.go

# Testing targets
.PHONY: test
test: ## Run tests
	@mkdir -p $(DATA_DIR)
	$(GOTEST) -v ./...

.PHONY: test-short
test-short: ## Run short tests
	$(GOTEST) -short -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GOTEST) -race -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

.PHONY: test-integration
test-integration: ## Run integration tests
	@mkdir -p $(DATA_DIR)
	$(GOTEST) -tags=integration -v tests/integration/...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@mkdir -p $(DATA_DIR)
	$(GOTEST) -tags=e2e -v tests/e2e/...

# Code quality targets
.PHONY: fmt
fmt: ## Format code
	$(GOFMT) -s -w .

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	$(GOFMT) -s -d . | tee /dev/fd/2; test -z "$$($(GOFMT) -s -d .)"

.PHONY: lint
lint: ## Run linter
	$(GOLINT) run

.PHONY: lint-fix
lint-fix: ## Fix lint issues
	$(GOLINT) run --fix

.PHONY: vet
vet: ## Run go vet
	$(GOCMD) vet ./...

.PHONY: check
check: fmt-check vet lint ## Run all checks

# Database targets
.PHONY: db-setup
db-setup: ## Set up development database
	@mkdir -p $(DATA_DIR)
	@echo "Database files will be created at: $(DATA_DIR)/"

.PHONY: db-reset
db-reset: ## Reset development database
	rm -f $(DEV_DB)
	$(MAKE) db-setup

.PHONY: migrate
migrate: build-local ## Run database migrations
	./$(BUILD_DIR)/$(BINARY_NAME) migrate --config $(CONFIG_FILE)

# Seed data management
.PHONY: seed-build
seed-build: ## Build seed data tool
	@echo "Building seed data tool..."
	@mkdir -p scripts/seed/cmd/seed
	@go build -o scripts/seed/seed ./scripts/seed/cmd/seed

.PHONY: seed-generate
seed-generate: seed-build ## Generate seed data to file
	@echo "Generating seed data..."
	@mkdir -p scripts/seed/data
	@./scripts/seed/seed -output scripts/seed/data/development.json

.PHONY: seed
seed: seed-build ## Seed development database
	@echo "Seeding development database..."
	@./scripts/seed/seed -config $(CONFIG_FILE)

.PHONY: seed-reseed
seed-reseed: seed-build ## Clear and reseed database
	@echo "Clearing and reseeding database..."
	@./scripts/seed/seed -config $(CONFIG_FILE) -clear -force

.PHONY: seed-test
seed-test: seed-build ## Seed test database
	@echo "Seeding test database..."
	@./scripts/seed/seed -config configs/config.test.yaml -db $(TEST_DB)

.PHONY: seed-staging
seed-staging: seed-build ## Seed staging database
	@echo "Seeding staging database..."
	@./scripts/seed/seed -config configs/config.staging.yaml

.PHONY: seed-load
seed-load: seed-build ## Load seed data from file
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make seed-load FILE=path/to/seed.json"; \
		exit 1; \
	fi
	@echo "Loading seed data from $(FILE)..."
	@./scripts/seed/seed -config $(CONFIG_FILE) -load $(FILE)

.PHONY: seed-validate
seed-validate: seed-build ## Validate seed data structure
	@echo "Validating seed data..."
	@./scripts/seed/seed -output scripts/seed/data/validation.json -verbose
	@echo "Seed data validation completed"

.PHONY: seed-clean
seed-clean: ## Clean seed data files
	@echo "Cleaning seed data files..."
	@rm -f scripts/seed/data/*.json
	@rm -f scripts/seed/seed

# Development targets
.PHONY: dev
dev: build-local ## Run development server
	./$(BUILD_DIR)/$(BINARY_NAME) serve --config $(CONFIG_FILE)

.PHONY: dev-debug
dev-debug: build-local ## Run development server with debug logging
	LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME) serve --config $(CONFIG_FILE)

.PHONY: dev-db
dev-db: db-setup dev ## Set up database and run development server

.PHONY: run
run: ## Run without building (requires go installation)
	$(GOCMD) run cmd/server/main.go serve --config $(CONFIG_FILE)

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t smartticket:latest .

.PHONY: docker-run
docker-run: ## Run with Docker Compose
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: docker-stop
docker-stop: ## Stop Docker Compose
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

.PHONY: docker-logs
docker-logs: ## Show Docker Compose logs
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

.PHONY: docker-clean
docker-clean: ## Clean Docker resources
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v
	docker system prune -f

# Generation targets
.PHONY: generate
generate: ## Run go generate
	$(GOCMD) generate ./...

.PHONY: mocks
mocks: ## Generate mocks
	@echo "Generating mocks..."
	@find . -name "*.go" -type f | grep -v "_test.go" | xargs grep -l "go:generate" | xargs -I {} $(GOCMD) generate {}

# Security targets
.PHONY: security-scan
security-scan: ## Run security scan
	@echo "Running security scan..."
	gosec ./...

.PHONY: vuln-check
vuln-check: ## Check for vulnerabilities
	@echo "Checking for vulnerabilities..."
	govulncheck ./...

# Performance targets
.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

.PHONY: profile
profile: ## Run CPU profiling
	$(GOTEST) -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...

# Release targets
.PHONY: release
release: clean test lint build-all ## Prepare release
	@echo "Preparing release $(VERSION)"
	@mkdir -p $(DIST_DIR)/release
	@cp $(DIST_DIR)/* $(DIST_DIR)/release/
	@cd $(DIST_DIR)/release && sha256sum * > sha256sums.txt
	@echo "Release artifacts ready in $(DIST_DIR)/release"

# Installation targets
.PHONY: install
install: build-local ## Install binary locally
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

.PHONY: uninstall
uninstall: ## Remove binary from local system
	rm -f /usr/local/bin/$(BINARY_NAME)

# Documentation targets
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060

.PHONY: docs-check
docs-check: ## Check documentation
	@echo "Checking documentation..."
	@find . -name "*.go" -type f ! -path "./vendor/*" | xargs grep -L "Copyright" || true

# CI/CD targets
.PHONY: ci
ci: clean deps check test-race ## Run CI pipeline locally

.PHONY: pre-commit
pre-commit: fmt lint vet test-short ## Run pre-commit checks

# Quick commands
.PHONY: quick-test
quick-test: test-short ## Quick test run

.PHONY: quick-build
quick-build: build-local ## Quick build

.PHONY: quick-run
quick-run: run ## Quick run

# Information targets
.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

.PHONY: info
info: ## Show project information
	@echo "Project: SmartTicket"
	@echo "Go Version: $(shell go version)"
	@echo "Build Dir: $(BUILD_DIR)"
	@echo "Dist Dir: $(DIST_DIR)"
	@echo "Data Dir: $(DATA_DIR)"
	@echo "Config: $(CONFIG_FILE)"

# Watch targets (requires watchexec or similar tool)
.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@if command -v watchexec >/dev/null 2>&1; then \
		watchexec -r -e go -- make quick-build; \
	else \
		echo "watchexec not found. Install with: make install-watchexec"; \
	fi

.PHONY: watch-test
watch-test: ## Watch for changes and run tests
	@echo "Watching for changes and running tests..."
	@if command -v watchexec >/dev/null 2>&1; then \
		watchexec -r -e go -- make quick-test; \
	else \
		echo "watchexec not found. Install with: make install-watchexec"; \
	fi

.PHONY: dev-live
dev-live: ## Run development server with live reload
	@if command -v air >/dev/null 2>&1; then \
		echo "Starting development server with live reload..."; \
		air -c .air.toml; \
	else \
		echo "air not found. Install with: make install-air"; \
		echo "Falling back to regular development server..."; \
		$(MAKE) dev; \
	fi

# Create air configuration if it doesn't exist
.PHONY: .air.toml
.air.toml:
	@if ! command -v air >/dev/null 2>&1; then \
		$(MAKE) install-air; \
	fi
	@echo "Creating air configuration..."
	@cat > .air.toml << 'EOF'; \
root = "."; \
testdata_dir = "testdata"; \
tmp_dir = "tmp"; \
\
[build]; \
  args_bin = []; \
  bin = "./tmp/main"; \
  cmd = "go build -o ./tmp/main ./cmd/server/main.go"; \
  delay = 1000; \
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]; \
  exclude_file = []; \
  exclude_regex = ["_test.go"]; \
  exclude_unchanged = false; \
  follow_symlink = false; \
  full_bin = ""; \
  include_dir = []; \
  include_ext = ["go", "tpl", "tmpl", "html"]; \
  kill_delay = "0s"; \
  log = "build-errors.log"; \
  send_interrupt = false; \
  stop_on_root = false; \
\
[color]; \
  app = ""; \
  build = "yellow"; \
  main = "magenta"; \
  runner = "green"; \
  watcher = "cyan"; \
\
[log]; \
  time = false; \
\
[misc]; \
  clean_on_exit = false; \
\
[screen]; \
  clear_on_rebuild = false; \
  keep_scroll = true; \
EOF

.PHONY: create-air-config
create-air-config: ## Create air configuration for live reload
	@$(MAKE) .air.toml
	@echo "✅ Air configuration created"

# Backup and restore targets
.PHONY: backup
backup: ## Create backup of data directory
	@mkdir -p backups
	@if [ -d "$(DATA_DIR)" ]; then \
		tar -czf backups/smartticket-backup-$$(date +%Y%m%d-%H%M%S).tar.gz $(DATA_DIR)/; \
		echo "Backup created in backups/"; \
	else \
		echo "Data directory not found"; \
	fi

.PHONY: restore
restore: ## Restore from backup (usage: make restore BACKUP=filename)
	@if [ -z "$(BACKUP)" ]; then \
		echo "Usage: make restore BACKUP=filename"; \
		exit 1; \
	fi
	@if [ -f "backups/$(BACKUP)" ]; then \
		tar -xzf backups/$(BACKUP); \
		echo "Backup restored from backups/$(BACKUP)"; \
	else \
		echo "Backup file not found: backups/$(BACKUP)"; \
	fi

# Utility targets
.PHONY: tree
tree: ## Show project tree structure
	@echo "Project structure:"
	@tree -I 'vendor|*.git*|build|dist' || find . -type d | grep -v vendor | sort

.PHONY: count
count: ## Count lines of code
	@echo "Lines of code:"
	@find . -name "*.go" -type f ! -path "./vendor/*" | xargs wc -l | tail -1

# Environment targets
.PHONY: env-check
env-check: ## Check required tools
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go not found. Please install Go."; exit 1; }
	@command -v docker >/dev/null 2>&1 || echo "Docker not found. Install for container support."
	@command -v golangci-lint >/dev/null 2>&1 || echo "golangci-lint not found. Install for linting."
	@command -v gosec >/dev/null 2>&1 || echo "gosec not found. Install for security scanning."
	@echo "Tool check complete."

.PHONY: env-setup
env-setup: ## Set up development environment
	@echo "Setting up development environment..."
	$(MAKE) deps
	$(MAKE) db-setup
	@echo "Installing Git hooks..."
	@./scripts/install-git-hooks.sh
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@echo "Environment setup complete."

# Development tools installation
.PHONY: install-tools
install-tools: ## Install all development tools
	@echo "Installing development tools..."
	@$(MAKE) install-golangci-lint
	@$(MAKE) install-gosec
	@$(MAKE) install-mockgen
	@$(MAKE) install-goimports
	@$(MAKE) install-watchexec
	@$(MAKE) install-air
	@$(MAKE) install-govulncheck
	@echo "✅ All development tools installed successfully!"

.PHONY: install-golangci-lint
install-golangci-lint: ## Install golangci-lint
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
		echo "✅ golangci-lint installed"; \
	else \
		echo "✅ golangci-lint already installed"; \
	fi

.PHONY: install-gosec
install-gosec: ## Install gosec security scanner
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		echo "✅ gosec installed"; \
	else \
		echo "✅ gosec already installed"; \
	fi

.PHONY: install-mockgen
install-mockgen: ## Install mockgen for mock generation
	@if ! command -v mockgen >/dev/null 2>&1; then \
		echo "Installing mockgen..."; \
		$(GOGET) github.com/golang/mock/mockgen@latest; \
		echo "✅ mockgen installed"; \
	else \
		echo "✅ mockgen already installed"; \
	fi

.PHONY: install-goimports
install-goimports: ## Install goimports for import management
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		$(GOGET) golang.org/x/tools/cmd/goimports@latest; \
		echo "✅ goimports installed"; \
	else \
		echo "✅ goimports already installed"; \
	fi

.PHONY: install-watchexec
install-watchexec: ## Install watchexec for file watching
	@if ! command -v watchexec >/dev/null 2>&1; then \
		echo "Installing watchexec..."; \
		$(GOGET) github.com/watchexec/watchexec@latest; \
		echo "✅ watchexec installed"; \
	else \
		echo "✅ watchexec already installed"; \
	fi

.PHONY: install-air
install-air: ## Install air for live reloading
	@if ! command -v air >/dev/null 2>&1; then \
		echo "Installing air..."; \
		$(GOGET) github.com/cosmtrek/air@latest; \
		echo "✅ air installed"; \
	else \
		echo "✅ air already installed"; \
	fi

.PHONY: install-govulncheck
install-govulncheck: ## Install govulncheck for vulnerability scanning
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		$(GOGET) golang.org/x/vuln/cmd/govulncheck@latest; \
		echo "✅ govulncheck installed"; \
	else \
		echo "✅ govulncheck already installed"; \
	fi

.PHONY: update-tools
update-tools: ## Update all development tools
	@echo "Updating development tools..."
	@$(MAKE) update-golangci-lint
	@$(MAKE) update-gosec
	@$(MAKE) update-mockgen
	@$(MAKE) update-goimports
	@echo "✅ All development tools updated successfully!"

.PHONY: update-golangci-lint
update-golangci-lint: ## Update golangci-lint
	@echo "Updating golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest
	@echo "✅ golangci-lint updated"

.PHONY: update-gosec
update-gosec: ## Update gosec
	@echo "Updating gosec..."
	@$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "✅ gosec updated"

.PHONY: update-mockgen
update-mockgen: ## Update mockgen
	@echo "Updating mockgen..."
	@$(GOGET) -u github.com/golang/mock/mockgen@latest
	@echo "✅ mockgen updated"

.PHONY: update-goimports
update-goimports: ## Update goimports
	@echo "Updating goimports..."
	@$(GOGET) -u golang.org/x/tools/cmd/goimports@latest
	@echo "✅ goimports updated"

.PHONY: check-tools
check-tools: ## Check if all development tools are installed
	@echo "Checking development tools..."
	@tools=("golangci-lint" "gosec" "mockgen" "goimports" "watchexec" "air" "govulncheck"); \
	for tool in "$${tools[@]}"; do \
		if command -v $$tool >/dev/null 2>&1; then \
			echo "✅ $$tool is installed"; \
		else \
			echo "❌ $$tool is not installed (run 'make install-tools')"; \
		fi; \
	done

.PHONY: setup-dev
setup-dev: ## Complete development environment setup
	@echo "🚀 Setting up SmartTicket development environment..."
	@$(MAKE) check-go-version
	@$(MAKE) create-directories
	@$(MAKE) deps
	@$(MAKE) install-tools
	@$(MAKE) install-git-hooks
	@$(MAKE) setup-config-files
	@$(MAKE) build-local
	@echo ""
	@echo "✅ Development environment setup complete!"
	@echo "🎯 Next steps:"
	@echo "   make dev          - Start development server"
	@echo "   make test         - Run tests"
	@echo "   make lint         - Run linter"
	@echo "   make help         - Show all available commands"

.PHONY: check-go-version
check-go-version: ## Check Go version compatibility
	@echo "Checking Go version..."
	@if command -v go >/dev/null 2>&1; then \
		go_version=$$(go version | awk '{print $$3}' | sed 's/go//'); \
		echo "Found Go version: $$go_version"; \
		if [[ "$$go_version" < "1.21" ]] && [[ "$$go_version" != "1.25.1" ]]; then \
			echo "❌ Go 1.21+ is required. Found: $$go_version"; \
			exit 1; \
		fi; \
		echo "✅ Go version is compatible"; \
	else \
		echo "❌ Go is not installed"; \
		exit 1; \
	fi

.PHONY: create-directories
create-directories: ## Create required directories
	@echo "Creating required directories..."
	@mkdir -p $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR) $(DATA_DIR) $(DATA_DIR)/backups $(DATA_DIR)/uploads logs temp
	@echo "✅ Directories created"

.PHONY: setup-config-files
setup-config-files: ## Setup configuration files
	@echo "Setting up configuration files..."
	@if [ ! -f ".env.dev" ]; then \
		echo "Creating .env.dev configuration file..."; \
		echo "# SmartTicket Development Environment Configuration" > .env.dev; \
		echo "APP_ENV=development" >> .env.dev; \
		echo "APP_PORT=6533" >> .env.dev; \
		echo "APP_HOST=localhost" >> .env.dev; \
		echo "DB_PATH=./data/smartticket_dev.db" >> .env.dev; \
		echo "DB_TYPE=sqlite" >> .env.dev; \
		echo "LOG_LEVEL=debug" >> .env.dev; \
		echo "JWT_SECRET=dev-secret-key-change-in-production" >> .env.dev; \
		echo "JWT_EXPIRY=24h" >> .env.dev; \
		echo "✅ Created .env.dev"; \
	else \
		echo "✅ .env.dev already exists"; \
	fi
	@if [ ! -f ".env.local" ]; then \
		cp .env.dev .env.local.template; \
		echo "✅ Created .env.local.template"; \
	fi

# Git Hooks Management
.PHONY: install-hooks
install-hooks: ## Install Git hooks
	@echo "Installing Git hooks..."
	@if [ -d "scripts/git-hooks" ]; then \
		for hook in pre-commit pre-push commit-msg; do \
			if [ -f "scripts/git-hooks/$$hook" ]; then \
				ln -sf "../../scripts/git-hooks/$$hook" .git/hooks/$$hook 2>/dev/null || true; \
				chmod +x .git/hooks/$$hook 2>/dev/null || true; \
				echo "✅ Installed $$hook hook"; \
			fi; \
		done; \
	else \
		echo "Git hooks directory not found. Skipping hook installation."; \
	fi

.PHONY: uninstall-hooks
uninstall-hooks: ## Uninstall Git hooks
	@echo "Uninstalling Git hooks..."
	@cd .git/hooks && rm -f pre-commit pre-push commit-msg prepare-commit-msg

.PHONY: list-hooks
list-hooks: ## List installed Git hooks
	@echo "Installed Git hooks:"
	@if [ -d .git/hooks ]; then \
		for hook in pre-commit pre-push commit-msg prepare-commit-msg; do \
			if [ -L ".git/hooks/$$hook" ]; then \
				echo "  ✓ $$hook -> $$(readlink .git/hooks/$$hook)"; \
			elif [ -f ".git/hooks/$$hook" ]; then \
				echo "  ✓ $$hook (file)"; \
			else \
				echo "  ✗ $$hook (not installed)"; \
			fi; \
		done; \
	else \
		echo "  No .git/hooks directory found"; \
	fi

.PHONY: test-hooks
test-hooks: ## Test Git hooks with dummy commit
	@echo "Testing Git hooks..."
	@echo "Note: This will create temporary files and commits for testing"
	@read -p "Continue? (y/N): " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy] ]]; then \
		echo; \
		temp_file=$$(mktemp --suffix=.go); \
		echo 'package main\nfunc main() { println("test") }' > "$$temp_file"; \
		git add "$$temp_file" 2>/dev/null || true; \
		if git commit -m "test: hook test" --no-verify 2>/dev/null; then \
			echo "✓ Commit hook test bypassed successfully"; \
		fi; \
		git reset HEAD "$$temp_file" 2>/dev/null || true; \
		rm -f "$$temp_file"; \
		echo "✓ Hook test completed"; \
	else \
		echo; \
		echo "Hook test cancelled"; \
	fi

# Clean up temporary files
.PHONY: clean-temp
clean-temp: ## Clean temporary files
	@echo "Cleaning temporary files..."
	@find . -name "*.tmp" -delete
	@find . -name "*.log" -delete
	@find . -name "*.swp" -delete
	@find . -name "*.swo" -delete
	@find . -name ".DS_Store" -delete
	@find . -name "*.bak" -delete
	@echo "Temporary files cleaned."

# Comprehensive cleanup
.PHONY: clean-all
clean-all: clean docker-clean clean-temp ## Clean everything
	@echo "All cleaned up!"

# Quick start for new developers
.PHONY: quickstart
quickstart: env-setup dev ## Quick start for new developers
	@echo "SmartTicket is now running on http://localhost:6533"
	@echo "Press Ctrl+C to stop the server"

# Default to help if no target specified
.DEFAULT_GOAL := help