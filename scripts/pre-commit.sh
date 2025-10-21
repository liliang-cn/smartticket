#!/bin/bash

# SmartTicket Pre-commit Hook
# Ensures code quality before commits

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check if required tools are installed
check_tools() {
    print_status "Checking required tools..."

    local tools=("go" "gofmt" "golangci-lint" "gosec")
    local missing_tools=()

    for tool in "${tools[@]}"; do
        if ! command_exists "$tool"; then
            missing_tools+=("$tool")
        fi
    done

    if [ ${#missing_tools[@]} -gt 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        print_status "Install missing tools with: make install-tools"
        exit 1
    fi

    print_success "All required tools are installed"
}

# Run go fmt
run_fmt() {
    print_status "Running go fmt..."

    local unformatted_files
    unformatted_files=$(gofmt -s -l .)

    if [ -n "$unformatted_files" ]; then
        print_warning "Code is not properly formatted:"
        echo "$unformatted_files"

        read -p "Do you want to format the files now? (y/N): " -n 1 -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Formatting code..."
            gofmt -s -w .
            print_success "Code formatted successfully"
        else
            print_error "Code formatting required. Aborting commit."
            exit 1
        fi
    else
        print_success "Code is properly formatted"
    fi
}

# Run go vet
run_vet() {
    print_status "Running go vet..."

    if ! go vet ./...; then
        print_error "go vet found issues. Please fix them before committing."
        exit 1
    fi

    print_success "go vet passed"
}

# Run golangci-lint
run_lint() {
    print_status "Running golangci-lint..."

    if ! command_exists golangci-lint; then
        print_warning "golangci-lint not found. Skipping lint checks."
        return
    fi

    if ! golangci-lint run --timeout=5m; then
        print_error "golangci-lint found issues. Please fix them before committing."
        exit 1
    fi

    print_success "golangci-lint passed"
}

# Run gosec security scan
run_security_scan() {
    print_status "Running security scan..."

    if ! command_exists gosec; then
        print_warning "gosec not found. Skipping security scan."
        return
    fi

    if ! gosec ./...; then
        print_error "gosec found security issues. Please review and fix them before committing."
        exit 1
    fi

    print_success "Security scan passed"
}

# Run tests
run_tests() {
    print_status "Running tests..."

    # Check if there are any Go test files
    if ! find . -name "*_test.go" -type f -not -path "./vendor/*" | head -1 >/dev/null; then
        print_warning "No test files found. Skipping tests."
        return
    fi

    # Run short tests for pre-commit to keep it fast
    if ! go test -short ./...; then
        print_error "Tests failed. Please fix them before committing."
        exit 1
    fi

    print_success "Tests passed"
}

# Check for large files that shouldn't be committed
check_file_sizes() {
    print_status "Checking file sizes..."

    # Check for files larger than 10MB
    local large_files
    large_files=$(find . -type f -not -path "./vendor/*" -not -path "./.git/*" -not -path "./build/*" -not -path "./dist/*" -size +10M)

    if [ -n "$large_files" ]; then
        print_warning "Found large files that may not need to be committed:"
        echo "$large_files"

        read -p "Continue with commit? (y/N): " -n 1 -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Large files found. Aborting commit."
            exit 1
        fi
    fi

    print_success "File size check passed"
}

# Check for sensitive data
check_sensitive_data() {
    print_status "Checking for sensitive data..."

    local sensitive_patterns=(
        "password"
        "secret"
        "key"
        "token"
        "credential"
        "private"
    )

    local found_issues=false

    # Check for potential API keys, passwords, etc.
    for pattern in "${sensitive_patterns[@]}"; do
        local matches
        matches=$(find . -type f -not -path "./vendor/*" -not -path "./.git/*" -not -path "./build/*" -not -path "./dist/*" -exec grep -l -i "$pattern" {} \; 2>/dev/null || true)

        if [ -n "$matches" ]; then
            print_warning "Found potential sensitive data with pattern '$pattern':"
            echo "$matches"
            found_issues=true
        fi
    done

    # Check for common secret file patterns
    local secret_files=(
        ".env"
        "*.pem"
        "*.key"
        "*.crt"
        "*.p12"
        "id_rsa"
        "id_ed25519"
    )

    for pattern in "${secret_files[@]}"; do
        local matches
        matches=$(find . -name "$pattern" -not -path "./.git/*" -not -path "./vendor/*" 2>/dev/null || true)

        if [ -n "$matches" ]; then
            print_warning "Found secret files that may not need to be committed:"
            echo "$matches"
            found_issues=true
        fi
    done

    if [ "$found_issues" = true ]; then
        read -p "Sensitive data found. Continue with commit? (y/N): " -n 1 -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Sensitive data found. Aborting commit."
            exit 1
        fi
    fi

    print_success "Sensitive data check passed"
}

# Check for binary/executable files
check_binaries() {
    print_status "Checking for binary files..."

    local binary_files
    binary_files=$(find . -type f -executable -not -path "./vendor/*" -not -path "./.git/*" -not -path "./build/*" -not -path "./dist/*" 2>/dev/null || true)

    if [ -n "$binary_files" ]; then
        print_warning "Found executable files that may not need to be committed:"
        echo "$binary_files"

        read -p "Continue with commit? (y/N): " -n 1 -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Executable files found. Aborting commit."
            exit 1
        fi
    fi

    print_success "Binary file check passed"
}

# Check for TODO/FIXME comments that should be addressed
check_todos() {
    print_status "Checking for TODO/FIXME comments..."

    local todo_count
    todo_count=$(find . -name "*.go" -type f -not -path "./vendor/*" -exec grep -l -i "TODO\|FIXME\|HACK\|XXX" {} \; 2>/dev/null | wc -l || true)

    if [ "$todo_count" -gt 0 ]; then
        print_warning "Found $todo_count TODO/FIXME comments in Go files"
        print_status "Consider addressing them before committing"
        # Don't block the commit for todos, just warn
    else
        print_success "No TODO/FIXME comments found"
    fi
}

# Check Go module consistency
check_go_modules() {
    print_status "Checking Go module consistency..."

    # Check if go.mod and go.sum are consistent
    if ! go mod verify; then
        print_error "go.mod and go.sum are inconsistent. Run 'go mod tidy' and try again."
        exit 1
    fi

    # Check for unused dependencies
    local unused_deps
    unused_deps=$(go mod tidy -x 2>&1 | grep "unused" || true)

    if [ -n "$unused_deps" ]; then
        print_warning "Found unused dependencies:"
        echo "$unused_deps"
        print_status "Consider running 'go mod tidy' to clean up"
    fi

    print_success "Go module consistency check passed"
}

# Check for potential breaking changes
check_breaking_changes() {
    print_status "Checking for breaking changes..."

    # Check if there are changes to public APIs
    local api_changes
    api_changes=$(git diff --cached --name-only --diff-filter=ACM | grep -E "\.(go)$" | head -10 || true)

    if [ -n "$api_changes" ]; then
        print_warning "Potential API changes detected:"
        echo "$api_changes"
        print_status "Review these changes carefully for breaking changes"
    else
        print_success "No obvious breaking changes detected"
    fi
}

# Check if tests are up to date with code changes
check_test_coverage() {
    print_status "Checking test coverage for changed files..."

    # Get list of changed Go files
    local changed_files
    changed_files=$(git diff --cached --name-only --diff-filter=ACM | grep -E "\.(go)$" | head -5 || true)

    if [ -n "$changed_files" ]; then
        print_warning "Changed Go files:"
        echo "$changed_files"

        # Check if corresponding test files exist
        local missing_tests=()
        for file in $changed_files; do
            local test_file="${file%.go}_test.go"
            if [ ! -f "$test_file" ]; then
                missing_tests+=("$test_file")
            fi
        done

        if [ ${#missing_tests[@]} -gt 0 ]; then
            print_warning "Missing test files:"
            printf '%s\n' "${missing_tests[@]}"
            print_status "Consider adding tests for new functionality"
        else
            print_success "All changed files have corresponding test files"
        fi
    else
        print_success "No Go files changed in this commit"
    fi
}

# Main execution
main() {
    print_status "🚀 SmartTicket Pre-commit Hook"
    print_status "=================================="

    # Check if we're in a git repository
    if [ ! -d ".git" ]; then
        print_error "Not in a git repository. Skipping pre-commit checks."
        exit 0
    fi

    # Check if this is a commit or push
    local commit_type="commit"
    if [ "$2" = "push" ]; then
        commit_type="push"
    fi

    # Run all checks
    check_tools
    check_go_modules
    run_fmt
    run_vet
    run_lint
    run_security_scan
    run_tests
    check_file_sizes
    check_sensitive_data
    check_binaries
    check_todos
    check_breaking_changes
    check_test_coverage

    print_status "=================================="
    print_success "✅ All pre-commit checks passed! Ready to $commit_type."
}

# Handle arguments
case "${1:-hook}" in
    "hook")
        main
        ;;
    "push")
        main
        ;;
    "fmt")
        run_fmt
        ;;
    "lint")
        run_lint
        ;;
    "test")
        run_tests
        ;;
    "security")
        run_security_scan
        ;;
    "all")
        check_tools
        check_go_modules
        run_fmt
        run_vet
        run_lint
        run_security_scan
        run_tests
        check_file_sizes
        check_sensitive_data
        check_binaries
        check_todos
        check_breaking_changes
        check_test_coverage
        ;;
    "help"|"-h"|"--help")
        echo "SmartTicket Pre-commit Hook"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  hook     Run full pre-commit checks (default)"
        echo "  push     Run pre-push checks"
        echo "  fmt      Run go fmt"
        echo "  lint     Run golangci-lint"
        echo "  test     Run tests"
        echo "  security Run security scan"
        echo "  all      Run all checks"
        echo "  help     Show this help message"
        exit 0
        ;;
    *)
        echo "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac