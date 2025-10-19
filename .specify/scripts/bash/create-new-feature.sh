#!/bin/bash

# Create New Feature Script for Speckit Framework
set -euo pipefail

# Parse arguments
JSON_OUTPUT=false
FEATURE_DESC=""
SHORT_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --short-name)
            SHORT_NAME="$2"
            shift 2
            ;;
        *)
            FEATURE_DESC="$1"
            shift
            ;;
    esac
done

# Validate inputs
if [[ -z "$FEATURE_DESC" ]]; then
    echo "Error: Feature description is required" >&2
    exit 1
fi

if [[ -z "$SHORT_NAME" ]]; then
    echo "Error: Short name is required" >&2
    exit 1
fi

# Generate branch name
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BRANCH_NAME="feature/${SHORT_NAME}-${TIMESTAMP}"

# Create feature directory
FEATURE_DIR=".specify/features/${SHORT_NAME}-${TIMESTAMP}"
mkdir -p "$FEATURE_DIR"

# Create spec file path
SPEC_FILE="${FEATURE_DIR}/spec.md"

# Initialize git branch (if in git repo)
if git rev-parse --git-dir > /dev/null 2>&1; then
    git checkout -b "$BRANCH_NAME" 2>/dev/null || true
fi

# Output JSON if requested
if [[ "$JSON_OUTPUT" == "true" ]]; then
    cat <<EOF
{
  "BRANCH_NAME": "$BRANCH_NAME",
  "SPEC_FILE": "$(pwd)/$SPEC_FILE",
  "FEATURE_DIR": "$(pwd)/$FEATURE_DIR"
}
EOF
else
    echo "Created feature:"
    echo "  Branch: $BRANCH_NAME"
    echo "  Spec: $SPEC_FILE"
fi