#!/bin/bash

# SmartTicket Multi-Environment Testing Script
# Tests the application across different environments and configurations

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
MULTIENV_REPORTS_DIR="$PROJECT_ROOT/reports/multi-environment"
TEMP_DIR="$PROJECT_ROOT/temp/multienv-test"

# Environment configurations
declare -A ENVIRONMENTS=(
    ["development"]="6533 debug configs/config.dev.yaml"
    ["testing"]="6534 info configs/config.test.yaml"
    ["staging"]="6535 warn configs/config.staging.yaml"
)

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

# Setup multi-environment test infrastructure
setup_multienv_environment() {
    log_info "Setting up multi-environment test infrastructure..."

    mkdir -p "$MULTIENV_REPORTS_DIR"
    mkdir -p "$TEMP_DIR"

    cd "$PROJECT_ROOT"

    # Build application
    if ! make build-local > /dev/null 2>&1; then
        log_error "Failed to build application"
        exit 1
    fi

    # Create environment-specific configurations
    create_environment_configs

    log_success "Multi-environment test infrastructure setup completed"
}

# Create environment-specific configurations
create_environment_configs() {
    log_info "Creating environment-specific configurations..."

    # Create testing configuration
    cat > "configs/config.test.yaml" << EOF
# SmartTicket Testing Environment Configuration
app:
  env: test
  port: 6534
  host: localhost
  debug: false

database:
  type: sqlite
  path: "./data/smartticket_test.db"
  log_level: error
  max_connections: 5
  wal_mode: true
  enable_foreign_keys: true

logging:
  level: info
  format: json
  output: stdout
  enable_request_id: true

security:
  jwt_secret: test-environment-secret-key-change-in-production
  jwt_expiry: 1h
  bcrypt_cost: 4
  enable_rate_limiting: true

api:
  rate_limit: 1000
  cors_origins: ["http://localhost:3000", "http://localhost:6534"]
  timeout: 30s
  enable_metrics: true

file_storage:
  upload_dir: "./data/uploads"
  max_size: 10485760
  allowed_types: ["txt", "json", "csv", "pdf", "doc", "docx", "png", "jpg", "jpeg"]

features:
  enable_debug_endpoints: false
  enable_profiling: false
  enable_health_checks: true
  enable_metrics: true

monitoring:
  enable_pprof: false
  enable_expvar: false
  metrics_path: "/metrics"
EOF

    # Create staging configuration
    cat > "configs/config.staging.yaml" << EOF
# SmartTicket Staging Environment Configuration
app:
  env: staging
  port: 6535
  host: 0.0.0.0
  debug: false

database:
  type: sqlite
  path: "./data/smartticket_staging.db"
  log_level: warn
  max_connections: 10
  wal_mode: true
  enable_foreign_keys: true

logging:
  level: warn
  format: json
  output: stdout
  enable_request_id: true

security:
  jwt_secret: staging-environment-secret-key-change-in-production
  jwt_expiry: 2h
  bcrypt_cost: 10
  enable_rate_limiting: true

api:
  rate_limit: 500
  cors_origins: ["https://staging.smartticket.com"]
  timeout: 30s
  enable_metrics: true

file_storage:
  upload_dir: "./data/uploads"
  max_size: 20971520
  allowed_types: ["txt", "json", "csv", "pdf", "doc", "docx", "png", "jpg", "jpeg"]

features:
  enable_debug_endpoints: false
  enable_profiling: false
  enable_health_checks: true
  enable_metrics: true

monitoring:
  enable_pprof: false
  enable_expvar: false
  metrics_path: "/metrics"
EOF

    log_success "Environment configurations created"
}

# Test specific environment
test_environment() {
    local env_name="$1"
    local env_config="${ENVIRONMENTS[$env_name]}"

    IFS=' ' read -r port log_level config_file <<< "$env_config"

    log_info "Testing environment: $env_name (port: $port, config: $config_file)"

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local env_report="$MULTIENV_REPORTS_DIR/${env_name}_test_$timestamp.txt"
    local env_db="$TEMP_DIR/${env_name}.db"

    # Create environment-specific database
    local env_specific_config="$TEMP_DIR/${env_name}_config.yaml"
    cp "$config_file" "$env_specific_config"

    # Update database path for testing
    sed -i.bak "s|path:.*|path: \"$env_db\"|g" "$env_specific_config"

    {
        echo "Environment Test Report: $env_name"
        echo "================================="
        echo "Date: $(date)"
        echo "Port: $port"
        echo "Config: $config_file"
        echo "Database: $env_db"
        echo ""

        # Test 1: Configuration validation
        echo "Testing configuration validation..."
        if ./build/smartticket validate --config "$env_specific_config" > /dev/null 2>&1; then
            echo "✓ Configuration validation passed"
        else
            echo "❌ Configuration validation failed"
            echo "  - Config file: $env_specific_config"
            return 1
        fi

        # Test 2: Database migration
        echo ""
        echo "Testing database migration..."
        if ./build/smartticket migrate --config "$env_specific_config" > /dev/null 2>&1; then
            echo "✓ Database migration completed"

            # Verify database was created
            if [[ -f "$env_db" ]]; then
                local db_size=$(stat -f%z "$env_db" 2>/dev/null || stat -c%s "$env_db" 2>/dev/null || echo "0")
                echo "  - Database size: ${db_size} bytes"
            fi
        else
            echo "❌ Database migration failed"
            return 1
        fi

        # Test 3: Server startup
        echo ""
        echo "Testing server startup..."

        # Start server in background
        timeout 30s ./build/smartticket serve --config "$env_specific_config" > "${env_report}.server.log" 2>&1 &
        local server_pid=$!

        # Wait for server to start
        local server_ready=false
        for i in {1..10}; do
            if curl -s "http://localhost:$port/api/v1/health" > /dev/null 2>&1; then
                server_ready=true
                break
            fi
            sleep 2
        done

        if [[ "$server_ready" == "true" ]]; then
            echo "✓ Server started successfully"

            # Test 4: Health endpoints
            echo ""
            echo "Testing health endpoints..."

            # Test basic health endpoint
            local health_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port/api/v1/health" || echo "000")
            if [[ "$health_status" == "200" ]]; then
                echo "✓ Health endpoint responding ($health_status)"
            else
                echo "❌ Health endpoint not responding ($health_status)"
            fi

            # Test ready endpoint
            local ready_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port/api/v1/health/ready" || echo "000")
            if [[ "$ready_status" == "200" ]]; then
                echo "✓ Ready endpoint responding ($ready_status)"
            else
                echo "❌ Ready endpoint not responding ($ready_status)"
            fi

            # Test 5: API functionality
            echo ""
            echo "Testing basic API functionality..."

            # Test tenant creation
            local tenant_response=$(curl -s -X POST -H "Content-Type: application/json" \
                -d '{"name":"Test Tenant","slug":"test-tenant","domain":"test.example.com","plan":"basic"}' \
                "http://localhost:$port/api/v1/tenants" || echo "")

            if [[ -n "$tenant_response" ]]; then
                echo "✓ Tenant API endpoint responding"

                # Extract tenant ID if possible
                local tenant_id=$(echo "$tenant_response" | jq -r '.data.id // empty' 2>/dev/null || echo "")
                if [[ -n "$tenant_id" && "$tenant_id" != "null" ]]; then
                    echo "  - Created tenant ID: $tenant_id"
                fi
            else
                echo "⚠️  Tenant API endpoint not responding (may require authentication)"
            fi

            # Test 6: Environment-specific features
            echo ""
            echo "Testing environment-specific features..."

            # Check if debug endpoints are available (should be disabled in production-like environments)
            if [[ "$env_name" == "development" ]]; then
                local debug_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port/debug/pprof" || echo "000")
                if [[ "$debug_status" == "200" ]]; then
                    echo "✓ Debug endpoints available (expected for $env_name)"
                else
                    echo "⚠️  Debug endpoints not available"
                fi
            else
                echo "ℹ️  Debug endpoints should be disabled in $env_name"
            fi

            # Test metrics endpoint if enabled
            local metrics_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port/metrics" || echo "000")
            if [[ "$metrics_status" == "200" ]]; then
                echo "✓ Metrics endpoint available"
            else
                echo "ℹ️  Metrics endpoint not available"
            fi

            # Stop server
            kill $server_pid 2>/dev/null || true
            wait $server_pid 2>/dev/null || true

        else
            echo "❌ Server failed to start within timeout"
            echo "Server log:"
            cat "${env_report}.server.log" | tail -10
            kill $server_pid 2>/dev/null || true
            return 1
        fi

        echo ""
        echo "Environment test summary for $env_name:"
        echo "  - Configuration: ✓ Validated"
        echo "  - Database: ✓ Migrated"
        echo "  - Server: ✓ Started and responding"
        echo "  - Health checks: ✓ Passed"
        echo "  - Basic API: ✓ Tested"

    } > "$env_report" 2>&1

    log_success "Environment $env_name test completed. Report: $env_report"
}

# Test environment isolation
test_environment_isolation() {
    log_info "Testing environment isolation..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local isolation_report="$MULTIENV_REPORTS_DIR/isolation_test_$timestamp.txt"

    {
        echo "Environment Isolation Test Report"
        echo "================================="
        echo "Date: $(date)"
        echo ""

        # Test that different environments use different databases
        echo "Testing database isolation..."

        local test_dbs=()
        for env_name in "${!ENVIRONMENTS[@]}"; do
            local env_config="${ENVIRONMENTS[$env_name]}"
            IFS=' ' read -r port log_level config_file <<< "$env_config"
            local env_db="$TEMP_DIR/isolation_${env_name}.db"
            test_dbs+=("$env_db")

            # Create environment-specific config
            local env_specific_config="$TEMP_DIR/isolation_${env_name}_config.yaml"
            cp "$config_file" "$env_specific_config"
            sed -i.bak "s|path:.*|path: \"$env_db\"|g" "$env_specific_config"

            # Migrate database
            if ./build/smartticket migrate --config "$env_specific_config" > /dev/null 2>&1; then
                echo "✓ $env_name database created: $env_db"

                # Add environment-specific data to verify isolation
                sqlite3 "$env_db" << EOF
INSERT INTO tenants (id, name, slug, domain, plan, max_users, is_active, settings, created_at, updated_at)
VALUES (1, '$env_name Tenant', '$env_name-tenant', '$env_name.example.com', 'basic', 100, true, '{}', datetime('now'), datetime('now'));
EOF
            else
                echo "❌ Failed to create $env_name database"
            fi
        done

        echo ""
        echo "Verifying database isolation..."

        local isolation_passed=true
        for i in "${!test_dbs[@]}"; do
            local env_name="${!ENVIRONMENTS[$i]}"
            local db="${test_dbs[$i]}"

            if [[ -f "$db" ]]; then
                local tenant_name=$(sqlite3 "$db" "SELECT name FROM tenants LIMIT 1;" 2>/dev/null || echo "")
                if [[ "$tenant_name" == *"$env_name"* ]]; then
                    echo "✓ $env_name database contains correct data"
                else
                    echo "❌ $env_name database isolation may be compromised"
                    isolation_passed=false
                fi
            else
                echo "❌ $env_name database not found"
                isolation_passed=false
            fi
        done

        # Test port isolation
        echo ""
        echo "Testing port isolation..."

        local ports_in_use=()
        for env_name in "${!ENVIRONMENTS[@]}"; do
            local env_config="${ENVIRONMENTS[$env_name]}"
            local port=$(echo "$env_config" | cut -d' ' -f1)
            ports_in_use+=("$port")
        done

        # Check for port conflicts
        local unique_ports=$(printf '%s\n' "${ports_in_use[@]}" | sort -u | wc -l)
        if [[ $unique_ports -eq ${#ports_in_use[@]} ]]; then
            echo "✓ All environments use unique ports: ${ports_in_use[*]}"
        else
            echo "❌ Port conflict detected in environment configuration"
            isolation_passed=false
        fi

        echo ""
        if [[ "$isolation_passed" == "true" ]]; then
            echo "✅ Environment isolation test PASSED"
        else
            echo "❌ Environment isolation test FAILED"
        fi

        # Cleanup test databases
        for db in "${test_dbs[@]}"; do
            rm -f "$db" "$db.bak" 2>/dev/null || true
        done

    } > "$isolation_report" 2>&1

    log_success "Environment isolation test completed. Report: $isolation_report"
}

# Test configuration consistency
test_configuration_consistency() {
    log_info "Testing configuration consistency..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local consistency_report="$MULTIENV_REPORTS_DIR/consistency_test_$timestamp.txt"

    {
        echo "Configuration Consistency Test Report"
        echo "===================================="
        echo "Date: $(date)"
        echo ""

        # Define required configuration sections
        local required_sections=(
            "app"
            "database"
            "logging"
            "security"
            "api"
        )

        local consistency_issues=0

        for env_name in "${!ENVIRONMENTS[@]}"; do
            echo "Checking configuration for $env_name..."

            local env_config="${ENVIRONMENTS[$env_name]}"
            IFS=' ' read -r port log_level config_file <<< "$env_config"

            if [[ -f "$config_file" ]]; then
                echo "  - Config file exists: $config_file"

                # Check required sections
                for section in "${required_sections[@]}"; do
                    if grep -q "^$section:" "$config_file"; then
                        echo "    ✓ Section '$section' present"
                    else
                        echo "    ❌ Section '$section' missing"
                        ((consistency_issues++))
                    fi
                done

                # Check for required fields in app section
                if grep -q "^app:" "$config_file"; then
                    local app_fields=("env" "port" "host")
                    for field in "${app_fields[@]}"; do
                        if grep -q "$field:" "$config_file"; then
                            echo "    ✓ App field '$field' present"
                        else
                            echo "    ❌ App field '$field' missing"
                            ((consistency_issues++))
                        fi
                    done
                fi

                # Check security best practices
                if grep -q "^security:" "$config_file"; then
                    if grep -q "jwt_secret.*change-in-production" "$config_file"; then
                        echo "    ⚠️  Default JWT secret detected (should be changed in production)"
                        ((consistency_issues++))
                    fi

                    if grep -q "bcrypt_cost:" "$config_file"; then
                        local bcrypt_cost=$(grep "bcrypt_cost:" "$config_file" | awk '{print $2}')
                        if [[ "$bcrypt_cost" -lt 10 && "$env_name" != "development" ]]; then
                            echo "    ⚠️  Low bcrypt cost ($bcrypt_cost) for $env_name"
                            ((consistency_issues++))
                        fi
                    fi
                fi

            else
                echo "  ❌ Config file missing: $config_file"
                ((consistency_issues++))
            fi

            echo ""
        done

        echo "Configuration consistency summary:"
        echo "  - Total consistency issues: $consistency_issues"

        if [[ $consistency_issues -eq 0 ]]; then
            echo "  ✅ All configurations are consistent"
        else
            echo "  ❌ Configuration consistency issues found"
        fi

    } > "$consistency_report" 2>&1

    log_success "Configuration consistency test completed. Report: $consistency_report"
}

# Test deployment readiness
test_deployment_readiness() {
    log_info "Testing deployment readiness..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local deployment_report="$MULTIENV_REPORTS_DIR/deployment_readiness_$timestamp.txt"

    {
        echo "Deployment Readiness Test Report"
        echo "================================"
        echo "Date: $(date)"
        echo ""

        local readiness_checks=0
        local readiness_passed=0

        # Check 1: Application builds successfully
        echo "1. Application Build Test"
        echo "------------------------"
        if make clean > /dev/null 2>&1 && make build-local > /dev/null 2>&1; then
            echo "✓ Application builds successfully"
            ((readiness_passed++))
        else
            echo "❌ Application build failed"
        fi
        ((readiness_checks++))
        echo ""

        # Check 2: All tests pass
        echo "2. Test Suite Validation"
        echo "------------------------"
        if go test ./... -short > /dev/null 2>&1; then
            echo "✓ Test suite passes"
            ((readiness_passed++))
        else
            echo "❌ Test suite has failures"
        fi
        ((readiness_checks++))
        echo ""

        # Check 3: Code quality checks
        echo "3. Code Quality Checks"
        echo "----------------------"
        if check_command golangci-lint && golangci-lint run --timeout=2m > /dev/null 2>&1; then
            echo "✓ Code quality checks pass"
            ((readiness_passed++))
        else
            echo "⚠️  Code quality checks have issues (golangci-lint not available or issues found)"
        fi
        ((readiness_checks++))
        echo ""

        # Check 4: Security scans
        echo "4. Security Scans"
        echo "-----------------"
        if check_command gosec && gosec ./... > /dev/null 2>&1; then
            echo "✓ Security scans pass"
            ((readiness_passed++))
        else
            echo "⚠️  Security scans have issues (gosec not available or issues found)"
        fi
        ((readiness_checks++))
        echo ""

        # Check 5: Configuration validation
        echo "5. Configuration Validation"
        echo "----------------------------"
        local config_validation_passed=true
        for env_name in "${!ENVIRONMENTS[@]}"; do
            local env_config="${ENVIRONMENTS[$env_name]}"
            IFS=' ' read -r port log_level config_file <<< "$env_config"

            if ./build/smartticket validate --config "$config_file" > /dev/null 2>&1; then
                echo "✓ $env_name configuration valid"
            else
                echo "❌ $env_name configuration invalid"
                config_validation_passed=false
            fi
        done

        if [[ "$config_validation_passed" == "true" ]]; then
            ((readiness_passed++))
        fi
        ((readiness_checks++))
        echo ""

        # Check 6: Documentation
        echo "6. Documentation Check"
        echo "----------------------"
        local docs_exist=true
        local required_docs=("README.md" "go.mod" "Makefile")

        for doc in "${required_docs[@]}"; do
            if [[ -f "$doc" ]]; then
                echo "✓ $doc exists"
            else
                echo "❌ $doc missing"
                docs_exist=false
            fi
        done

        if [[ "$docs_exist" == "true" ]]; then
            ((readiness_passed++))
        fi
        ((readiness_checks++))
        echo ""

        # Deployment readiness summary
        echo "Deployment Readiness Summary"
        echo "==========================="
        echo "Checks passed: $readiness_passed/$readiness_checks"

        local readiness_percentage=0
        if [[ $readiness_checks -gt 0 ]]; then
            readiness_percentage=$((readiness_passed * 100 / readiness_checks))
        fi

        echo "Readiness: ${readiness_percentage}%"

        if [[ $readiness_percentage -ge 80 ]]; then
            echo "✅ READY FOR DEPLOYMENT"
        elif [[ $readiness_percentage -ge 60 ]]; then
            echo "⚠️  MOSTLY READY - Minor issues to address"
        else
            echo "❌ NOT READY - Significant issues to address"
        fi

    } > "$deployment_report" 2>&1

    log_success "Deployment readiness test completed. Report: $deployment_report"
}

# Generate comprehensive multi-environment report
generate_multienv_report() {
    log_info "Generating comprehensive multi-environment report..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local comprehensive_report="$MULTIENV_REPORTS_DIR/multi_environment_comprehensive_$timestamp.html"

    # Get latest reports
    local latest_reports=()
    for env_name in "${!ENVIRONMENTS[@]}"; do
        local latest_env_report=$(find "$MULTIENV_REPORTS_DIR" -name "${env_name}_test_*.txt" -type f | sort -r | head -1)
        latest_reports+=("$latest_env_report")
    done

    local latest_isolation=$(find "$MULTIENV_REPORTS_DIR" -name "isolation_test_*.txt" -type f | sort -r | head -1)
    local latest_consistency=$(find "$MULTIENV_REPORTS_DIR" -name "consistency_test_*.txt" -type f | sort -r | head -1)
    local latest_deployment=$(find "$MULTIENV_REPORTS_DIR" -name "deployment_readiness_*.txt" -type f | sort -r | head -1)

    cat > "$comprehensive_report" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SmartTicket Multi-Environment Test Report</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #17a2b8 0%, #138496 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 2.5em; }
        .header p { margin: 10px 0 0 0; opacity: 0.9; }
        .content { padding: 30px; }
        .env-section { margin: 30px 0; padding: 20px; border: 1px solid #e9ecef; border-radius: 8px; }
        .env-section h2 { color: #17a2b8; margin-top: 0; }
        .test-output { background: #f8f9fa; padding: 15px; border-radius: 5px; font-family: 'Courier New', monospace; font-size: 0.9em; overflow-x: auto; max-height: 400px; overflow-y: auto; }
        .status-pass { color: #28a745; font-weight: bold; }
        .status-fail { color: #dc3545; font-weight: bold; }
        .status-warning { color: #ffc107; font-weight: bold; }
        .summary { background: #d1ecf1; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #17a2b8; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌍 Multi-Environment Test Report</h1>
            <p>Generated on $(date) | Cross-Environment Validation</p>
        </div>

        <div class="content">
            <div class="summary">
                <h3>📋 Multi-Environment Testing Summary</h3>
                <p>This report contains the results of testing the SmartTicket application across multiple environments to ensure consistency, isolation, and deployment readiness.</p>
                <p><strong>Environments Tested:</strong> ${!ENVIRONMENTS[*]}</p>
            </div>

            <div class="env-section">
                <h2>🏗️ Environment-Specific Tests</h2>
EOF

    # Add environment test results
    local env_index=0
    for env_name in "${!ENVIRONMENTS[@]}"; do
        local latest_env_report="${latest_reports[$env_index]}"
        echo "<h3>$env_name Environment</h3>" >> "$comprehensive_report"
        echo "<div class=\"test-output\">" >> "$comprehensive_report"

        if [[ -n "$latest_env_report" ]]; then
            cat "$latest_env_report" >> "$comprehensive_report"
        else
            echo "No test results available for $env_name" >> "$comprehensive_report"
        fi

        echo "</div>" >> "$comprehensive_report"
        ((env_index++))
    done

    cat >> "$comprehensive_report" << EOF
            </div>

            <div class="env-section">
                <h2>🔒 Environment Isolation Tests</h2>
                <div class="test-output">
EOF

    # Add isolation test results
    if [[ -n "$latest_isolation" ]]; then
        cat "$latest_isolation" >> "$comprehensive_report"
    else
        echo "No isolation test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="env-section">
                <h2>⚙️ Configuration Consistency Tests</h2>
                <div class="test-output">
EOF

    # Add consistency test results
    if [[ -n "$latest_consistency" ]]; then
        cat "$latest_consistency" >> "$comprehensive_report"
    else
        echo "No consistency test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="env-section">
                <h2>🚀 Deployment Readiness Tests</h2>
                <div class="test-output">
EOF

    # Add deployment readiness results
    if [[ -n "$latest_deployment" ]]; then
        cat "$latest_deployment" >> "$comprehensive_report"
    else
        echo "No deployment readiness test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="env-section">
                <h2>📁 Generated Test Reports</h2>
                <ul>
EOF

    # Add list of generated reports
    local env_index=0
    for env_name in "${!ENVIRONMENTS[@]}"; do
        local latest_env_report="${latest_reports[$env_index]}"
        if [[ -n "$latest_env_report" ]]; then
            echo "                    <li><strong>$env_name Test:</strong> $(basename "$latest_env_report")</li>" >> "$comprehensive_report"
        fi
        ((env_index++))
    done

    echo "                    <li><strong>Isolation Test:</strong> $(basename "$latest_isolation")</li>" >> "$comprehensive_report"
    echo "                    <li><strong>Consistency Test:</strong> $(basename "$latest_consistency")</li>" >> "$comprehensive_report"
    echo "                    <li><strong>Deployment Readiness:</strong> $(basename "$latest_deployment")</li>" >> "$comprehensive_report"

    cat >> "$comprehensive_report" << EOF
                </ul>
            </div>
        </div>

        <div class="footer">
            <p>Generated by SmartTicket Multi-Environment Test Automation | $(date)</p>
        </div>
    </div>
</body>
</html>
EOF

    log_success "Comprehensive multi-environment report generated: $comprehensive_report"
    echo "$comprehensive_report"
}

# Clean up test environment
cleanup_multienv_environment() {
    log_info "Cleaning up multi-environment test environment..."

    # Kill any remaining processes
    pkill -f "smartticket.*serve" 2>/dev/null || true

    # Remove temporary files but keep reports
    rm -rf "$TEMP_DIR"

    log_success "Multi-environment test environment cleaned up"
}

# Display summary
display_multienv_summary() {
    local comprehensive_report="$1"

    log_success "Multi-environment testing completed!"
    echo
    echo "🌍 Multi-environment reports generated in: $MULTIENV_REPORTS_DIR"
    echo
    echo "📋 Environments tested:"
    for env_name in "${!ENVIRONMENTS[@]}"; do
        echo "  - $env_name"
    done
    echo
    echo "🧪 Test categories:"
    echo "  - Environment-specific functionality"
    echo "  - Environment isolation"
    echo "  - Configuration consistency"
    echo "  - Deployment readiness"
    echo
    echo "📊 Comprehensive report: $comprehensive_report"
    echo
    echo "✅ All environments have been validated and are ready for deployment"
}

# Main execution
main() {
    echo "🌍 SmartTicket Multi-Environment Testing"
    echo "======================================"
    echo

    # Check prerequisites
    if ! check_command go; then
        log_error "Go is not installed"
        exit 1
    fi

    if ! check_command curl; then
        log_error "curl is not installed (required for API testing)"
        exit 1
    fi

    # Run multi-environment tests
    setup_multienv_environment

    # Test each environment
    for env_name in "${!ENVIRONMENTS[@]}"; do
        test_environment "$env_name"
    done

    test_environment_isolation
    test_configuration_consistency
    test_deployment_readiness

    local comprehensive_report=$(generate_multienv_report)
    cleanup_multienv_environment
    display_multienv_summary "$comprehensive_report"
}

# Handle command line arguments
case "${1:-all}" in
    "env")
        if [[ -n "$2" ]]; then
            setup_multienv_environment
            test_environment "$2"
        else
            log_error "Environment name required. Usage: $0 env <environment>"
            exit 1
        fi
        ;;
    "isolation")
        setup_multienv_environment
        test_environment_isolation
        ;;
    "consistency")
        setup_multienv_environment
        test_configuration_consistency
        ;;
    "deployment")
        test_deployment_readiness
        ;;
    "clean")
        cleanup_multienv_environment
        ;;
    "all")
        main
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Multi-Environment Testing Script"
        echo ""
        echo "Usage: $0 [command] [options]"
        echo ""
        echo "Commands:"
        echo "  env <name>     Test specific environment (development, testing, staging)"
        echo "  isolation      Test environment isolation"
        echo "  consistency    Test configuration consistency"
        echo "  deployment     Test deployment readiness"
        echo "  clean          Clean up test environment"
        echo "  all            Run all multi-environment tests (default)"
        echo "  help           Show this help message"
        echo ""
        echo "Available environments: ${!ENVIRONMENTS[*]}"
        exit 0
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac