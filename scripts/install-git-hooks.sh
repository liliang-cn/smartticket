#!/bin/bash

# SmartTicket Git Hooks Installation Script
# This script installs custom Git hooks for the SmartTicket project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[SETUP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SETUP]${NC} ✓ $1"
}

print_warning() {
    echo -e "${YELLOW}[SETUP]${NC} ⚠ $1"
}

print_error() {
    echo -e "${RED}[SETUP]${NC} ✗ $1"
}

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$SCRIPT_DIR/git-hooks"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

print_status "Installing Git hooks for SmartTicket..."
print_status "Project root: $PROJECT_ROOT"
print_status "Hooks directory: $HOOKS_DIR"
print_status "Git hooks directory: $GIT_HOOKS_DIR"

# Check if we're in a Git repository
if [ ! -d "$GIT_HOOKS_DIR" ]; then
    print_error "Not in a Git repository (no .git/hooks directory found)"
    exit 1
fi

# Check if hooks directory exists
if [ ! -d "$HOOKS_DIR" ]; then
    print_error "Git hooks source directory not found: $HOOKS_DIR"
    exit 1
fi

# List of hooks to install
HOOKS=(
    "pre-commit"
    "pre-push"
    "commit-msg"
    "prepare-commit-msg"
)

# Backup existing hooks if they exist
print_status "Backing up existing hooks..."
BACKUP_DIR="$GIT_HOOKS_DIR/backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

hooks_backed_up=false
for hook in "${HOOKS[@]}"; do
    if [ -f "$GIT_HOOKS_DIR/$hook" ] && [ ! -L "$GIT_HOOKS_DIR/$hook" ]; then
        cp "$GIT_HOOKS_DIR/$hook" "$BACKUP_DIR/"
        print_status "Backed up existing $hook"
        hooks_backed_up=true
    fi
done

if [ "$hooks_backed_up" = false ]; then
    print_status "No existing hooks to backup"
else
    print_success "Existing hooks backed up to $BACKUP_DIR"
fi

# Install hooks
print_status "Installing custom Git hooks..."

hooks_installed=0
for hook in "${HOOKS[@]}"; do
    source_hook="$HOOKS_DIR/$hook"
    target_hook="$GIT_HOOKS_DIR/$hook"

    if [ -f "$source_hook" ]; then
        # Remove existing hook if it exists
        if [ -f "$target_hook" ] || [ -L "$target_hook" ]; then
            rm -f "$target_hook"
        fi

        # Create symlink to the hook
        ln -s "$source_hook" "$target_hook"

        # Make sure the hook is executable
        chmod +x "$source_hook"

        print_success "Installed $hook hook"
        hooks_installed=$((hooks_installed + 1))
    else
        print_error "Hook file not found: $source_hook"
    fi
done

# Verify installation
print_status "Verifying hook installation..."
hooks_verified=0
for hook in "${HOOKS[@]}"; do
    target_hook="$GIT_HOOKS_DIR/$hook"
    if [ -L "$target_hook" ]; then
        if [ -x "$(readlink "$target_hook")" ]; then
            print_success "✓ $hook - properly installed and executable"
            hooks_verified=$((hooks_verified + 1))
        else
            print_error "✗ $hook - installed but not executable"
        fi
    else
        print_error "✗ $hook - not installed"
    fi
done

# Summary
echo ""
print_status "Installation Summary:"
echo "  Hooks processed: ${#HOOKS[@]}"
echo "  Hooks installed: $hooks_installed"
echo "  Hooks verified: $hooks_verified"

if [ "$hooks_installed" -eq "$hooks_verified" ] && [ "$hooks_installed" -gt 0 ]; then
    print_success "Git hooks installation completed successfully!"
    echo ""
    print_status "The following hooks are now active:"
    for hook in "${HOOKS[@]}"; do
        if [ -L "$GIT_HOOKS_DIR/$hook" ]; then
            echo "  - $hook"
        fi
    done
    echo ""
    print_status "Hook functionality:"
    echo "  - pre-commit:   Runs code formatting, linting, and basic tests"
    echo "  - pre-push:     Runs full test suite and comprehensive checks"
    echo "  - commit-msg:   Validates commit message format"
    echo "  - prepare-commit-msg: Helps create standardized commit messages"
    echo ""
    print_status "To disable hooks temporarily, use:"
    echo "  git commit --no-verify"
    echo "  git push --no-verify"
    echo ""
    print_warning "Note: These hooks will help maintain code quality and consistency"
else
    print_error "Git hooks installation failed"
    exit 1
fi

# Offer to run a quick test
echo ""
read -p "Would you like to run a quick test of the pre-commit hook? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Testing pre-commit hook with a temporary file..."

    # Create a temporary Go file with formatting issues
    temp_file=$(mktemp --suffix=.go)
    cat > "$temp_file" << 'EOF'
package main

import("fmt";"os")

func main(){
if len(os.Args)>1{fmt.Println(os.Args[1])
}else{
fmt.Println("Hello, World!")
}
}
EOF

    # Try to add it to git staging
    git add "$temp_file" 2>/dev/null || true

    # Try to run the pre-commit hook
    if "$GIT_HOOKS_DIR/pre-commit" 2>/dev/null; then
        print_success "Pre-commit hook test passed (expected to fail on bad code)"
    else
        print_status "Pre-commit hook correctly rejected badly formatted code"
    fi

    # Clean up
    git reset HEAD "$temp_file" 2>/dev/null || true
    rm -f "$temp_file"

    print_success "Hook test completed"
fi

print_success "Git hooks installation is complete!"