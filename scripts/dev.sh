#!/bin/bash

# SmartTicket Development Script
# This script provides a convenient way to run the development environment

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default configuration
CONFIG_FILE="configs/config.dev.yaml"
PORT="6533"
LOG_LEVEL="info"
DATA_DIR="data"
BUILD_DIR="build"
BINARY_NAME="smartticket"

# Function to print colored output
print_header() {
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN} SmartTicket Development Environment   ${NC}"
    echo -e "${CYAN}========================================${NC}"
}

print_info() {
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

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

SmartTicket development environment helper

COMMANDS:
    init            Initialize development environment
    build           Build the application
    run             Run the application (build if needed)
    test            Run tests
    watch           Watch for changes and auto-restart
    clean           Clean build artifacts
    db-reset        Reset development database
    logs            Show application logs
    help            Show this help message

OPTIONS:
    -c, --config FILE     Configuration file [default: configs/config.dev.yaml]
    -p, --port PORT       Port number [default: 6533]
    -l, --log LEVEL       Log level (debug|info|warn|error) [default: info]
    -d, --data DIR        Data directory [default: data]
    -b, --build DIR       Build directory [default: build]
    -v, --verbose         Verbose output

EXAMPLES:
    $0 init                   # Initialize environment
    $0 run                    # Build and run application
    $0 run -p 8080            # Run on port 8080
    $0 test                   # Run tests
    $0 watch                  # Watch for changes
    $0 clean                  # Clean artifacts

EOF
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        print_info "Please install Go: https://golang.org/dl/"
        exit 1
    fi

    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: $GO_VERSION"

    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]]; then
        print_error "go.mod not found. Please run this script from the project root."
        exit 1
    fi

    print_success "Prerequisites check passed"
}

# Function to initialize development environment
init_environment() {
    print_info "Initializing development environment..."

    # Create necessary directories
    mkdir -p "$DATA_DIR"
    mkdir -p "$BUILD_DIR"
    mkdir -p logs
    mkdir -p backups

    # Download dependencies
    print_info "Downloading Go dependencies..."
    go mod download
    go mod verify

    # Install development tools
    print_info "Installing development tools..."

    # Install golangci-lint if not present
    if ! command -v golangci-lint &> /dev/null; then
        print_info "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
    fi

    # Install gosec if not present
    if ! command -v gosec &> /dev/null; then
        print_info "Installing gosec..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi

    # Install watchexec if not present (for watch command)
    if ! command -v watchexec &> /dev/null; then
        print_info "Installing watchexec..."
        go install github.com/watchexec/watchexec@latest
    fi

    # Copy example configuration if not exists
    if [[ ! -f "$CONFIG_FILE" ]] && [[ -f "configs/config.example.yaml" ]]; then
        print_info "Creating configuration file from example..."
        cp configs/config.example.yaml "$CONFIG_FILE"
        print_warning "Please review and update $CONFIG_FILE as needed"
    fi

    print_success "Development environment initialized"
}

# Function to build the application
build_application() {
    print_info "Building SmartTicket..."

    # Create build directory
    mkdir -p "$BUILD_DIR"

    # Set build flags
    LDFLAGS="-ldflags \"-X main.Version=dev -X main.BuildTime=$(date +%Y-%m-%dT%H:%M:%S%z) -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.Environment=development\""

    # Build command
    BUILD_CMD="go build $LDFLAGS -o $BUILD_DIR/$BINARY_NAME cmd/server/main.go"

    if [[ "$VERBOSE" == "true" ]]; then
        print_info "Running: $BUILD_CMD"
    fi

    if eval "$BUILD_CMD"; then
        print_success "Build completed"

        # Show binary info
        if [[ -f "$BUILD_DIR/$BINARY_NAME" ]]; then
            BINARY_SIZE=$(ls -lh "$BUILD_DIR/$BINARY_NAME" | awk '{print $5}')
            print_info "Binary: $BUILD_DIR/$BINARY_NAME ($BINARY_SIZE)"
        fi
    else
        print_error "Build failed"
        exit 1
    fi
}

# Function to run the application
run_application() {
    print_info "Starting SmartTicket..."

    # Check if binary exists
    if [[ ! -f "$BUILD_DIR/$BINARY_NAME" ]]; then
        print_info "Binary not found. Building first..."
        build_application
    fi

    # Set environment variables
    export PORT="$PORT"
    export LOG_LEVEL="$LOG_LEVEL"
    export DATA_DIR="$DATA_DIR"

    print_info "Configuration:"
    print_info "  Config: $CONFIG_FILE"
    print_info "  Port: $PORT"
    print_info "  Log Level: $LOG_LEVEL"
    print_info "  Data Directory: $DATA_DIR"

    # Check if port is available
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "Port $PORT is already in use"
        print_info "You can change the port with: $0 run -p <port>"
        return 1
    fi

    print_info "Starting SmartTicket on http://localhost:$PORT"
    print_info "Press Ctrl+C to stop the server"

    # Run the application
    exec "$BUILD_DIR/$BINARY_NAME" serve --config "$CONFIG_FILE"
}

# Function to run tests
run_tests() {
    print_info "Running tests..."

    # Set environment for testing
    export DATA_DIR="${DATA_DIR}_test"

    # Create test data directory
    mkdir -p "$DATA_DIR"

    if [[ "$VERBOSE" == "true" ]]; then
        go test -v ./...
    else
        go test ./...
    fi

    print_success "All tests passed"
}

# Function to watch for changes
watch_changes() {
    print_info "Watching for changes..."

    if ! command -v watchexec &> /dev/null; then
        print_error "watchexec not found. Please run: $0 init"
        exit 1
    fi

    print_info "Auto-restarting on file changes..."
    print_info "Press Ctrl+C to stop watching"

    # Watch for changes and rebuild/restart
    watchexec -r -e go --exts go -- "$0 run"
}

# Function to clean artifacts
clean_artifacts() {
    print_info "Cleaning build artifacts..."

    # Remove build directory
    if [[ -d "$BUILD_DIR" ]]; then
        rm -rf "$BUILD_DIR"
        print_info "Removed $BUILD_DIR"
    fi

    # Remove temporary files
    find . -name "*.tmp" -delete 2>/dev/null || true
    find . -name "*.log" -delete 2>/dev/null || true
    find . -name "*.swp" -delete 2>/dev/null || true
    find . -name "*.swo" -delete 2>/dev/null || true

    print_success "Cleanup completed"
}

# Function to reset database
reset_database() {
    print_info "Resetting development database..."

    if [[ -f "$DATA_DIR/smartticket_dev.db" ]]; then
        rm "$DATA_DIR/smartticket_dev.db"
        print_info "Removed development database"
    fi

    if [[ -f "$DATA_DIR/smartticket_test.db" ]]; then
        rm "$DATA_DIR/smartticket_test.db"
        print_info "Removed test database"
    fi

    print_success "Database reset completed"
}

# Function to show logs
show_logs() {
    print_info "Showing application logs..."

    if [[ -f "logs/smartticket.log" ]]; then
        tail -f logs/smartticket.log
    else
        print_warning "Log file not found: logs/smartticket.log"
        print_info "Make sure logging is configured in $CONFIG_FILE"
    fi
}

# Function to show status
show_status() {
    print_header

    print_info "Project Status:"
    echo "  Go Version: $(go version)"
    echo "  Working Directory: $(pwd)"
    echo "  Git Branch: $(git branch --show-current 2>/dev/null || echo 'not a git repository')"
    echo "  Git Commit: $(git rev-parse --short HEAD 2>/dev/null || echo 'not a git repository')"
    echo ""

    if [[ -f "$BUILD_DIR/$BINARY_NAME" ]]; then
        BINARY_SIZE=$(ls -lh "$BUILD_DIR/$BINARY_NAME" | awk '{print $5}')
        echo "  Binary Status: Built ($BINARY_SIZE)"
        echo "  Binary Path: $BUILD_DIR/$BINARY_NAME"
    else
        echo "  Binary Status: Not built"
    fi

    if [[ -f "$CONFIG_FILE" ]]; then
        echo "  Config File: $CONFIG_FILE ✓"
    else
        echo "  Config File: $CONFIG_FILE ✗"
    fi

    if [[ -d "$DATA_DIR" ]]; then
        DB_FILES=$(find "$DATA_DIR" -name "*.db" 2>/dev/null | wc -l)
        echo "  Database Files: $DB_FILES in $DATA_DIR"
    else
        echo "  Data Directory: Not created"
    fi

    # Check if port is in use
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "  Port $PORT: In use"
    else
        echo "  Port $PORT: Available"
    fi

    echo ""
}

# Parse command line arguments
COMMAND=""
VERBOSE="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        init|build|run|test|watch|clean|db-reset|logs|status|help)
            COMMAND="$1"
            shift
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -l|--log)
            LOG_LEVEL="$2"
            shift 2
            ;;
        -d|--data)
            DATA_DIR="$2"
            shift 2
            ;;
        -b|--build)
            BUILD_DIR="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="true"
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Default command if none specified
if [[ -z "$COMMAND" ]]; then
    COMMAND="status"
fi

# Show status first for certain commands
if [[ "$COMMAND" == "status" ]] || [[ "$VERBOSE" == "true" ]]; then
    show_status
fi

# Execute command
case $COMMAND in
    init)
        check_prerequisites
        init_environment
        ;;
    build)
        check_prerequisites
        build_application
        ;;
    run)
        check_prerequisites
        run_application
        ;;
    test)
        check_prerequisites
        run_tests
        ;;
    watch)
        check_prerequisites
        watch_changes
        ;;
    clean)
        clean_artifacts
        ;;
    db-reset)
        reset_database
        ;;
    logs)
        show_logs
        ;;
    status)
        # Status already shown above
        ;;
    help)
        show_usage
        ;;
    *)
        print_error "Unknown command: $COMMAND"
        show_usage
        exit 1
        ;;
esac