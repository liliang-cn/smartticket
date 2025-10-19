#!/bin/bash

# Setup Plan Script for Speckit Framework
set -euo pipefail

# Parse arguments
JSON_OUTPUT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Find the most recent feature directory
FEATURE_DIR=$(find .specify/features -name "golang-initialization-*" -type d | sort | tail -1)
if [[ -z "$FEATURE_DIR" ]]; then
    echo "Error: No golang-initialization feature directory found" >&2
    exit 1
fi

# Set paths
FEATURE_SPEC="${FEATURE_DIR}/spec.md"
IMPL_PLAN="${FEATURE_DIR}/plan.md"
SPECS_DIR="$(pwd)/.specify"

# Get current branch
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")

# Create plan.md from template
if [[ ! -f "$IMPL_PLAN" ]]; then
    cp "${SPECS_DIR}/templates/plan-template.md" "$IMPL_PLAN" 2>/dev/null || echo "# Implementation Plan\n\n## Technical Context\n\n## Constitution Check\n\n## Phase 0: Research\n\n## Phase 1: Design & Contracts\n\n## Phase 2: Implementation\n" > "$IMPL_PLAN"
fi

# Output JSON if requested
if [[ "$JSON_OUTPUT" == "true" ]]; then
    cat <<EOF
{
  "FEATURE_SPEC": "$(pwd)/${FEATURE_SPEC}",
  "IMPL_PLAN": "$(pwd)/${IMPL_PLAN}",
  "SPECS_DIR": "$(pwd)/${SPECS_DIR}",
  "BRANCH": "$BRANCH"
}
EOF
else
    echo "Plan setup complete:"
    echo "  Feature Spec: $FEATURE_SPEC"
    echo "  Implementation Plan: $IMPL_PLAN"
    echo "  Specs Dir: $SPECS_DIR"
    echo "  Branch: $BRANCH"
fi