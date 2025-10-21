# Quality Assurance Quickstart Guide

## Overview

This guide provides step-by-step instructions for setting up and using the comprehensive quality assurance system for the SmartTicket project.

## Prerequisites

### Required Tools

```bash
# Go 1.21+ (already installed)
go version

# Essential QA tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sonatypecommunity/nancy@latest
go install github.com/rakyll/hey@latest
go install github.com/tsenart/vegeta@latest
go install github.com/matm/gocov-html@latest
go install github.com/wadey/gocovmerge@latest
```

### Environment Setup

```bash
# Set up environment variables
export SMARTTICKET_ENV=test
export SMARTTICKET_LOG_LEVEL=debug
export SMARTTICKET_DB_PATH=./data/qa_test.db
export SMARTTICKET_JWT_SECRET=test-secret-key-for-qa-only
export SMARTTICKET_SERVER_PORT=6533
```

## Quickstart Steps

### 1. Initialize QA Environment

```bash
# Create QA directories
mkdir -p qa/{reports,profiles,coverage,security,benchmarks,load-tests}

# Initialize test database
make migrate QA_ENV=test

# Verify setup
make health
```

### 2. Run Comprehensive Test Suite

```bash
# Run all tests with coverage
make test-cover-100

# Run specific test categories
make test-unit
make test-integration
make test-e2e

# View coverage report
open qa/coverage/coverage.html
```

### 3. Execute Performance Testing

```bash
# Run benchmarks
make bench-all

# Run with profiling
make profile-server

# Execute load tests
make profile-load

# View performance reports
cat qa/benchmarks/results/latest.txt
```

### 4. Perform Security Assessment

```bash
# Run comprehensive security scan
make security-audit

# Run specific security tools
make security-gosec
make security-vulncheck
make security-nancy

# View security reports
cat qa/security/security-report-$(date +%Y%m%d).txt
```

### 5. Quality Gate Evaluation

```bash
# Evaluate all quality gates
make quality-gates

# Check specific gates
make quality-coverage
make quality-performance
make quality-security
make quality-lint

# View quality dashboard
make quality-dashboard
```

## Configuration

### QA Configuration File

Create `configs/config.qa.yaml`:

```yaml
server:
  host: "localhost"
  port: 6533
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 60

database:
  path: "./data/qa_test.db"
  max_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 300

logging:
  level: "debug"
  format: "json"
  output: "stdout"

security:
  jwt_secret: "test-secret-key-for-qa-only"
  jwt_access_duration: "24h"
  jwt_refresh_duration: "168h"

quality_assurance:
  coverage:
    threshold: 100.0
    exclude_patterns:
      - "*/mocks/*"
      - "*/testutils/*"

  performance:
    response_time_p95_ms: 200
    response_time_p99_ms: 500
    memory_usage_mb: 512
    throughput_rps: 1000

  security:
    scan_types:
      - "gosec"
      - "govulncheck"
      - "staticcheck"
      - "nancy"
    severity_threshold: "medium"

  testing:
    parallel_workers: 4
    timeout_per_test: "30s"
    retry_failed_tests: true
    retry_count: 3
```

### Makefile QA Targets

Add to your existing Makefile:

```makefile
# QA Directory Structure
QA_DIR = qa
REPORTS_DIR = $(QA_DIR)/reports
COVERAGE_DIR = $(QA_DIR)/coverage
PROFILES_DIR = $(QA_DIR)/profiles
SECURITY_DIR = $(QA_DIR)/security
BENCHMARKS_DIR = $(QA_DIR)/benchmarks
LOAD_TESTS_DIR = $(QA_DIR)/load-tests

# Initialize QA Environment
.PHONY: qa-init
qa-init: ## Initialize QA environment
	@echo "🔧 Initializing QA environment..."
	@mkdir -p $(REPORTS_DIR) $(COVERAGE_DIR) $(PROFILES_DIR) $(SECURITY_DIR) $(BENCHMARKS_DIR) $(LOAD_TESTS_DIR)
	@echo "✅ QA environment initialized"

# Comprehensive Testing
.PHONY: test-cover-100
test-cover-100: ## Run tests with 100% coverage requirement
	@echo "🧪 Running tests with 100% coverage requirement..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@COVERAGE=$$(go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 100" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% below 100% requirement"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets requirement"; \
	fi

# Performance Testing
.PHONY: bench-all
bench-all: ## Run all benchmarks
	@echo "🏎️ Running comprehensive benchmarks..."
	@mkdir -p $(BENCHMARKS_DIR)
	$(GOTEST) -bench=. -benchmem -run=^$$ -count=3 ./... | tee $(BENCHMARKS_DIR)/benchmark-$(shell date +%Y%m%d-%H%M%S).txt
	@echo "✅ Benchmarks completed"

.PHONY: profile-server
profile-server: ## Start server with profiling enabled
	@echo "🔍 Starting server with profiling..."
	@LOG_LEVEL=debug $(GORUN) cmd/server/main.go serve --config configs/config.qa.yaml --profile

.PHONY: profile-load
profile-load: ## Run load tests with profiling
	@echo "⚡ Running load tests..."
	@./scripts/load-test.sh --profile --duration=60s --concurrent=50 --rate=100

# Security Testing
.PHONY: security-audit
security-audit: ## Run comprehensive security audit
	@echo "🔒 Running comprehensive security audit..."
	@mkdir -p $(SECURITY_DIR)
	@echo "1. Running gosec..." && gosec -fmt json -out $(SECURITY_DIR)/gosec-$(shell date +%Y%m%d).json ./... || true
	@echo "2. Running vulnerability check..." && govulncheck -json ./... > $(SECURITY_DIR)/vuln-$(shell date +%Y%m%d).json || true
	@echo "3. Checking dependencies..." && cat go.sum | nancy sleuth --output-format json > $(SECURITY_DIR)/nancy-$(shell date +%Y%m%d).json || true
	@echo "4. Running staticcheck..." && staticcheck ./... > $(SECURITY_DIR)/staticcheck-$(shell date +%Y%m%d).txt || true
	@echo "✅ Security audit complete"

# Quality Gates
.PHONY: quality-gates
quality-gates: ## Evaluate all quality gates
	@echo "🚦 Evaluating quality gates..."
	@make quality-coverage
	@make quality-performance
	@make quality-security
	@make quality-lint
	@echo "✅ All quality gates evaluated"

.PHONY: quality-coverage
quality-coverage: ## Check coverage quality gate
	@echo "📊 Checking coverage gate..."
	@COVERAGE=$$(go tool cover -func=qa/coverage/coverage.out 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//' || echo "0"); \
	if [ $$(echo "$$COVERAGE >= 100" | bc -l) -eq 1 ]; then \
		echo "✅ Coverage gate passed: $$COVERAGE%"; \
	else \
		echo "❌ Coverage gate failed: $$COVERAGE% (required: 100%)"; \
		exit 1; \
	fi

.PHONY: quality-performance
quality-performance: ## Check performance quality gate
	@echo "⚡ Checking performance gate..."
	@if [ -f qa/benchmarks/results/latest.txt ]; then \
		echo "✅ Performance data available"; \
	else \
		echo "⚠️ No performance data found - running benchmarks..."; \
		make bench-all; \
	fi

.PHONY: quality-security
quality-security: ## Check security quality gate
	@echo "🔒 Checking security gate..."
	@if [ -f qa/security/gosec-$(shell date +%Y%m%d).json ]; then \
		CRITICAL=$$(jq -r '.Issues | map(select(.severity == "HIGH")) | length' qa/security/gosec-$(shell date +%Y%m%d).json); \
		if [ "$$CRITICAL" -eq 0 ]; then \
			echo "✅ Security gate passed: No critical vulnerabilities"; \
		else \
			echo "❌ Security gate failed: $$CRITICAL critical vulnerabilities"; \
			exit 1; \
		fi; \
	else \
		echo "⚠️ No security data found - running security audit..."; \
		make security-audit; \
	fi

.PHONY: quality-lint
quality-lint: ## Check linting quality gate
	@echo "🧹 Checking linting gate..."
	@if golangci-lint run; then \
		echo "✅ Linting gate passed"; \
	else \
		echo "❌ Linting gate failed"; \
		exit 1; \
	fi

# QA Dashboard
.PHONY: quality-dashboard
quality-dashboard: ## Generate quality dashboard
	@echo "📈 Generating quality dashboard..."
	@mkdir -p $(REPORTS_DIR)
	@echo "# Quality Dashboard - $(shell date)" > $(REPORTS_DIR)/dashboard.md
	@echo "" >> $(REPORTS_DIR)/dashboard.md
	@echo "## Coverage" >> $(REPORTS_DIR)/dashboard.md
	@if [ -f qa/coverage/coverage.out ]; then \
		COVERAGE=$$(go tool cover -func=qa/coverage/coverage.out | tail -1 | awk '{print $$3}'); \
		echo "- Current Coverage: $$COVERAGE" >> $(REPORTS_DIR)/dashboard.md; \
	fi
	@echo "" >> $(REPORTS_DIR)/dashboard.md
	@echo "## Security" >> $(REPORTS_DIR)/dashboard.md
	@if [ -f qa/security/gosec-$(shell date +%Y%m%d).json ]; then \
		ISSUES=$$(jq '.Issues | length' qa/security/gosec-$(shell date +%Y%m%d).json); \
		echo "- Security Issues: $$ISSUES" >> $(REPORTS_DIR)/dashboard.md; \
	fi
	@echo "✅ Dashboard generated: $(REPORTS_DIR)/dashboard.md"
	@open $(REPORTS_DIR)/dashboard.md

# Clean QA Artifacts
.PHONY: qa-clean
qa-clean: ## Clean QA artifacts
	@echo "🧹 Cleaning QA artifacts..."
	@rm -rf $(QA_DIR)
	@echo "✅ QA artifacts cleaned"
```

## Daily QA Workflow

### Development Workflow

```bash
# 1. Before starting development
make qa-init

# 2. During development - run tests frequently
make test-unit

# 3. Before committing - run quality gates
make quality-gates

# 4. After pushing - full QA pipeline
make test-cover-100 && make bench-all && make security-audit
```

### Pre-Release Checklist

```bash
# 1. Complete test coverage
make test-cover-100

# 2. Performance validation
make bench-all

# 3. Security assessment
make security-audit

# 4. Quality gate validation
make quality-gates

# 5. Generate quality report
make quality-dashboard

# 6. Review all reports
open qa/coverage/coverage.html
cat qa/benchmarks/results/latest.txt
cat qa/security/security-report-$(date +%Y%m%d).txt
```

## Continuous Integration

### GitHub Actions Integration

Create `.github/workflows/qa.yml`:

```yaml
name: Quality Assurance

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  quality-assurance:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

    - name: Install QA tools
      run: |
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        go install golang.org/x/vuln/cmd/govulncheck@latest
        go install github.com/sonatypecommunity/nancy@latest

    - name: Initialize QA environment
      run: make qa-init

    - name: Run tests with coverage
      run: make test-cover-100

    - name: Run benchmarks
      run: make bench-all

    - name: Run security audit
      run: make security-audit

    - name: Evaluate quality gates
      run: make quality-gates

    - name: Upload coverage reports
      uses: actions/upload-artifact@v3
      with:
        name: coverage-reports
        path: qa/coverage/

    - name: Upload security reports
      uses: actions/upload-artifact@v3
      with:
        name: security-reports
        path: qa/security/

    - name: Upload benchmark results
      uses: actions/upload-artifact@v3
      with:
        name: benchmark-results
        path: qa/benchmarks/
```

## Monitoring and Alerting

### Performance Monitoring

```bash
# Monitor real-time performance
curl http://localhost:6533/api/v1/internal/performance/metrics | jq

# Check SLA compliance
curl http://localhost:6533/api/v1/internal/quality/gates?gate_type=performance | jq

# View performance trends
curl "http://localhost:6533/api/v1/internal/performance/metrics?from_date=$(date -d '1 day ago' -Iseconds)&to_date=$(date -Iseconds)" | jq
```

### Security Monitoring

```bash
# Check security status
curl http://localhost:6533/api/v1/internal/security/scans?severity=high | jq

# Monitor vulnerability trends
curl http://localhost:6533/api/v1/internal/security/scans?status=open | jq

# Get security summary
curl http://localhost:6533/api/v1/internal/security/scans | jq '.data.summary'
```

### Coverage Monitoring

```bash
# Get current coverage
curl http://localhost:6533/api/v1/internal/test/coverage | jq '.data.overall_coverage'

# Check coverage trends
curl "http://localhost:6533/api/v1/internal/test/coverage?from_date=$(date -d '7 days ago' -Iseconds)" | jq

# Get module-wise coverage
curl http://localhost:6533/api/v1/internal/test/coverage | jq '.data.module_coverage'
```

## Troubleshooting

### Common Issues

#### Coverage Below 100%
```bash
# Identify uncovered lines
go tool cover -func=qa/coverage/coverage.out | grep -v "100.0%"

# Generate detailed coverage report
go tool cover -html=qa/coverage/coverage.out -o qa/coverage/detailed.html

# Run specific module tests
go test -coverprofile=qa/coverage/module.out ./internal/ticket/...
```

#### Performance Regressions
```bash
# Compare benchmark results
benchcmp qa/benchmarks/results/baseline.txt qa/benchmarks/results/latest.txt

# Profile specific functions
go test -cpuprofile=qa/profiles/cpu.prof -bench=BenchmarkFunction ./internal/...

# Analyze profiles
go tool pprof qa/profiles/cpu.prof
```

#### Security Vulnerabilities
```bash
# Run specific security scanner
gosec -severity high ./...

# Check for dependency vulnerabilities
govulncheck ./...

# Generate detailed security report
gosec -fmt json -out qa/security/detailed-report.json ./...
```

## Best Practices

### Test Coverage
- Write tests before writing code (TDD)
- Aim for 100% line coverage
- Test edge cases and error conditions
- Use table-driven tests for multiple scenarios

### Performance
- Establish performance baselines
- Monitor performance regressions
- Profile bottlenecks regularly
- Set realistic performance targets

### Security
- Scan code regularly for vulnerabilities
- Keep dependencies updated
- Follow secure coding practices
- Review security findings promptly

### Quality Gates
- Automate quality checks in CI/CD
- Set appropriate quality thresholds
- Review quality trends regularly
- Address quality issues promptly

## Support and Resources

### Documentation
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [gosec Documentation](https://github.com/securecodewarrior/gosec)
- [golangci-lint Configuration](https://golangci-lint.run/usage/configuration/)
- [Vegeta Load Testing](https://github.com/tsenart/vegeta)

### Tools and Utilities
- [go tool cover](https://golang.org/cmd/cover/) - Coverage analysis
- [go tool pprof](https://golang.org/cmd/pprof/) - Performance profiling
- [benchstat](https://golang.org/x/perf/cmd/benchstat/) - Benchmark comparison

### Scripts and Automation
- `scripts/load-test.sh` - Load testing automation
- `scripts/generate-qa-report.sh` - QA report generation
- `scripts/setup-qa-environment.sh` - Environment setup

This comprehensive QA system ensures the SmartTicket project maintains high quality standards throughout development and deployment.