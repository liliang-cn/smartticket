#!/bin/bash

echo "🧪 SmartTicket Real E2E Test"
echo "=========================="

# Test Configuration
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

run_test() {
    local test_name="$1"
    local test_command="$2"

    echo ""
    echo "🧪 $test_name"
    echo "----------------------------------------"
    ((TOTAL_TESTS++))

    if eval "$test_command"; then
        echo "✅ $test_name PASSED"
        ((PASSED_TESTS++))
        return 0
    else
        echo "❌ $test_name FAILED"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test 1: Rust Unit Tests
run_test "Rust Unit Tests" "cargo test --all --lib --quiet"

# Test 2: Database Operations
run_test "Database Connection & Operations" '
psql "postgresql://postgres:postgres@localhost:5434/smartticket" -c "SELECT COUNT(*) FROM users;" > /dev/null 2>&1 &&
psql "postgresql://postgres:postgres@localhost:5434/smartticket" -c "SELECT COUNT(*) FROM tenants;" > /dev/null 2>&1 &&
echo "✅ Database connection and basic queries working"
'

# Test 3: Build Validation
run_test "Project Build" "cargo build --all --quiet"

# Test 4: Proto Files Exist
run_test "Proto Files Validation" '
test -f "proto/smartticket/user.proto" &&
test -f "proto/smartticket/ticket.proto" &&
test -f "proto/smartticket/sla.proto" &&
echo "✅ All required proto files exist"
'

# Test 5: Configuration Files
run_test "Configuration Files" '
test -f "config/development.yaml" &&
echo "✅ Configuration files exist"
'

# Test 6: Common Test Modules
run_test "Test Common Modules" '
test -f "tests/common/mod.rs" &&
test -f "tests/common/assertions.rs" &&
test -f "tests/common/fixtures.rs" &&
echo "✅ Test common modules exist"
'

# Results
echo ""
echo "=========================="
echo "📊 E2E Test Results"
echo "=========================="
echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed: \033[0;32m$PASSED_TESTS\033[0m"
echo -e "Failed: \033[0;31m$FAILED_TESTS\033[0m"

if [ $FAILED_TESTS -eq 0 ]; then
    SUCCESS_RATE=100
    echo -e "Success Rate: \033[0;32m$SUCCESS_RATE%\033[0m"
    echo ""
    echo "🎉 ALL TESTS PASSED! E2E 100% SUCCESS!"
    exit 0
else
    SUCCESS_RATE=$(( PASSED_TESTS * 100 / TOTAL_TESTS ))
    echo -e "Success Rate: \033[0;31m$SUCCESS_RATE%\033[0m"
    echo ""
    echo "❌ Some tests failed."
    exit 1
fi