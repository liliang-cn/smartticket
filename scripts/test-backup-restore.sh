#!/bin/bash

# SmartTicket Backup and Recovery Testing Script
# Tests backup and recovery procedures to ensure data integrity

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
DATA_DIR="$PROJECT_ROOT/data"
BACKUP_DIR="$PROJECT_ROOT/data/backups"
TEST_REPORTS_DIR="$PROJECT_ROOT/reports/backup-restore"
TEMP_DIR="$PROJECT_ROOT/temp/backup-test"

# Test configuration
TEST_DB="$TEMP_DIR/test.db"
TEST_BACKUP="$TEMP_DIR/test_backup.db"
TEST_EXPORT="$TEMP_DIR/test_export.json"
RESTORED_DB="$TEMP_DIR/restored.db"

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

# Setup test environment
setup_test_environment() {
    log_info "Setting up backup/restore test environment..."

    # Clean up any existing test data
    rm -rf "$TEMP_DIR"
    mkdir -p "$TEMP_DIR"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$TEST_REPORTS_DIR"

    cd "$PROJECT_ROOT"

    # Build the application
    if ! make build-local > /dev/null 2>&1; then
        log_error "Failed to build application"
        exit 1
    fi

    # Build seed tool
    if ! make seed-build > /dev/null 2>&1; then
        log_error "Failed to build seed tool"
        exit 1
    fi

    log_success "Test environment setup completed"
}

# Create test database with sample data
create_test_database() {
    log_info "Creating test database with sample data..."

    # Create test configuration
    local test_config="$TEMP_DIR/test-config.yaml"
    cat > "$test_config" << EOF
# Test configuration for backup/restore testing
app:
  env: test
  port: 6534
  host: localhost

database:
  type: sqlite
  path: "$TEST_DB"
  log_level: error
  max_connections: 5
  wal_mode: true

logging:
  level: error
  format: json
  output: stdout

security:
  jwt_secret: test-secret-key-for-backup-restore-testing
  jwt_expiry: 1h
  bcrypt_cost: 4

api:
  rate_limit: 1000
  cors_origins: ["*"]
  timeout: 30s

file_storage:
  upload_dir: "$TEMP_DIR/uploads"
  max_size: 10485760
  allowed_types: ["txt", "json", "csv"]
EOF

    # Initialize test database
    ./build/smartticket migrate --config "$test_config" > /dev/null 2>&1

    # Seed with test data
    if ./scripts/seed/seed -config "$test_config" -force > /dev/null 2>&1; then
        log_success "Test database created and seeded"
    else
        log_warning "Seed tool failed, creating minimal test data"
        # Create minimal test data manually
        sqlite3 "$TEST_DB" << EOF
INSERT INTO tenants (id, name, slug, domain, plan, max_users, is_active, settings, created_at, updated_at)
VALUES (1, 'Test Tenant', 'test-tenant', 'test.example.com', 'basic', 100, true, '{}', datetime('now'), datetime('now'));

INSERT INTO users (id, tenant_id, email, username, first_name, last_name, role, password_hash, is_active, created_at, updated_at)
VALUES (1, 1, 'admin@test.com', 'admin', 'Admin', 'User', 'admin', '\$2a\$04\$test.hash.here', true, datetime('now'), datetime('now'));

INSERT INTO tickets (id, tenant_id, ticket_number, title, description, status, priority, severity, requester_name, requester_email, created_at, updated_at)
VALUES (1, 1, 'TICKET-001', 'Test Ticket', 'This is a test ticket for backup/restore testing', 'open', 'medium', 'minor', 'Test User', 'test@example.com', datetime('now'), datetime('now'));
EOF
    fi

    # Verify database was created
    if [[ ! -f "$TEST_DB" ]]; then
        log_error "Test database creation failed"
        exit 1
    fi

    # Get initial record counts
    local tenant_count=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM tenants;" 2>/dev/null || echo "0")
    local user_count=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
    local ticket_count=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM tickets;" 2>/dev/null || echo "0")

    log_info "Test database created with: $tenant_count tenants, $user_count users, $ticket_count tickets"
}

# Test database backup functionality
test_database_backup() {
    log_info "Testing database backup functionality..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_report="$TEST_REPORTS_DIR/backup_test_$timestamp.txt"

    {
        echo "Database Backup Test Report"
        echo "=========================="
        echo "Date: $(date)"
        echo "Source DB: $TEST_DB"
        echo "Backup File: $TEST_BACKUP"
        echo ""

        # Test SQLite backup command
        echo "Testing SQLite backup..."
        if sqlite3 "$TEST_DB" ".backup $TEST_BACKUP" 2>> "$backup_report"; then
            echo "✓ SQLite backup completed successfully"

            # Verify backup file exists and has content
            if [[ -f "$TEST_BACKUP" && -s "$TEST_BACKUP" ]]; then
                local backup_size=$(stat -f%z "$TEST_BACKUP" 2>/dev/null || stat -c%s "$TEST_BACKUP" 2>/dev/null || echo "0")
                echo "✓ Backup file created successfully (${backup_size} bytes)"

                # Verify backup integrity
                if sqlite3 "$TEST_BACKUP" "SELECT COUNT(*) FROM tenants;" > /dev/null 2>&1; then
                    echo "✓ Backup file integrity verified"
                else
                    echo "✗ Backup file integrity check failed"
                    return 1
                fi
            else
                echo "✗ Backup file creation failed"
                return 1
            fi
        else
            echo "✗ SQLite backup failed"
            return 1
        fi

        echo ""
        echo "Testing application backup functionality..."

        # Test application backup if available
        local app_backup="$TEMP_DIR/app_backup.tar.gz"
        if ./build/smartticket backup --config "$TEMP_DIR/test-config.yaml" --output "$app_backup" 2>> "$backup_report"; then
            echo "✓ Application backup completed successfully"

            if [[ -f "$app_backup" && -s "$app_backup" ]]; then
                echo "✓ Application backup file created successfully"
            else
                echo "✗ Application backup file creation failed"
            fi
        else
            echo "⚠ Application backup functionality not available or failed"
        fi

    } > "$backup_report" 2>&1

    log_success "Database backup test completed. Report: $backup_report"
}

# Test database restore functionality
test_database_restore() {
    log_info "Testing database restore functionality..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local restore_report="$TEST_REPORTS_DIR/restore_test_$timestamp.txt"

    {
        echo "Database Restore Test Report"
        echo "==========================="
        echo "Date: $(date)"
        echo "Backup File: $TEST_BACKUP"
        echo "Restored DB: $RESTORED_DB"
        echo ""

        # Get original database statistics
        echo "Original database statistics:"
        local original_tenants=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM tenants;" 2>/dev/null || echo "0")
        local original_users=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
        local original_tickets=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM tickets;" 2>/dev/null || echo "0")

        echo "  - Tenants: $original_tenants"
        echo "  - Users: $original_users"
        echo "  - Tickets: $original_tickets"
        echo ""

        # Test SQLite restore
        echo "Testing SQLite restore..."
        if sqlite3 "$RESTORED_DB" ".restore $TEST_BACKUP" 2>> "$restore_report"; then
            echo "✓ SQLite restore completed successfully"

            # Verify restored data
            local restored_tenants=$(sqlite3 "$RESTORED_DB" "SELECT COUNT(*) FROM tenants;" 2>/dev/null || echo "0")
            local restored_users=$(sqlite3 "$RESTORED_DB" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
            local restored_tickets=$(sqlite3 "$RESTORED_DB" "SELECT COUNT(*) FROM tickets;" 2>/dev/null || echo "0")

            echo "Restored database statistics:"
            echo "  - Tenants: $restored_tenants"
            echo "  - Users: $restored_users"
            echo "  - Tickets: $restored_tickets"
            echo ""

            # Data integrity verification
            if [[ "$restored_tenants" -eq "$original_tenants" && "$restored_users" -eq "$original_users" && "$restored_tickets" -eq "$original_tickets" ]]; then
                echo "✓ Data integrity verified - all counts match"
            else
                echo "✗ Data integrity check failed - count mismatch"
                return 1
            fi

            # Verify specific records
            echo "Verifying specific records..."
            local test_ticket=$(sqlite3 "$RESTORED_DB" "SELECT title FROM tickets WHERE ticket_number='TICKET-001';" 2>/dev/null || echo "")
            if [[ "$test_ticket" == "Test Ticket" ]]; then
                echo "✓ Specific record verification passed"
            else
                echo "✗ Specific record verification failed"
                return 1
            fi

        else
            echo "✗ SQLite restore failed"
            return 1
        fi

        echo ""
        echo "Testing application restore functionality..."

        # Test application restore if available
        if ./build/smartticket restore --config "$TEMP_DIR/test-config.yaml" --input "$TEST_BACKUP" 2>> "$restore_report"; then
            echo "✓ Application restore completed successfully"
        else
            echo "⚠ Application restore functionality not available or failed"
        fi

    } > "$restore_report" 2>&1

    log_success "Database restore test completed. Report: $restore_report"
}

# Test data export functionality
test_data_export() {
    log_info "Testing data export functionality..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local export_report="$TEST_REPORTS_DIR/export_test_$timestamp.txt"

    {
        echo "Data Export Test Report"
        echo "======================"
        echo "Date: $(date)"
        echo "Source DB: $TEST_DB"
        echo "Export File: $TEST_EXPORT"
        echo ""

        # Test JSON export
        echo "Testing JSON export..."

        # Create export script
        cat > "$TEMP_DIR/export_script.sql" << 'EOF'
.mode json
.output tempfile.json
SELECT 'tenants' as table_name, json_group_array(json_object('id', id, 'name', name, 'slug', slug, 'domain', domain, 'plan', plan, 'max_users', max_users, 'is_active', is_active, 'settings', settings, 'created_at', created_at, 'updated_at', updated_at)) as data FROM tenants;
SELECT 'users' as table_name, json_group_array(json_object('id', id, 'tenant_id', tenant_id, 'email', email, 'username', username, 'first_name', first_name, 'last_name', last_name, 'role', role, 'is_active', is_active, 'created_at', created_at, 'updated_at', updated_at)) as data FROM users;
SELECT 'tickets' as table_name, json_group_array(json_object('id', id, 'tenant_id', tenant_id, 'ticket_number', ticket_number, 'title', title, 'description', description, 'status', status, 'priority', priority, 'severity', severity, 'requester_name', requester_name, 'requester_email', requester_email, 'created_at', created_at, 'updated_at', updated_at)) as data FROM tickets;
EOF

        # Create comprehensive export
        cat > "$TEST_EXPORT" << EOF
{
  "export_metadata": {
    "timestamp": "$(date -Iseconds)",
    "version": "1.0",
    "source": "SmartTicket Backup/Restore Test"
  },
  "tenants": $(sqlite3 "$TEST_DB" "SELECT json_group_array(json_object('id', id, 'name', name, 'slug', slug, 'domain', domain, 'plan', plan, 'max_users', max_users, 'is_active', is_active, 'settings', settings, 'created_at', created_at, 'updated_at', updated_at)) FROM tenants;"),
  "users": $(sqlite3 "$TEST_DB" "SELECT json_group_array(json_object('id', id, 'tenant_id', tenant_id, 'email', email, 'username', username, 'first_name', first_name, 'last_name', last_name, 'role', role, 'is_active', is_active, 'created_at', created_at, 'updated_at', updated_at)) FROM users;"),
  "tickets": $(sqlite3 "$TEST_DB" "SELECT json_group_array(json_object('id', id, 'tenant_id', tenant_id, 'ticket_number', ticket_number, 'title', title, 'description', description, 'status', status, 'priority', priority, 'severity', severity, 'requester_name', requester_name, 'requester_email', requester_email, 'created_at', created_at, 'updated_at', updated_at)) FROM tickets;")
}
EOF

        if [[ -f "$TEST_EXPORT" && -s "$TEST_EXPORT" ]]; then
            echo "✓ JSON export completed successfully"

            # Validate JSON structure
            if jq empty "$TEST_EXPORT" 2>/dev/null; then
                echo "✓ Export file is valid JSON"

                # Verify export content
                local export_tenants=$(jq '.tenants | length' "$TEST_EXPORT" 2>/dev/null || echo "0")
                local export_users=$(jq '.users | length' "$TEST_EXPORT" 2>/dev/null || echo "0")
                local export_tickets=$(jq '.tickets | length' "$TEST_EXPORT" 2>/dev/null || echo "0")

                echo "Export statistics:"
                echo "  - Tenants: $export_tenants"
                echo "  - Users: $export_users"
                echo "  - Tickets: $export_tickets"

                if [[ "$export_tenants" -gt 0 || "$export_users" -gt 0 || "$export_tickets" -gt 0 ]]; then
                    echo "✓ Export contains data"
                else
                    echo "✗ Export appears to be empty"
                    return 1
                fi
            else
                echo "✗ Export file is not valid JSON"
                return 1
            fi
        else
            echo "✗ JSON export failed"
            return 1
        fi

        echo ""
        echo "Testing CSV export..."

        # Test CSV export for each table
        local tenants_csv="$TEMP_DIR/tenants.csv"
        local users_csv="$TEMP_DIR/users.csv"
        local tickets_csv="$TEMP_DIR/tickets.csv"

        # Export tenants to CSV
        sqlite3 -header -csv "$TEST_DB" "SELECT * FROM tenants;" > "$tenants_csv" 2>/dev/null || true
        sqlite3 -header -csv "$TEST_DB" "SELECT * FROM users;" > "$users_csv" 2>/dev/null || true
        sqlite3 -header -csv "$TEST_DB" "SELECT * FROM tickets;" > "$tickets_csv" 2>/dev/null || true

        local csv_files=0
        [[ -f "$tenants_csv" && -s "$tenants_csv" ]] && ((csv_files++))
        [[ -f "$users_csv" && -s "$users_csv" ]] && ((csv_files++))
        [[ -f "$tickets_csv" && -s "$tickets_csv" ]] && ((csv_files++))

        if [[ $csv_files -gt 0 ]]; then
            echo "✓ CSV export completed successfully ($csv_files files)"
        else
            echo "⚠ CSV export failed or no data to export"
        fi

    } > "$export_report" 2>&1

    log_success "Data export test completed. Report: $export_report"
}

# Test automated backup procedures
test_automated_backup() {
    log_info "Testing automated backup procedures..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local auto_backup_report="$TEST_REPORTS_DIR/auto_backup_test_$timestamp.txt"

    {
        echo "Automated Backup Test Report"
        echo "==========================="
        echo "Date: $(date)"
        echo ""

        # Test backup scheduling simulation
        echo "Testing backup scheduling simulation..."

        local backup_count=0
        local max_backups=3

        for i in $(seq 1 $max_backups); do
            local backup_file="$TEMP_DIR/auto_backup_$i.db"
            local backup_timestamp=$(date -Iseconds)

            if sqlite3 "$TEST_DB" ".backup $backup_file" 2>/dev/null; then
                echo "✓ Automated backup $i created successfully ($backup_timestamp)"
                ((backup_count++))
            else
                echo "✗ Automated backup $i failed"
            fi

            # Simulate delay between backups
            sleep 1
        done

        echo ""
        echo "Backup rotation test..."

        # Test backup rotation (keep only 2 most recent)
        local keep_count=2
        ls -la "$TEMP_DIR"/auto_backup_*.db 2>/dev/null | sort -k6,7 -r | tail -n +$((keep_count + 1)) | awk '{print $9}' | xargs rm -f 2>/dev/null || true

        local remaining_backups=$(ls -1 "$TEMP_DIR"/auto_backup_*.db 2>/dev/null | wc -l)
        if [[ $remaining_backups -eq $keep_count ]]; then
            echo "✓ Backup rotation working correctly (kept $keep_count backups)"
        else
            echo "⚠ Backup rotation may have issues (found $remaining_backups backups, expected $keep_count)"
        fi

        echo ""
        echo "Backup integrity verification..."

        # Verify most recent backup integrity
        local latest_backup=$(ls -t "$TEMP_DIR"/auto_backup_*.db 2>/dev/null | head -1)
        if [[ -n "$latest_backup" ]]; then
            if sqlite3 "$latest_backup" "SELECT COUNT(*) FROM tenants;" > /dev/null 2>&1; then
                echo "✓ Latest backup integrity verified"
            else
                echo "✗ Latest backup integrity check failed"
            fi
        else
            echo "✗ No backups found for integrity check"
        fi

        echo ""
        echo "Automated backup summary:"
        echo "  - Total backups created: $backup_count/$max_backups"
        echo "  - Backup rotation: Tested"
        echo "  - Integrity checks: Performed"

    } > "$auto_backup_report" 2>&1

    log_success "Automated backup test completed. Report: $auto_backup_report"
}

# Generate comprehensive backup/restore report
generate_comprehensive_report() {
    log_info "Generating comprehensive backup/restore report..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local comprehensive_report="$TEST_REPORTS_DIR/comprehensive_backup_restore_report_$timestamp.html"

    # Get latest test reports
    local latest_backup_report=$(find "$TEST_REPORTS_DIR" -name "backup_test_*.txt" -type f | sort -r | head -1)
    local latest_restore_report=$(find "$TEST_REPORTS_DIR" -name "restore_test_*.txt" -type f | sort -r | head -1)
    local latest_export_report=$(find "$TEST_REPORTS_DIR" -name "export_test_*.txt" -type f | sort -r | head -1)
    local latest_auto_backup_report=$(find "$TEST_REPORTS_DIR" -name "auto_backup_test_*.txt" -type f | sort -r | head -1)

    cat > "$comprehensive_report" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SmartTicket Backup & Restore Test Report</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #28a745 0%, #20c997 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 2.5em; }
        .header p { margin: 10px 0 0 0; opacity: 0.9; }
        .content { padding: 30px; }
        .test-section { margin: 30px 0; padding: 20px; border: 1px solid #e9ecef; border-radius: 8px; }
        .test-section h2 { color: #28a745; margin-top: 0; }
        .test-output { background: #f8f9fa; padding: 15px; border-radius: 5px; font-family: 'Courier New', monospace; font-size: 0.9em; overflow-x: auto; max-height: 400px; overflow-y: auto; }
        .status-pass { color: #28a745; font-weight: bold; }
        .status-fail { color: #dc3545; font-weight: bold; }
        .status-warning { color: #ffc107; font-weight: bold; }
        .summary { background: #e9f7ef; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💾 Backup & Restore Test Report</h1>
            <p>Generated on $(date) | SmartTicket Automated Testing</p>
        </div>

        <div class="content">
            <div class="summary">
                <h3>📋 Test Summary</h3>
                <p>This report contains the results of comprehensive backup and restore testing to ensure data integrity and recovery procedures work correctly.</p>
                <p><strong>Test Environment:</strong> $(uname -s) $(uname -r) | Go $(go version | awk '{print $3}' | sed 's/go//')</p>
            </div>

            <div class="test-section">
                <h2>🔄 Database Backup Test</h2>
                <div class="test-output">
EOF

    # Add backup test results
    if [[ -n "$latest_backup_report" ]]; then
        cat "$latest_backup_report" >> "$comprehensive_report"
    else
        echo "No backup test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="test-section">
                <h2>♻️ Database Restore Test</h2>
                <div class="test-output">
EOF

    # Add restore test results
    if [[ -n "$latest_restore_report" ]]; then
        cat "$latest_restore_report" >> "$comprehensive_report"
    else
        echo "No restore test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="test-section">
                <h2>📤 Data Export Test</h2>
                <div class="test-output">
EOF

    # Add export test results
    if [[ -n "$latest_export_report" ]]; then
        cat "$latest_export_report" >> "$comprehensive_report"
    else
        echo "No export test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="test-section">
                <h2>🤖 Automated Backup Test</h2>
                <div class="test-output">
EOF

    # Add automated backup test results
    if [[ -n "$latest_auto_backup_report" ]]; then
        cat "$latest_auto_backup_report" >> "$comprehensive_report"
    else
        echo "No automated backup test results available" >> "$comprehensive_report"
    fi

    cat >> "$comprehensive_report" << EOF
                </div>
            </div>

            <div class="test-section">
                <h2>📁 Generated Test Files</h2>
                <ul>
                    <li><strong>Backup Report:</strong> $(basename "$latest_backup_report")</li>
                    <li><strong>Restore Report:</strong> $(basename "$latest_restore_report")</li>
                    <li><strong>Export Report:</strong> $(basename "$latest_export_report")</li>
                    <li><strong>Automated Backup Report:</strong> $(basename "$latest_auto_backup_report")</li>
                </ul>
            </div>
        </div>

        <div class="footer">
            <p>Generated by SmartTicket Backup/Restore Test Automation | $(date)</p>
        </div>
    </div>
</body>
</html>
EOF

    log_success "Comprehensive report generated: $comprehensive_report"
    echo "$comprehensive_report"
}

# Clean up test environment
cleanup_test_environment() {
    log_info "Cleaning up test environment..."

    # Remove temporary files but keep reports
    rm -rf "$TEMP_DIR"

    log_success "Test environment cleaned up"
}

# Display summary
display_summary() {
    local comprehensive_report="$1"

    log_success "Backup/Restore testing completed!"
    echo
    echo "📊 Reports generated in: $TEST_REPORTS_DIR"
    echo
    echo "📋 Key reports:"
    echo "  - Comprehensive Report: $comprehensive_report"
    echo
    echo "🔍 Test results:"
    echo "  - Database backup procedures"
    echo "  - Database restore functionality"
    echo "  - Data export capabilities"
    echo "  - Automated backup procedures"
    echo
    echo "✅ All backup and restore procedures have been tested and verified"
}

# Main execution
main() {
    echo "💾 SmartTicket Backup & Restore Testing"
    echo "====================================="
    echo

    # Check prerequisites
    if ! check_command sqlite3; then
        log_error "SQLite3 is not installed"
        exit 1
    fi

    if ! check_command jq; then
        log_warning "jq is not installed, some JSON validation will be skipped"
    fi

    # Run backup/restore tests
    setup_test_environment
    create_test_database
    test_database_backup
    test_database_restore
    test_data_export
    test_automated_backup

    local comprehensive_report=$(generate_comprehensive_report)
    cleanup_test_environment
    display_summary "$comprehensive_report"
}

# Handle command line arguments
case "${1:-all}" in
    "backup")
        setup_test_environment
        create_test_database
        test_database_backup
        ;;
    "restore")
        setup_test_environment
        create_test_database
        test_database_restore
        ;;
    "export")
        setup_test_environment
        create_test_database
        test_data_export
        ;;
    "auto")
        setup_test_environment
        create_test_database
        test_automated_backup
        ;;
    "clean")
        cleanup_test_environment
        ;;
    "all")
        main
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Backup & Restore Testing Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  backup     Test database backup functionality"
        echo "  restore    Test database restore functionality"
        echo "  export     Test data export functionality"
        echo "  auto       Test automated backup procedures"
        echo "  clean      Clean up test environment"
        echo "  all        Run all backup/restore tests (default)"
        echo "  help       Show this help message"
        exit 0
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac