#!/bin/bash

# SmartTicket Build Script
# This script automates the build process for different environments

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT="development"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR="build"
BINARY_NAME="smartticket"

# Function to print colored output
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
Usage: $0 [OPTIONS]

Build SmartTicket binary for different environments

OPTIONS:
    -e, --env ENVIRONMENT     Build environment (development|staging|production) [default: development]
    -v, --version VERSION    Set version [default: auto-detect from git]
    -o, --output DIR         Output directory [default: build]
    -b, --binary NAME        Binary name [default: smartticket]
    -h, --help               Show this help message

EXAMPLES:
    $0                       # Build for development
    $0 -e production         # Build for production
    $0 -v v1.0.0 -o dist    # Build with custom version and output directory
    $0 --env staging --binary smartticket-staging  # Build for staging

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -b|--binary)
            BINARY_NAME="$2"
            shift 2
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

# Validate environment
case $ENVIRONMENT in
    development|staging|production)
        ;;
    *)
        print_error "Invalid environment: $ENVIRONMENT"
        print_error "Valid environments: development, staging, production"
        exit 1
        ;;
esac

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
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

    # Check if dependencies are downloaded
    if [[ ! -d "vendor" ]] && ! go mod verify &> /dev/null; then
        print_warning "Dependencies not verified. Downloading..."
        go mod download
    fi

    print_success "Prerequisites check passed"
}

# Function to set build flags
set_build_flags() {
    # Base LDFLAGS
    LDFLAGS="-ldflags \"-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT -X main.Environment=$ENVIRONMENT\""

    # Environment-specific flags
    case $ENVIRONMENT in
        production)
            LDFLAGS="$LDFLAGS -s -w"  # Strip debug information for production
            CGO_ENABLED=0
            ;;
        staging)
            LDFLAGS="$LDFLAGS -s -w"
            CGO_ENABLED=0
            ;;
        development)
            # Keep debug information for development
            CGO_ENABLED=1
            ;;
    esac

    # Target platform (default to current platform)
    GOOS=$(go env GOOS)
    GOARCH=$(go env GOARCH)

    print_info "Build configuration:"
    print_info "  Environment: $ENVIRONMENT"
    print_info "  Version: $VERSION"
    print_info "  Output: $OUTPUT_DIR"
    print_info "  Binary: $BINARY_NAME"
    print_info "  Platform: $GOOS/$GOARCH"
    print_info "  CGO_ENABLED: $CGO_ENABLED"
}

# Function to run tests
run_tests() {
    if [[ "$ENVIRONMENT" == "production" ]]; then
        print_info "Running tests for production build..."
        if ! go test -v ./...; then
            print_error "Tests failed. Cannot build for production."
            exit 1
        fi
        print_success "All tests passed"
    fi
}

# Function to run linter
run_linter() {
    if [[ "$ENVIRONMENT" == "production" ]]; then
        print_info "Running linter for production build..."
        if command -v golangci-lint &> /dev/null; then
            if ! golangci-lint run; then
                print_error "Linting failed. Cannot build for production."
                exit 1
            fi
            print_success "Linting passed"
        else
            print_warning "golangci-lint not found. Skipping linting."
        fi
    fi
}

# Function to build binary
build_binary() {
    print_info "Building binary..."

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Set environment variables
    export CGO_ENABLED=$CGO_ENABLED
    export GOOS=$GOOS
    export GOARCH=$GOARCH

    # Build command
    BUILD_CMD="go build $LDFLAGS -o $OUTPUT_DIR/$BINARY_NAME cmd/server/main.go"

    print_info "Running: $BUILD_CMD"

    if eval "$BUILD_CMD"; then
        print_success "Build completed successfully"

        # Show binary information
        if [[ -f "$OUTPUT_DIR/$BINARY_NAME" ]]; then
            BINARY_SIZE=$(ls -lh "$OUTPUT_DIR/$BINARY_NAME" | awk '{print $5}')
            print_info "Binary size: $BINARY_SIZE"

            # Try to get version info from binary
            if command -v file &> /dev/null; then
                print_info "Binary type: $(file "$OUTPUT_DIR/$BINARY_NAME")"
            fi
        fi
    else
        print_error "Build failed"
        exit 1
    fi
}

# Function to create additional artifacts
create_artifacts() {
    if [[ "$ENVIRONMENT" == "production" ]]; then
        print_info "Creating additional artifacts..."

        # Create checksum
        if command -v sha256sum &> /dev/null; then
            cd "$OUTPUT_DIR"
            sha256sum "$BINARY_NAME" > "${BINARY_NAME}.sha256"
            cd - > /dev/null
            print_success "Checksum created: ${BINARY_NAME}.sha256"
        fi

        # Create version info file
        cat > "$OUTPUT_DIR/version.json" << EOF
{
    "version": "$VERSION",
    "build_time": "$BUILD_TIME",
    "git_commit": "$GIT_COMMIT",
    "environment": "$ENVIRONMENT",
    "go_version": "$(go version)",
    "platform": "$GOOS/$GOARCH"
}
EOF
        print_success "Version info created: version.json"
    fi
}

# Function to create package
create_package() {
    if [[ "$ENVIRONMENT" == "production" ]]; then
        print_info "Creating package..."

        PACKAGE_NAME="smartticket-${VERSION}-${GOOS}-${GOARCH}"
        PACKAGE_DIR="$OUTPUT_DIR/$PACKAGE_NAME"

        mkdir -p "$PACKAGE_DIR"

        # Copy binary and artifacts
        cp "$OUTPUT_DIR/$BINARY_NAME" "$PACKAGE_DIR/"
        if [[ -f "$OUTPUT_DIR/${BINARY_NAME}.sha256" ]]; then
            cp "$OUTPUT_DIR/${BINARY_NAME}.sha256" "$PACKAGE_DIR/"
        fi
        if [[ -f "$OUTPUT_DIR/version.json" ]]; then
            cp "$OUTPUT_DIR/version.json" "$PACKAGE_DIR/"
        fi

        # Copy example configuration
        if [[ -f "configs/config.example.yaml" ]]; then
            cp configs/config.example.yaml "$PACKAGE_DIR/config.yaml"
        fi

        # Copy README
        if [[ -f "README.md" ]]; then
            cp README.md "$PACKAGE_DIR/"
        fi

        # Create archive
        cd "$OUTPUT_DIR"
        if command -v tar &> /dev/null; then
            if [[ "$GOOS" == "windows" ]]; then
                if command -v zip &> /dev/null; then
                    zip -r "${PACKAGE_NAME}.zip" "$PACKAGE_NAME"
                    print_success "Package created: ${PACKAGE_NAME}.zip"
                else
                    print_warning "zip not found. Skipping package creation."
                fi
            else
                tar -czf "${PACKAGE_NAME}.tar.gz" "$PACKAGE_NAME"
                print_success "Package created: ${PACKAGE_NAME}.tar.gz"
            fi
        fi
        cd - > /dev/null
    fi
}

# Main execution
main() {
    print_info "Starting SmartTicket build process..."

    check_prerequisites
    set_build_flags
    run_tests
    run_linter
    build_binary
    create_artifacts
    create_package

    print_success "Build process completed successfully!"
    print_info "Artifacts are available in: $OUTPUT_DIR"

    if [[ "$ENVIRONMENT" == "production" ]] && [[ -f "$OUTPUT_DIR/$BINARY_NAME" ]]; then
        print_info "To run the application:"
        print_info "  ./$OUTPUT_DIR/$BINARY_NAME serve --config config.yaml"
    fi
}

# Trap to handle interruption
trap 'print_error "Build interrupted"; exit 1' INT TERM

# Run main function
main "$@"