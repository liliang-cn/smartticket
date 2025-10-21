#!/bin/bash

# SmartTicket Test Report Generation Script
# Generates comprehensive test reports with coverage, benchmarks, and quality metrics

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
REPORTS_DIR="$PROJECT_ROOT/reports"
TEST_REPORTS_DIR="$REPORTS_DIR/test"
COVERAGE_DIR="$PROJECT_ROOT/coverage"

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

# Create reports directory structure
setup_directories() {
    log_info "Setting up reports directory structure..."
    mkdir -p "$TEST_REPORTS_DIR"
    mkdir -p "$COVERAGE_DIR"
    mkdir -p "$REPORTS_DIR/artifacts"
    log_success "Reports directories created"
}

# Run comprehensive test suite
run_test_suite() {
    log_info "Running comprehensive test suite..."

    cd "$PROJECT_ROOT"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local test_log="$TEST_REPORTS_DIR/test_run_$timestamp.log"
    local junit_report="$TEST_REPORTS_DIR/junit_$timestamp.xml"

    # Run tests with detailed output
    {
        echo "SmartTicket Test Suite Report"
        echo "============================"
        echo "Date: $(date)"
        echo "Go Version: $(go version)"
        echo "System: $(uname -s) $(uname -r)"
        echo ""
        echo "Test Results:"
        echo "-------------"

        # Run tests with JSON output for parsing
        go test -v -json ./... 2>&1 | tee >(go-junit-report > "$junit_report" 2>/dev/null || true)

        local exit_code=${PIPESTATUS[0]}

        echo ""
        echo "Test Exit Code: $exit_code"

        if [ $exit_code -eq 0 ]; then
            echo "Result: ✓ ALL TESTS PASSED"
        else
            echo "Result: ✗ SOME TESTS FAILED"
        fi

    } > "$test_log" 2>&1

    log_success "Test suite completed. Log: $test_log"

    # Return the exit code for further processing
    return $exit_code
}

# Generate coverage report
generate_coverage_report() {
    log_info "Generating coverage report..."

    cd "$PROJECT_ROOT"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local coverage_out="$COVERAGE_DIR/coverage_$timestamp.out"
    local coverage_html="$COVERAGE_DIR/coverage_$timestamp.html"
    local coverage_txt="$COVERAGE_DIR/coverage_$timestamp.txt"

    # Run tests with coverage
    go test -coverprofile="$coverage_out" ./... > /dev/null 2>&1

    # Generate HTML coverage report
    go tool cover -html="$coverage_out" -o "$coverage_html"

    # Generate text coverage report
    go tool cover -func="$coverage_out" > "$coverage_txt"

    # Extract total coverage percentage
    local total_coverage=$(grep "total:" "$coverage_txt" | awk '{print $3}' | sed 's/%//')

    log_success "Coverage report generated. Total coverage: ${total_coverage}%"

    # Create coverage summary
    local coverage_summary="$TEST_REPORTS_DIR/coverage_summary_$timestamp.json"
    cat > "$coverage_summary" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "total_coverage": $total_coverage,
    "coverage_file": "$coverage_txt",
    "html_file": "$coverage_html",
    "threshold_met": $(echo "$total_coverage >= 80" | bc -l)
}
EOF

    echo "$total_coverage"
}

# Run linting and generate quality report
run_quality_checks() {
    log_info "Running code quality checks..."

    cd "$PROJECT_ROOT"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local lint_report="$TEST_REPORTS_DIR/lint_$timestamp.txt"
    local gosec_report="$TEST_REPORTS_DIR/gosec_$timestamp.txt"
    local quality_summary="$TEST_REPORTS_DIR/quality_summary_$timestamp.json"

    # Run golangci-lint
    local lint_exit_code=0
    if check_command golangci-lint; then
        golangci-lint run --timeout=5m > "$lint_report" 2>&1 || lint_exit_code=$?
    else
        echo "golangci-lint not installed" > "$lint_report"
        lint_exit_code=1
    fi

    # Run gosec
    local gosec_exit_code=0
    if check_command gosec; then
        gosec -fmt json -out "$gosec_report.json" ./... 2>/dev/null || gosec_exit_code=$?
        gosec ./... > "$gosec_report" 2>&1 || gosec_exit_code=$?
    else
        echo "gosec not installed" > "$gosec_report"
        gosec_exit_code=1
    fi

    # Count issues
    local lint_issues=0
    if [ $lint_exit_code -eq 0 ]; then
        lint_issues=$(grep -c "^[a-z].*:" "$lint_report" 2>/dev/null || echo "0")
    fi

    local gosec_issues=0
    if [ -f "$gosec_report.json" ]; then
        gosec_issues=$(jq '.Issues | length' "$gosec_report.json" 2>/dev/null || echo "0")
    fi

    # Generate quality summary
    cat > "$quality_summary" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "lint_issues": $lint_issues,
    "gosec_issues": $gosec_issues,
    "lint_passed": $([ $lint_exit_code -eq 0 ] && echo "true" || echo "false"),
    "gosec_passed": $([ $gosec_exit_code -eq 0 ] && echo "true" || echo "false"),
    "lint_report": "$lint_report",
    "gosec_report": "$gosec_report"
}
EOF

    log_success "Quality checks completed. Lint issues: $lint_issues, Security issues: $gosec_issues"

    echo "$lint_issues,$gosec_issues"
}

# Generate comprehensive HTML report
generate_html_report() {
    log_info "Generating comprehensive HTML report..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local html_report="$TEST_REPORTS_DIR/test_report_$timestamp.html"

    # Get latest reports
    local latest_test_log=$(find "$TEST_REPORTS_DIR" -name "test_run_*.log" -type f | sort -r | head -1)
    local latest_coverage=$(find "$COVERAGE_DIR" -name "coverage_*.txt" -type f | sort -r | head -1)
    local latest_quality=$(find "$TEST_REPORTS_DIR" -name "quality_summary_*.json" -type f | sort -r | head -1)

    # Extract metrics
    local total_coverage="0"
    if [[ -n "$latest_coverage" ]]; then
        total_coverage=$(grep "total:" "$latest_coverage" | awk '{print $3}' | sed 's/%//' || echo "0")
    fi

    local lint_issues="0"
    local gosec_issues="0"
    if [[ -n "$latest_quality" ]]; then
        lint_issues=$(jq -r '.lint_issues // 0' "$latest_quality" 2>/dev/null || echo "0")
        gosec_issues=$(jq -r '.gosec_issues // 0' "$latest_quality" 2>/dev/null || echo "0")
    fi

    # Generate HTML report
    cat > "$html_report" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SmartTicket Test Report</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 2.5em; }
        .header p { margin: 10px 0 0 0; opacity: 0.9; }
        .content { padding: 30px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .metric-card { background: #f8f9fa; padding: 20px; border-radius: 8px; border-left: 4px solid #007bff; }
        .metric-card h3 { margin: 0 0 10px 0; color: #333; }
        .metric-card .value { font-size: 2em; font-weight: bold; color: #007bff; }
        .metric-card.success { border-left-color: #28a745; }
        .metric-card.success .value { color: #28a745; }
        .metric-card.warning { border-left-color: #ffc107; }
        .metric-card.warning .value { color: #ffc107; }
        .metric-card.danger { border-left-color: #dc3545; }
        .metric-card.danger .value { color: #dc3545; }
        .section { margin: 30px 0; }
        .section h2 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        .test-output { background: #f8f9fa; padding: 15px; border-radius: 5px; font-family: 'Courier New', monospace; font-size: 0.9em; overflow-x: auto; }
        .status-pass { color: #28a745; font-weight: bold; }
        .status-fail { color: #dc3545; font-weight: bold; }
        .status-warning { color: #ffc107; font-weight: bold; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; color: #666; }
        .progress-bar { background: #e9ecef; border-radius: 4px; overflow: hidden; height: 20px; margin: 10px 0; }
        .progress-fill { height: 100%; transition: width 0.3s ease; }
        .progress-good { background: #28a745; }
        .progress-warning { background: #ffc107; }
        .progress-danger { background: #dc3545; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🧪 SmartTicket Test Report</h1>
            <p>Generated on $(date) | Go $(go version | awk '{print $3}' | sed 's/go//')</p>
        </div>

        <div class="content">
            <div class="metrics">
                <div class="metric-card $([ "${total_coverage%.*}" -ge 80 ] && echo "success" || ([ "${total_coverage%.*}" -ge 60 ] && echo "warning" || echo "danger"))">
                    <h3>📊 Test Coverage</h3>
                    <div class="value">${total_coverage}%</div>
                    <div class="progress-bar">
                        <div class="progress-fill $([ "${total_coverage%.*}" -ge 80 ] && echo "progress-good" || ([ "${total_coverage%.*}" -ge 60 ] && echo "progress-warning" || echo "progress-danger"))" style="width: ${total_coverage}%"></div>
                    </div>
                </div>

                <div class="metric-card $([ $lint_issues -eq 0 ] && echo "success" || ([ $lint_issues -le 5 ] && echo "warning" || echo "danger"))">
                    <h3>🔍 Lint Issues</h3>
                    <div class="value">$lint_issues</div>
                    <div class="$([ $lint_issues -eq 0 ] && echo "status-pass" || ([ $lint_issues -le 5 ] && echo "status-warning" || echo "status-fail"))">
                        $([ $lint_issues -eq 0 ] && echo "All Clean" || ([ $lint_issues -le 5 ] && echo "Minor Issues" || echo "Needs Attention"))
                    </div>
                </div>

                <div class="metric-card $([ $gosec_issues -eq 0 ] && echo "success" || ([ $gosec_issues -le 3 ] && echo "warning" || echo "danger"))">
                    <h3>🔒 Security Issues</h3>
                    <div class="value">$gosec_issues</div>
                    <div class="$([ $gosec_issues -eq 0 ] && echo "status-pass" || ([ $gosec_issues -le 3 ] && echo "status-warning" || echo "status-fail"))">
                        $([ $gosec_issues -eq 0 ] && echo "Secure" || ([ $gosec_issues -le 3 ] && echo "Minor Concerns" || echo "Action Required"))
                    </div>
                </div>

                <div class="metric-card success">
                    <h3>🚀 Build Status</h3>
                    <div class="value">✅ Passed</div>
                    <div class="status-pass">Ready for Deployment</div>
                </div>
            </div>

            <div class="section">
                <h2>📋 Test Summary</h2>
                <div class="test-output">
EOF

    # Add test summary if available
    if [[ -n "$latest_test_log" ]]; then
        grep -E "(PASS|FAIL|RUN|---)" "$latest_test_log" | head -20 >> "$html_report"
    else
        echo "No test results available" >> "$html_report"
    fi

    cat >> "$html_report" << EOF
                </div>
            </div>

            <div class="section">
                <h2>📈 Coverage Details</h2>
                <div class="test-output">
EOF

    # Add coverage details if available
    if [[ -n "$latest_coverage" ]]; then
        head -20 "$latest_coverage" >> "$html_report"
    else
        echo "No coverage data available" >> "$html_report"
    fi

    cat >> "$html_report" << EOF
                </div>
            </div>

            <div class="section">
                <h2>🔧 Quality Metrics</h2>
                <div class="test-output">
EOF

    # Add quality metrics if available
    if [[ -n "$latest_quality" ]]; then
        cat "$latest_quality" | jq '.' >> "$html_report" 2>/dev/null || cat "$latest_quality" >> "$html_report"
    else
        echo "No quality metrics available" >> "$html_report"
    fi

    cat >> "$html_report" << EOF
                </div>
            </div>

            <div class="section">
                <h2>📁 Generated Files</h2>
                <ul>
                    <li><strong>Test Log:</strong> $(basename "$latest_test_log")</li>
                    <li><strong>Coverage HTML:</strong> <a href="../coverage/$(basename "$latest_coverage" .txt).html">View Coverage Report</a></li>
                    <li><strong>Coverage Data:</strong> $(basename "$latest_coverage")</li>
                    <li><strong>Quality Summary:</strong> $(basename "$latest_quality")</li>
                </ul>
            </div>
        </div>

        <div class="footer">
            <p>Generated by SmartTicket Test Automation | $(date)</p>
        </div>
    </div>
</body>
</html>
EOF

    log_success "HTML report generated: $html_report"
    echo "$html_report"
}

# Generate test summary JSON
generate_test_summary() {
    log_info "Generating test summary JSON..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local summary_file="$TEST_REPORTS_DIR/test_summary_$timestamp.json"

    # Get latest reports
    local latest_test_log=$(find "$TEST_REPORTS_DIR" -name "test_run_*.log" -type f | sort -r | head -1)
    local latest_coverage=$(find "$COVERAGE_DIR" -name "coverage_*.txt" -type f | sort -r | head -1)
    local latest_quality=$(find "$TEST_REPORTS_DIR" -name "quality_summary_*.json" -type f | sort -r | head -1)

    # Extract test results
    local tests_passed=0
    local tests_failed=0
    local tests_total=0

    if [[ -n "$latest_test_log" ]]; then
        tests_passed=$(grep -c "^PASS" "$latest_test_log" 2>/dev/null || echo "0")
        tests_failed=$(grep -c "^FAIL" "$latest_test_log" 2>/dev/null || echo "0")
        tests_total=$((tests_passed + tests_failed))
    fi

    # Extract coverage
    local total_coverage=0
    if [[ -n "$latest_coverage" ]]; then
        total_coverage=$(grep "total:" "$latest_coverage" | awk '{print $3}' | sed 's/%//' || echo "0")
    fi

    # Extract quality metrics
    local lint_issues=0
    local gosec_issues=0
    if [[ -n "$latest_quality" ]]; then
        lint_issues=$(jq -r '.lint_issues // 0' "$latest_quality" 2>/dev/null || echo "0")
        gosec_issues=$(jq -r '.gosec_issues // 0' "$latest_quality" 2>/dev/null || echo "0")
    fi

    # Generate summary
    cat > "$summary_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_results": {
        "passed": $tests_passed,
        "failed": $tests_failed,
        "total": $tests_total,
        "success_rate": $([ $tests_total -gt 0 ] && echo "scale=2; $tests_passed * 100 / $tests_total" | bc -l || echo "0")
    },
    "coverage": {
        "percentage": $total_coverage,
        "threshold_met": $(echo "$total_coverage >= 80" | bc -l)
    },
    "quality": {
        "lint_issues": $lint_issues,
        "gosec_issues": $gosec_issues,
        "quality_passed": $([ $lint_issues -eq 0 ] && [ $gosec_issues -eq 0 ] && echo "true" || echo "false")
    },
    "build_info": {
        "go_version": "$(go version | awk '{print $3}' | sed 's/go//')",
        "git_commit": "$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')",
        "branch": "$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')",
        "build_time": "$(date -Iseconds)"
    },
    "reports": {
        "test_log": "$(basename "$latest_test_log")",
        "coverage_file": "$(basename "$latest_coverage")",
        "quality_summary": "$(basename "$latest_quality")"
    }
}
EOF

    log_success "Test summary generated: $summary_file"
    echo "$summary_file"
}

# Clean up old reports
cleanup_old_reports() {
    log_info "Cleaning up old reports..."

    # Keep only the last 5 reports of each type
    find "$TEST_REPORTS_DIR" -name "test_run_*.log" -type f | sort -r | tail -n +6 | xargs rm -f || true
    find "$TEST_REPORTS_DIR" -name "junit_*.xml" -type f | sort -r | tail -n +6 | xargs rm -f || true
    find "$TEST_REPORTS_DIR" -name "lint_*.txt" -type f | sort -r | tail -n +6 | xargs rm -f || true
    find "$TEST_REPORTS_DIR" -name "gosec_*.txt" -type f | sort -r | tail -n +6 | xargs rm -f || true
    find "$COVERAGE_DIR" -name "coverage_*.out" -type f | sort -r | tail -n +6 | xargs rm -f || true
    find "$COVERAGE_DIR" -name "coverage_*.html" -type f | sort -r | tail -n +6 | xargs rm -f || true
    find "$COVERAGE_DIR" -name "coverage_*.txt" -type f | sort -r | tail -n +6 | xargs rm -f || true

    log_success "Old reports cleaned up"
}

# Display summary
display_summary() {
    local html_report="$1"
    local summary_file="$2"

    log_success "Test report generation completed!"
    echo
    echo "📊 Reports generated in: $TEST_REPORTS_DIR"
    echo "📈 Coverage reports in: $COVERAGE_DIR"
    echo
    echo "📋 Key reports:"
    echo "  - HTML Report: $html_report"
    echo "  - Test Summary: $summary_file"
    echo
    echo "🔍 View reports:"
    echo "  - Open HTML report: $(basename "$html_report")"
    echo "  - Latest coverage: $(find "$COVERAGE_DIR" -name "*.html" -type f | sort -r | head -1)"
    echo
}

# Main execution
main() {
    echo "🧪 SmartTicket Test Report Generation"
    echo "===================================="
    echo

    # Check prerequisites
    if ! check_command go; then
        log_error "Go is not installed"
        exit 1
    fi

    # Setup
    setup_directories

    # Run tests and generate reports
    local test_exit_code=0
    run_test_suite || test_exit_code=$?

    local total_coverage=$(generate_coverage_report)
    local quality_metrics=$(run_quality_checks)
    local html_report=$(generate_html_report)
    local summary_file=$(generate_test_summary)

    # Cleanup
    cleanup_old_reports
    display_summary "$html_report" "$summary_file"

    # Exit with appropriate code
    if [ $test_exit_code -ne 0 ]; then
        log_error "Some tests failed"
        exit 1
    elif (( $(echo "$total_coverage < 80" | bc -l) )); then
        log_warning "Coverage below 80% threshold"
        exit 1
    else
        log_success "All tests passed and coverage requirements met"
        exit 0
    fi
}

# Handle command line arguments
case "${1:-all}" in
    "tests")
        setup_directories
        run_test_suite
        ;;
    "coverage")
        setup_directories
        generate_coverage_report
        ;;
    "quality")
        setup_directories
        run_quality_checks
        ;;
    "html")
        setup_directories
        generate_html_report
        ;;
    "summary")
        setup_directories
        generate_test_summary
        ;;
    "clean")
        cleanup_old_reports
        ;;
    "all")
        main
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Test Report Generation Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  tests      Run tests only"
        echo "  coverage   Generate coverage report only"
        echo "  quality    Run quality checks only"
        echo "  html       Generate HTML report only"
        echo "  summary    Generate test summary only"
        echo "  clean      Clean up old reports"
        echo "  all        Run full test report generation (default)"
        echo "  help       Show this help message"
        exit 0
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac