#!/bin/bash

# SmartTicket Security Testing Script
# Comprehensive security testing including vulnerability scanning, dependency checks, and API security testing

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
SECURITY_REPORTS_DIR="$PROJECT_ROOT/reports/security"
TEMP_DIR="$PROJECT_ROOT/temp/security-test"

# Security thresholds
MAX_HIGH_VULNERABILITIES=0
MAX_MEDIUM_VULNERABILITIES=5
MAX_LOW_VULNERABILITIES=20

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

# Setup security test environment
setup_security_environment() {
    log_info "Setting up security test environment..."

    mkdir -p "$SECURITY_REPORTS_DIR"
    mkdir -p "$TEMP_DIR"

    cd "$PROJECT_ROOT"

    # Build application for testing
    if ! make build-local > /dev/null 2>&1; then
        log_error "Failed to build application for security testing"
        exit 1
    fi

    log_success "Security test environment setup completed"
}

# Run static application security testing (SAST)
run_sast_scanning() {
    log_info "Running Static Application Security Testing (SAST)..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local sast_report="$SECURITY_REPORTS_DIR/sast_$timestamp.txt"
    local gosec_report="$SECURITY_REPORTS_DIR/gosec_$timestamp.txt"
    local gosec_json="$SECURITY_REPORTS_DIR/gosec_$timestamp.json"

    {
        echo "Static Application Security Testing (SAST) Report"
        echo "=================================================="
        echo "Date: $(date)"
        echo "Go Version: $(go version)"
        echo "Project: $PROJECT_ROOT"
        echo ""

        # Run gosec security scanner
        echo "Running gosec security scanner..."
        local gosec_exit_code=0

        if check_command gosec; then
            log_info "Running gosec with JSON output..."
            if gosec -fmt json -out "$gosec_json" ./... 2>/dev/null; then
                echo "✓ gosec JSON report generated"

                # Convert to readable format
                if gosec -fmt text ./... > "$gosec_report" 2>&1; then
                    echo "✓ gosec text report generated"
                else
                    echo "⚠ gosec text report generation had issues"
                fi

                # Analyze results
                if [[ -f "$gosec_json" ]]; then
                    local high_issues=$(jq '[.Issues[] | select(.severity == "HIGH")] | length' "$gosec_json" 2>/dev/null || echo "0")
                    local medium_issues=$(jq '[.Issues[] | select(.severity == "MEDIUM")] | length' "$gosec_json" 2>/dev/null || echo "0")
                    local low_issues=$(jq '[.Issues[] | select(.severity == "LOW")] | length' "$gosec_json" 2>/dev/null || echo "0")
                    local total_issues=$(jq '.Issues | length' "$gosec_json" 2>/dev/null || echo "0")

                    echo ""
                    echo "gosec Security Analysis Results:"
                    echo "  - High Severity Issues: $high_issues"
                    echo "  - Medium Severity Issues: $medium_issues"
                    echo "  - Low Severity Issues: $low_issues"
                    echo "  - Total Issues: $total_issues"

                    # Check against thresholds
                    local threshold_passed=true
                    if [[ $high_issues -gt $MAX_HIGH_VULNERABILITIES ]]; then
                        echo "  ❌ HIGH severity issues exceed threshold ($MAX_HIGH_VULNERABILITIES)"
                        threshold_passed=false
                    fi

                    if [[ $medium_issues -gt $MAX_MEDIUM_VULNERABILITIES ]]; then
                        echo "  ⚠️  MEDIUM severity issues exceed threshold ($MAX_MEDIUM_VULNERABILITIES)"
                    fi

                    if [[ $low_issues -gt $MAX_LOW_VULNERABILITIES ]]; then
                        echo "  ⚠️  LOW severity issues exceed threshold ($MAX_LOW_VULNERABILITIES)"
                    fi

                    if [[ "$threshold_passed" == "true" ]]; then
                        echo "  ✅ Security thresholds met"
                    else
                        echo "  ❌ Security thresholds NOT met"
                        gosec_exit_code=1
                    fi
                fi
            else
                echo "❌ gosec execution failed"
                gosec_exit_code=1
            fi
        else
            echo "⚠️ gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
            gosec_exit_code=1
        fi

        # Run additional SAST checks
        echo ""
        echo "Running additional SAST checks..."

        # Check for hardcoded secrets
        echo "Checking for hardcoded secrets..."
        local secret_patterns=(
            "password[[:space:]]*=[[:space:]]*\"[^\"]+\""
            "secret[[:space:]]*=[[:space:]]*\"[^\"]+\""
            "key[[:space:]]*=[[:space:]]*\"[^\"]+\""
            "token[[:space:]]*=[[:space:]]*\"[^\"]+\""
            "api_key[[:space:]]*=[[:space:]]*\"[^\"]+\""
        )

        local secrets_found=false
        for pattern in "${secret_patterns[@]}"; do
            local matches=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.*" -exec grep -l -i "$pattern" {} \; 2>/dev/null || true)
            if [[ -n "$matches" ]]; then
                echo "  ⚠️  Potential hardcoded secrets found with pattern: $pattern"
                echo "$matches" | head -5
                secrets_found=true
            fi
        done

        if [[ "$secrets_found" == "false" ]]; then
            echo "  ✅ No obvious hardcoded secrets detected"
        fi

        # Check for unsafe functions
        echo ""
        echo "Checking for unsafe functions..."
        local unsafe_functions=("exec.Command" "os/exec" "eval" "unsafe")
        local unsafe_found=false

        for func in "${unsafe_functions[@]}"; do
            local matches=$(find . -name "*.go" -not -path "./vendor/*" -exec grep -l "$func" {} \; 2>/dev/null || true)
            if [[ -n "$matches" ]]; then
                echo "  ⚠️  Potentially unsafe function usage found: $func"
                echo "$matches" | head -3
                unsafe_found=true
            fi
        done

        if [[ "$unsafe_found" == "false" ]]; then
            echo "  ✅ No obvious unsafe function usage detected"
        fi

    } > "$sast_report" 2>&1

    log_success "SAST scanning completed. Report: $sast_report"
    return $gosec_exit_code
}

# Run dependency vulnerability scanning
run_dependency_scanning() {
    log_info "Running dependency vulnerability scanning..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local dep_report="$SECURITY_REPORTS_DIR/dependencies_$timestamp.txt"
    local vuln_report="$SECURITY_REPORTS_DIR/vulnerabilities_$timestamp.txt"

    {
        echo "Dependency Vulnerability Scanning Report"
        echo "======================================"
        echo "Date: $(date)"
        echo ""

        # Run govulncheck
        echo "Running govulncheck..."
        local vuln_exit_code=0

        if check_command govulncheck; then
            if govulncheck ./... > "$vuln_report" 2>&1; then
                echo "✓ govulncheck completed successfully"

                # Analyze vulnerability results
                local vuln_count=$(grep -c "CVE-" "$vuln_report" 2>/dev/null || echo "0")
                if [[ $vuln_count -gt 0 ]]; then
                    echo "  ⚠️  Found $vuln_count vulnerabilities in dependencies"
                    echo ""
                    echo "Vulnerability Summary:"
                    grep "CVE-" "$vuln_report" | head -10
                else
                    echo "  ✅ No vulnerabilities found in dependencies"
                fi
            else
                echo "❌ govulncheck failed"
                vuln_exit_code=1
            fi
        else
            echo "⚠️ govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
            echo "Attempting to use alternative method..."

            # Alternative: check go.mod for known vulnerable packages
            if [[ -f "go.mod" ]]; then
                echo "Checking go.mod for known vulnerable packages..."
                local vulnerable_packages=("github.com/golang/protobuf" "gopkg.in/yaml.v2")

                for pkg in "${vulnerable_packages[@]}"; do
                    if grep -q "$pkg" go.mod; then
                        echo "  ⚠️  Potentially vulnerable package found: $pkg"
                    fi
                done
            fi
        fi

        # Check for outdated dependencies
        echo ""
        echo "Checking for outdated dependencies..."
        if check_command go; then
            echo "Running 'go list -m -u all' to check for updates..."
            go list -m -u all 2>/dev/null | grep -E "\[.*\]" | head -10 || echo "  ✅ All dependencies appear to be up to date"
        fi

        # Check module integrity
        echo ""
        echo "Checking Go module integrity..."
        if go mod verify > /dev/null 2>&1; then
            echo "  ✅ Go module integrity verified"
        else
            echo "  ❌ Go module integrity check failed"
            vuln_exit_code=1
        fi

    } > "$dep_report" 2>&1

    log_success "Dependency scanning completed. Report: $dep_report"
    return $vuln_exit_code
}

# Run API security testing
run_api_security_testing() {
    log_info "Running API security testing..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local api_report="$SECURITY_REPORTS_DIR/api_security_$timestamp.txt"

    # Create test configuration
    local test_config="$TEMP_DIR/api-security-config.yaml"
    cat > "$test_config" << EOF
app:
  env: test
  port: 6535
  host: localhost

database:
  type: sqlite
  path: "$TEMP_DIR/api_security_test.db"
  log_level: error

logging:
  level: error
  format: json

security:
  jwt_secret: test-api-security-key
  jwt_expiry: 1h

api:
  rate_limit: 100
  cors_origins: ["*"]
EOF

    {
        echo "API Security Testing Report"
        echo "=========================="
        echo "Date: $(date)"
        echo ""

        # Start development server for API testing
        echo "Starting application for API security testing..."
        ./build/smartticket serve --config "$test_config" &
        local server_pid=$!

        # Wait for server to start
        sleep 3

        # Check if server is running
        if ! curl -s http://localhost:6535/api/v1/health > /dev/null; then
            echo "❌ Server failed to start, skipping API security tests"
            kill $server_pid 2>/dev/null || true
            return 1
        fi

        echo "✓ Server started successfully"

        # Test for common API security issues
        echo ""
        echo "Testing API endpoints for security issues..."

        # Test 1: Authentication bypass attempts
        echo "Testing authentication bypass..."

        # Try accessing protected endpoints without authentication
        local protected_endpoints=(
            "/api/v1/admin/users"
            "/api/v1/tickets"
            "/api/v1/knowledge/articles"
        )

        local auth_bypass_attempts=0
        for endpoint in "${protected_endpoints[@]}"; do
            local status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:6535$endpoint" || echo "000")
            if [[ "$status_code" == "200" ]]; then
                echo "  ⚠️  Potential auth bypass: $endpoint returned 200 without auth"
                ((auth_bypass_attempts++))
            elif [[ "$status_code" == "401" || "$status_code" == "403" ]]; then
                echo "  ✅ Properly protected: $endpoint returned $status_code"
            else
                echo "  ℹ️  $endpoint returned $status_code"
            fi
        done

        # Test 2: SQL Injection attempts
        echo ""
        echo "Testing for SQL injection vulnerabilities..."

        local sqli_payloads=(
            "'; DROP TABLE users; --"
            "' OR '1'='1"
            "'; SELECT * FROM users; --"
            "' UNION SELECT * FROM users --"
        )

        local sqli_attempts=0
        for payload in "${sqli_payloads[@]}"; do
            local encoded_payload=$(printf '%s' "$payload" | jq -sRr @uri)
            local status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:6535/api/v1/tickets?id=$encoded_payload" || echo "000")
            if [[ "$status_code" == "200" ]]; then
                echo "  ⚠️  Potential SQLi vulnerability with payload: $payload"
                ((sqli_attempts++))
            fi
        done

        if [[ $sqli_attempts -eq 0 ]]; then
            echo "  ✅ No obvious SQL injection vulnerabilities detected"
        fi

        # Test 3: XSS attempts
        echo ""
        echo "Testing for XSS vulnerabilities..."

        local xss_payloads=(
            "<script>alert('xss')</script>"
            "javascript:alert('xss')"
            "<img src=x onerror=alert('xss')>"
        )

        local xss_attempts=0
        for payload in "${xss_payloads[@]}"; do
            local status_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
                -d "{\"title\":\"$payload\",\"description\":\"Test\"}" \
                "http://localhost:6535/api/v1/tickets" || echo "000")
            if [[ "$status_code" == "200" || "$status_code" == "201" ]]; then
                echo "  ⚠️  Potential XSS vulnerability accepted payload: $payload"
                ((xss_attempts++))
            fi
        done

        if [[ $xss_attempts -eq 0 ]]; then
            echo "  ✅ No obvious XSS vulnerabilities detected"
        fi

        # Test 4: Rate limiting
        echo ""
        echo "Testing rate limiting..."

        local rapid_requests=0
        for i in {1..20}; do
            local status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:6535/api/v1/health" || echo "000")
            if [[ "$status_code" == "429" ]]; then
                ((rapid_requests++))
            fi
        done

        if [[ $rapid_requests -gt 0 ]]; then
            echo "  ✅ Rate limiting appears to be working ($rapid_requests requests blocked)"
        else
            echo "  ⚠️  Rate limiting may not be configured or may be too permissive"
        fi

        # Test 5: CORS configuration
        echo ""
        echo "Testing CORS configuration..."

        local cors_headers=$(curl -s -I -H "Origin: http://evil.com" "http://localhost:6535/api/v1/health" | grep -i "access-control-allow-origin" || echo "")
        if [[ -n "$cors_headers" ]]; then
            if [[ "$cors_headers" == *"*"* ]]; then
                echo "  ⚠️  CORS allows all origins (wildcard detected)"
            else
                echo "  ✅ CORS is configured with specific origins"
            fi
        else
            echo "  ℹ️  No CORS headers detected"
        fi

        # Stop server
        kill $server_pid 2>/dev/null || true
        wait $server_pid 2>/dev/null || true

        echo ""
        echo "API Security Testing Summary:"
        echo "  - Authentication bypass attempts: $auth_bypass_attempts"
        echo "  - Potential SQL injection issues: $sqli_attempts"
        echo "  - Potential XSS issues: $xss_attempts"
        echo "  - Rate limiting: Tested"

        # Overall assessment
        local total_issues=$((auth_bypass_attempts + sqli_attempts + xss_attempts))
        if [[ $total_issues -eq 0 ]]; then
            echo "  ✅ No critical security issues detected in API testing"
        else
            echo "  ⚠️  $total_issues potential security issues detected"
        fi

    } > "$api_report" 2>&1

    log_success "API security testing completed. Report: $api_report"
}

# Run container security scanning
run_container_security() {
    log_info "Running container security scanning..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local container_report="$SECURITY_REPORTS_DIR/container_security_$timestamp.txt"

    {
        echo "Container Security Scanning Report"
        echo "================================="
        echo "Date: $(date)"
        echo ""

        # Check if Dockerfile exists
        if [[ -f "Dockerfile" ]]; then
            echo "Analyzing Dockerfile for security issues..."

            # Check for security best practices in Dockerfile
            local security_issues=0

            # Check if running as root
            if grep -q "^USER" Dockerfile; then
                echo "  ✅ Dockerfile specifies USER (non-root execution)"
            else
                echo "  ⚠️  Dockerfile does not specify USER (potential root execution)"
                ((security_issues++))
            fi

            # Check for base image
            if grep -q "^FROM.*:latest" Dockerfile; then
                echo "  ⚠️  Dockerfile uses 'latest' tag (should use specific version)"
                ((security_issues++))
            else
                echo "  ✅ Dockerfile uses specific version tags"
            fi

            # Check for secrets
            if grep -qi "password\|secret\|key" Dockerfile; then
                echo "  ⚠️  Dockerfile may contain sensitive information"
                ((security_issues++))
            else
                echo "  ✅ No obvious secrets in Dockerfile"
            fi

            # Check for COPY instructions
            local copy_count=$(grep -c "^COPY" Dockerfile || echo "0")
            if [[ $copy_count -gt 0 ]]; then
                echo "  ℹ️  Found $copy_count COPY instructions in Dockerfile"
            fi

            echo ""
            echo "Dockerfile Security Assessment: $security_issues potential issues"

        else
            echo "ℹ️  No Dockerfile found, skipping container security analysis"
        fi

        # Run Trivy if available
        echo ""
        echo "Running container vulnerability scanning..."
        if check_command trivy; then
            if [[ -f "Dockerfile" ]]; then
                echo "Running Trivy filesystem scan..."
                if trivy fs --format table --quiet . > "$SECURITY_REPORTS_DIR/trivy_$timestamp.txt" 2>&1; then
                    echo "✓ Trivy filesystem scan completed"
                else
                    echo "⚠️ Trivy filesystem scan had issues"
                fi
            else
                echo "⚠️ No Dockerfile found for Trivy scanning"
            fi
        else
            echo "⚠️ Trivy not installed. Install with: https://aquasecurity.github.io/trivy/"
        fi

    } > "$container_report" 2>&1

    log_success "Container security scanning completed. Report: $container_report"
}

# Generate comprehensive security report
generate_security_report() {
    log_info "Generating comprehensive security report..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local comprehensive_report="$SECURITY_REPORTS_DIR/security_comprehensive_$timestamp.html"

    # Get latest security reports
    local latest_sast=$(find "$SECURITY_REPORTS_DIR" -name "sast_*.txt" -type f | sort -r | head -1)
    local latest_dependencies=$(find "$SECURITY_REPORTS_DIR" -name "dependencies_*.txt" -type f | sort -r | head -1)
    local latest_api=$(find "$SECURITY_REPORTS_DIR" -name "api_security_*.txt" -type f | sort -r | head -1)
    local latest_container=$(find "$SECURITY_REPORTS_DIR" -name "container_security_*.txt" -type f | sort -r | head -1)

    cat > "$comprehensive_report" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SmartTicket Security Test Report</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #dc3545 0%, #fd7e14 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 2.5em; }
        .header p { margin: 10px 0 0 0; opacity: 0.9; }
        .content { padding: 30px; }
        .security-section { margin: 30px 0; padding: 20px; border: 1px solid #e9ecef; border-radius: 8px; }
        .security-section h2 { color: #dc3545; margin-top: 0; }
        .security-output { background: #f8f9fa; padding: 15px; border-radius: 5px; font-family: 'Courier New', monospace; font-size: 0.9em; overflow-x: auto; max-height: 400px; overflow-y: auto; }
        .status-pass { color: #28a745; font-weight: bold; }
        .status-fail { color: #dc3545; font-weight: bold; }
        .status-warning { color: #ffc107; font-weight: bold; }
        .severity-high { color: #dc3545; font-weight: bold; }
        .severity-medium { color: #fd7e14; font-weight: bold; }
        .severity-low { color: #ffc107; font-weight: bold; }
        .summary { background: #f8d7da; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #dc3545; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔒 SmartTicket Security Test Report</h1>
            <p>Generated on $(date) | Comprehensive Security Assessment</p>
        </div>

        <div class="content">
            <div class="summary">
                <h3>🛡️ Security Assessment Summary</h3>
                <p>This report contains the results of comprehensive security testing including static analysis, dependency scanning, API security testing, and container security assessment.</p>
                <p><strong>Security Posture:</strong> Automated security validation for production readiness</p>
            </div>

            <div class="security-section">
                <h2>🔍 Static Application Security Testing (SAST)</h2>
                <div class="security-output">
EOF

    # Add SAST results
    if [[ -n "$latest_sast" ]]; then
        cat "$latest_sast" >> "$comprehensive_report"
    else
        echo "No SAST results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="security-section">
                <h2>📦 Dependency Vulnerability Scanning</h2>
                <div class="security-output">
EOF

    # Add dependency scanning results
    if [[ -n "$latest_dependencies" ]]; then
        cat "$latest_dependencies" >> "$comprehensive_report"
    else
        echo "No dependency scanning results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="security-section">
                <h2>🌐 API Security Testing</h2>
                <div class="security-output">
EOF

    # Add API security results
    if [[ -n "$latest_api" ]]; then
        cat "$latest_api" >> "$comprehensive_report"
    else
        echo "No API security testing results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="security-section">
                <h2>🐳 Container Security Assessment</h2>
                <div class="security-output">
EOF

    # Add container security results
    if [[ -n "$latest_container" ]]; then
        cat "$latest_container" >> "$comprehensive_report"
    else
        echo "No container security assessment available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="security-section">
                <h2>📁 Generated Security Reports</h2>
                <ul>
                    <li><strong>SAST Report:</strong> $(basename "$latest_sast")</li>
                    <li><strong>Dependency Report:</strong> $(basename "$latest_dependencies")</li>
                    <li><strong>API Security Report:</strong> $(basename "$latest_api")</li>
                    <li><strong>Container Security Report:</strong> $(basename "$latest_container")</li>
                </ul>
            </div>

            <div class="security-section">
                <h2>🔧 Security Recommendations</h2>
                <ul>
                    <li>Regularly update dependencies to patch known vulnerabilities</li>
                    <li>Implement proper authentication and authorization for all API endpoints</li>
                    <li>Use parameterized queries to prevent SQL injection</li>
                    <li>Validate and sanitize all user inputs</li>
                    <li>Implement rate limiting to prevent abuse</li>
                    <li>Run security scans regularly in CI/CD pipeline</li>
                    <li>Keep Docker images minimal and updated</li>
                    <li>Use non-root users in containers</li>
                </ul>
            </div>
        </div>

        <div class="footer">
            <p>Generated by SmartTicket Security Test Automation | $(date)</p>
        </div>
    </div>
</body>
</html>
EOF

    log_success "Comprehensive security report generated: $comprehensive_report"
    echo "$comprehensive_report"
}

# Clean up test environment
cleanup_security_environment() {
    log_info "Cleaning up security test environment..."

    # Kill any remaining processes
    pkill -f "smartticket.*serve" 2>/dev/null || true

    # Remove temporary files but keep reports
    rm -rf "$TEMP_DIR"

    log_success "Security test environment cleaned up"
}

# Display summary
display_security_summary() {
    local comprehensive_report="$1"

    log_success "Security testing completed!"
    echo
    echo "🔒 Security reports generated in: $SECURITY_REPORTS_DIR"
    echo
    echo "📋 Key security tests performed:"
    echo "  - Static Application Security Testing (SAST)"
    echo "  - Dependency vulnerability scanning"
    echo "  - API security testing"
    echo "  - Container security assessment"
    echo
    echo "📊 Comprehensive report: $comprehensive_report"
    echo
    echo "🛡️ Review all security reports and address any identified issues before deployment"
}

# Main execution
main() {
    echo "🔒 SmartTicket Security Testing"
    echo "=============================="
    echo

    # Check prerequisites
    if ! check_command go; then
        log_error "Go is not installed"
        exit 1
    fi

    if ! check_command curl; then
        log_error "curl is not installed (required for API security testing)"
        exit 1
    fi

    # Run security tests
    setup_security_environment

    local sast_result=0
    local dep_result=0
    local api_result=0
    local container_result=0

    run_sast_scanning || sast_result=$?
    run_dependency_scanning || dep_result=$?
    run_api_security_testing || api_result=$?
    run_container_security || container_result=$?

    local comprehensive_report=$(generate_security_report)
    cleanup_security_environment
    display_security_summary "$comprehensive_report"

    # Determine overall result
    local total_result=$((sast_result + dep_result + api_result + container_result))
    if [[ $total_result -gt 0 ]]; then
        log_warning "Security testing completed with some issues found. Review reports for details."
        exit 1
    else
        log_success "All security tests passed successfully!"
        exit 0
    fi
}

# Handle command line arguments
case "${1:-all}" in
    "sast")
        setup_security_environment
        run_sast_scanning
        ;;
    "deps"|"dependencies")
        setup_security_environment
        run_dependency_scanning
        ;;
    "api")
        setup_security_environment
        run_api_security_testing
        ;;
    "container")
        run_container_security
        ;;
    "clean")
        cleanup_security_environment
        ;;
    "all")
        main
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Security Testing Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  sast        Run static application security testing"
        echo "  deps        Run dependency vulnerability scanning"
        echo "  api         Run API security testing"
        echo "  container   Run container security assessment"
        echo "  clean       Clean up test environment"
        echo "  all         Run all security tests (default)"
        echo "  help        Show this help message"
        exit 0
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac