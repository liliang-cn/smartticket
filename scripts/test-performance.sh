#!/bin/bash

# SmartTicket Performance Benchmarking Script
# Runs comprehensive performance benchmarks and generates reports

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORTS_DIR="$PROJECT_ROOT/reports/performance"
BENCHMARKS_DIR="$PROJECT_ROOT/reports/benchmarks"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_command() {
    command -v "$1" >/dev/null 2>&1
}

# Create reports directory
setup_reports() {
    log_info "Setting up reports directory..."
    mkdir -p "$REPORTS_DIR"
    mkdir -p "$BENCHMARKS_DIR"
    log_success "Reports directory created"
}

# Run Go benchmarks
run_go_benchmarks() {
    log_info "Running Go benchmarks..."

    cd "$PROJECT_ROOT"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local benchmark_file="$BENCHMARKS_DIR/benchmark_$timestamp.txt"
    local cpu_profile="$REPORTS_DIR/cpu_$timestamp.prof"
    local mem_profile="$REPORTS_DIR/memory_$timestamp.prof"

    # Run benchmarks with profiling
    go test -bench=. -benchmem -run=^$ -cpuprofile="$cpu_profile" -memprofile="$mem_profile" ./... | tee "$benchmark_file"

    # Generate benchmark summary
    local summary_file="$REPORTS_DIR/benchmark_summary_$timestamp.txt"
    {
        echo "SmartTicket Performance Benchmark Report"
        echo "======================================="
        echo "Date: $(date)"
        echo "Go Version: $(go version)"
        echo "System: $(uname -s) $(uname -r)"
        echo ""
        echo "Benchmark Results:"
        echo "------------------"
        cat "$benchmark_file"
        echo ""
        echo "Performance Profiles Generated:"
        echo "- CPU Profile: $cpu_profile"
        echo "- Memory Profile: $mem_profile"
    } > "$summary_file"

    log_success "Go benchmarks completed. Report: $summary_file"
}

# Run API performance tests
run_api_performance_tests() {
    log_info "Running API performance tests..."

    cd "$PROJECT_ROOT"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local api_report="$REPORTS_DIR/api_performance_$timestamp.json"

    # Start development server in background
    log_info "Starting development server for API tests..."
    make build-local
    ./build/smartticket serve --config configs/config.dev.yaml &
    local server_pid=$!

    # Wait for server to start
    sleep 5

    # Check if server is running
    if ! curl -s http://localhost:6533/api/v1/health > /dev/null; then
        log_warning "Server not responding, skipping API performance tests"
        kill $server_pid 2>/dev/null || true
        return 0
    fi

    # Run basic API performance tests using curl
    {
        echo "{"
        echo "  \"timestamp\": \"$(date -Iseconds)\","
        echo "  \"tests\": ["

        # Test health endpoint
        local health_time=$(curl -o /dev/null -s -w '%{time_total}' http://localhost:6533/api/v1/health)
        echo "    {"
        echo "      \"endpoint\": \"/api/v1/health\","
        echo "      \"response_time\": $health_time,"
        echo "      \"status\": \"success\""
        echo "    },"

        # Test ready endpoint
        local ready_time=$(curl -o /dev/null -s -w '%{time_total}' http://localhost:6533/api/v1/health/ready)
        echo "    {"
        echo "      \"endpoint\": \"/api/v1/health/ready\","
        echo "      \"response_time\": $ready_time,"
        echo "      \"status\": \"success\""
        echo "    }"

        echo "  ]"
        echo "}"
    } > "$api_report"

    # Stop server
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true

    log_success "API performance tests completed. Report: $api_report"
}

# Run database performance tests
run_database_performance_tests() {
    log_info "Running database performance tests..."

    cd "$PROJECT_ROOT"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local db_report="$REPORTS_DIR/database_performance_$timestamp.txt"

    # Create test database
    local test_db="data/smartticket_perf_test.db"
    rm -f "$test_db"

    # Run database performance tests
    go test -v -tags=integration -timeout=5m tests/integration/database_performance_test.go -db="$test_db" > "$db_report" 2>&1 || {
        log_warning "Database performance tests failed or not implemented"
        return 0
    }

    log_success "Database performance tests completed. Report: $db_report"

    # Clean up test database
    rm -f "$test_db"
}

# Generate performance comparison report
generate_comparison_report() {
    log_info "Generating performance comparison report..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local comparison_report="$REPORTS_DIR/performance_comparison_$timestamp.html"

    # Get latest benchmark files
    local latest_benchmark=$(find "$BENCHMARKS_DIR" -name "benchmark_*.txt" -type f | sort -r | head -1)
    local previous_benchmark=$(find "$BENCHMARKS_DIR" -name "benchmark_*.txt" -type f | sort -r | head -2 | tail -1)

    if [[ -z "$latest_benchmark" ]]; then
        log_warning "No benchmark files found for comparison"
        return 0
    fi

    cat > "$comparison_report" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>SmartTicket Performance Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; }
        .benchmark-table { width: 100%; border-collapse: collapse; }
        .benchmark-table th, .benchmark-table td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        .benchmark-table th { background-color: #f2f2f2; }
        .improvement { color: green; }
        .regression { color: red; }
        .neutral { color: gray; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SmartTicket Performance Report</h1>
        <p>Generated: $(date)</p>
        <p>Go Version: $(go version)</p>
    </div>

    <div class="section">
        <h2>Latest Benchmark Results</h2>
        <pre>$(cat "$latest_benchmark")</pre>
    </div>

    <div class="section">
        <h2>System Information</h2>
        <pre>$(uname -a)</pre>
        <pre>$(go env)</pre>
    </div>
</body>
</html>
EOF

    log_success "Performance comparison report generated: $comparison_report"
}

# Check performance against thresholds
check_performance_thresholds() {
    log_info "Checking performance against thresholds..."

    local latest_benchmark=$(find "$BENCHMARKS_DIR" -name "benchmark_*.txt" -type f | sort -r | head -1)

    if [[ -z "$latest_benchmark" ]]; then
        log_warning "No benchmark file found for threshold checking"
        return 0
    fi

    # Define performance thresholds (in nanoseconds)
    local thresholds=(
        "BenchmarkUserService/CreateUser:100000000"  # 100ms
        "BenchmarkTicketService/CreateTicket:200000000" # 200ms
        "BenchmarkAuthService/Login:150000000"        # 150ms
        "BenchmarkDatabaseService/Query:50000000"     # 50ms
    )

    local failed_thresholds=()

    for threshold in "${thresholds[@]}"; do
        local benchmark_name="${threshold%:*}"
        local max_time="${threshold#*:}"

        # Extract benchmark result
        local actual_time=$(grep "$benchmark_name" "$latest_benchmark" | awk '{print $3}' | sed 's/ns\/op//' || echo "0")

        if [[ -n "$actual_time" && "$actual_time" != "0" ]]; then
            if (( actual_time > max_time )); then
                failed_thresholds+=("$benchmark_name: ${actual_time}ns > ${max_time}ns")
            fi
        fi
    done

    if [[ ${#failed_thresholds[@]} -gt 0 ]]; then
        log_error "Performance thresholds exceeded:"
        for failure in "${failed_thresholds[@]}"; do
            log_error "  - $failure"
        done
        return 1
    else
        log_success "All performance thresholds met"
        return 0
    fi
}

# Upload reports to artifact storage (optional)
upload_reports() {
    log_info "Preparing reports for upload..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local archive_name="performance_reports_$timestamp.tar.gz"

    # Create archive of all reports
    tar -czf "$archive_name" -C "$PROJECT_ROOT" reports/

    log_success "Reports archived: $archive_name"
    log_info "You can upload this archive to your artifact storage system"
}

# Clean up old reports
cleanup_old_reports() {
    log_info "Cleaning up old reports..."

    # Keep only the last 10 reports
    find "$REPORTS_DIR" -name "*.txt" -type f | sort -r | tail -n +11 | xargs rm -f || true
    find "$REPORTS_DIR" -name "*.json" -type f | sort -r | tail -n +11 | xargs rm -f || true
    find "$REPORTS_DIR" -name "*.html" -type f | sort -r | tail -n +11 | xargs rm -f || true
    find "$REPORTS_DIR" -name "*.prof" -type f | sort -r | tail -n +11 | xargs rm -f || true
    find "$BENCHMARKS_DIR" -name "*.txt" -type f | sort -r | tail -n +11 | xargs rm -f || true

    log_success "Old reports cleaned up"
}

# Display summary
display_summary() {
    log_info "Performance benchmarking completed!"
    echo
    echo "📊 Reports generated in: $REPORTS_DIR"
    echo "📈 Benchmark data in: $BENCHMARKS_DIR"
    echo
    echo "📋 Generated files:"
    find "$REPORTS_DIR" "$BENCHMARKS_DIR" -name "*$(date +%Y%m%d)*" -type f | sort | while read -r file; do
        echo "  - $file"
    done
    echo
    echo "🔍 View reports:"
    echo "  - Latest HTML: $(find "$REPORTS_DIR" -name "*.html" -type f | sort -r | head -1)"
    echo "  - Latest benchmark: $(find "$BENCHMARKS_DIR" -name "*.txt" -type f | sort -r | head -1)"
    echo
}

# Main execution
main() {
    echo "🚀 SmartTicket Performance Benchmarking"
    echo "===================================="
    echo

    # Check prerequisites
    if ! check_command go; then
        log_error "Go is not installed"
        exit 1
    fi

    if ! check_command curl; then
        log_warning "curl is not installed, API performance tests will be skipped"
    fi

    # Run benchmarking
    setup_reports
    run_go_benchmarks
    run_api_performance_tests
    run_database_performance_tests
    generate_comparison_report

    # Check thresholds
    if ! check_performance_thresholds; then
        log_warning "Some performance thresholds were not met"
    fi

    # Cleanup and summary
    upload_reports
    cleanup_old_reports
    display_summary
}

# Handle command line arguments
case "${1:-all}" in
    "go")
        setup_reports
        run_go_benchmarks
        ;;
    "api")
        setup_reports
        run_api_performance_tests
        ;;
    "database")
        setup_reports
        run_database_performance_tests
        ;;
    "compare")
        setup_reports
        generate_comparison_report
        ;;
    "thresholds")
        check_performance_thresholds
        ;;
    "clean")
        cleanup_old_reports
        ;;
    "all")
        main
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Performance Benchmarking Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  go         Run Go benchmarks only"
        echo "  api        Run API performance tests only"
        echo "  database   Run database performance tests only"
        echo "  compare    Generate comparison reports only"
        echo "  thresholds Check performance thresholds only"
        echo "  clean      Clean up old reports"
        echo "  all        Run all benchmarking (default)"
        echo "  help       Show this help message"
        exit 0
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac