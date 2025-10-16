# SmartTicket Makefile
# A simple Makefile for common development tasks

.PHONY: help build clean test run-db docker-build docker-run lint format check docs

# Default target
help:
	@echo "SmartTicket - B2B Multi-tenant Ticketing Platform"
	@echo ""
	@echo "Available commands:"
	@echo "  build        - Build the entire workspace"
	@echo "  build-debug  - Build in debug mode (faster)"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run all tests"
	@echo "  test-unit    - Run unit tests only"
	@echo "  lint         - Run clippy lints"
	@echo "  format       - Format code with rustfmt"
	@echo "  check        - Run format check, clippy, and tests"
	@echo "  run-db       - Start PostgreSQL database with Docker"
	@echo "  stop-db      - Stop PostgreSQL database"
	@echo "  migrate      - Run database migrations"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  dev          - Start development environment"
	@echo "  docs         - Generate documentation"
	@echo ""

# Build targets
build:
	@echo "🔨 Building SmartTicket workspace..."
	cargo build --release

build-debug:
	@echo "🔨 Building SmartTicket in debug mode..."
	cargo build

clean:
	@echo "🧹 Cleaning build artifacts..."
	cargo clean
	rm -f Cargo.lock

# Testing targets
test:
	@echo "🧪 Running all tests..."
	cargo test --workspace --all-features

test-unit:
	@echo "🧪 Running unit tests..."
	cargo test --workspace --lib

test-integration:
	@echo "🧪 Running integration tests..."
	cargo test --workspace --test '*'

# Code quality targets
lint:
	@echo "🔍 Running clippy lints..."
	cargo clippy --workspace --all-features -- -D warnings

format:
	@echo "✨ Formatting code..."
	cargo fmt --all

format-check:
	@echo "✨ Checking code format..."
	cargo fmt --all -- --check

check: format-check lint test
	@echo "✅ All checks passed!"

# Database targets
run-db:
	@echo "🐘 Starting PostgreSQL database..."
	docker run -d \
		--name smartticket-postgres \
		-e POSTGRES_DB=smartticket \
		-e POSTGRES_USER=smartticket \
		-e POSTGRES_PASSWORD=smartticket123 \
		-p 5432:5432 \
		m.daocloud.io/docker.io/library/postgres:15-alpine

stop-db:
	@echo "🛑 Stopping PostgreSQL database..."
	docker stop smartticket-postgres 2>/dev/null || true
	docker rm smartticket-postgres 2>/dev/null || true

migrate:
	@echo "📋 Running database migrations..."
	cargo run --bin migrate

reset-db: stop-db run-db migrate
	@echo "🔄 Database reset complete!"

# Docker targets
docker-build:
	@echo "🐳 Building Docker images..."
	docker-compose build

docker-run:
	@echo "🚀 Starting services with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "🛑 Stopping Docker Compose services..."
	docker-compose down

# Development targets
dev: run-db
	@echo "🚀 Starting development environment..."
	@echo "Database is running on port 5432"
	@echo "Run 'make migrate' to setup database schema"
	@echo "Run 'cargo run' to start the application"

dev-full: docker-run
	@echo "🚀 Starting full development environment with Docker Compose..."

# Documentation targets
docs:
	@echo "📚 Generating documentation..."
	cargo doc --workspace --no-deps --document-private-items

open-docs: docs
	@echo "🌐 Opening documentation in browser..."
	cargo doc --workspace --no-deps --open

# Utility targets
status:
	@echo "📊 Project Status:"
	@echo "Git status:"
	@git status --porcelain || echo "Not a git repository"
	@echo ""
	@echo "Docker containers:"
	@docker ps --filter "name=smartticket" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No SmartTicket containers running"
	@echo ""
	@echo "Database connection test:"
	@docker exec smartticket-postgres pg_isready -U smartticket 2>/dev/null && echo "✅ Database ready" || echo "❌ Database not running"

logs-db:
	@echo "📋 PostgreSQL logs:"
	docker logs -f smartticket-postgres

install-tools:
	@echo "🛠️ Installing development tools..."
	cargo install cargo-watch cargo-audit cargo-outdated
	rustup component add clippy rustfmt

# Quick development loop
watch:
	@echo "👀 Watching for changes and auto-rebuilding..."
	cargo watch -x 'run'

# Security audit
audit:
	@echo "🔒 Running security audit..."
	cargo audit

# Dependencies
deps-update:
	@echo "📦 Updating dependencies..."
	cargo update

deps-tree:
	@echo "🌳 Showing dependency tree..."
	cargo tree

# Quick commands for common tasks
quick-test: build-debug test-unit
	@echo "⚡ Quick test complete!"

quick-build: format lint build-debug
	@echo "⚡ Quick build complete!"

# Version info
version:
	@echo "📋 SmartTicket Version Info:"
	@echo "Rust: $(shell rustc --version)"
	@echo "Cargo: $(shell cargo --version)"
	@echo "Docker: $(shell docker --version)"
	@echo "Docker Compose: $(shell docker-compose --version)"

# CI/CD helpers
ci: format-check lint test
	@echo "✅ CI checks passed!"

# Local development setup
setup: install-tools run-db
	@echo "🎉 SmartTicket development environment setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Run 'make migrate' to setup the database"
	@echo "2. Run 'make run' to start the application"
	@echo "3. Visit http://localhost:8080 to access the application"

# Application run targets
run:
	@echo "🚀 Starting SmartTicket application..."
	cargo run

run-gateway:
	@echo "🚀 Starting SmartTicket Gateway..."
	cargo run -p smartticket-gateway

run-core:
	@echo "🚀 Starting SmartTicket Core Service..."
	cargo run -p smartticket-core

# Database backup/restore
backup-db:
	@echo "💾 Creating database backup..."
	docker exec smartticket-postgres pg_dump -U smartticket smartticket > backup_$(shell date +%Y%m%d_%H%M%S).sql

restore-db:
	@echo "📥 Restoring database from backup..."
	@read -p "Enter backup file path: " backup_file; \
	docker exec -i smartticket-postgres psql -U smartticket smartticket < $$backup_file