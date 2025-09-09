#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_status $YELLOW "Running Go linting checks..."

# Check formatting
print_status $YELLOW "Checking code formatting..."
UNFORMATTED=$(gofmt -s -l . | grep -v vendor/ || true)
if [ -n "$UNFORMATTED" ]; then
    print_status $RED "Code is not properly formatted:"
    echo "$UNFORMATTED"
    print_status $RED "Run 'gofmt -s -w .' to fix formatting issues"
    exit 1
fi
print_status $GREEN "✓ Code formatting is correct"

# Run go vet
print_status $YELLOW "Running go vet..."
if ! go vet ./...; then
    print_status $RED "✗ go vet found issues"
    exit 1
fi
print_status $GREEN "✓ go vet passed"

# Check for common issues
print_status $YELLOW "Checking for inefficient assignments..."
if command -v ineffassign >/dev/null 2>&1; then
    if ! ineffassign ./...; then
        print_status $RED "✗ ineffassign found issues"
        exit 1
    fi
    print_status $GREEN "✓ ineffassign passed"
else
    print_status $YELLOW "⚠ ineffassign not installed, skipping"
fi

# Check for misspellings
print_status $YELLOW "Checking for misspellings..."
if command -v misspell >/dev/null 2>&1; then
    if ! misspell -error .; then
        print_status $RED "✗ misspell found issues"
        exit 1
    fi
    print_status $GREEN "✓ misspell passed"
else
    print_status $YELLOW "⚠ misspell not installed, skipping"
fi

# Check for security issues with gosec if available
print_status $YELLOW "Checking for security issues..."
if command -v gosec >/dev/null 2>&1; then
    if ! gosec -quiet ./...; then
        print_status $RED "✗ gosec found security issues"
        exit 1
    fi
    print_status $GREEN "✓ gosec passed"
else
    print_status $YELLOW "⚠ gosec not installed, skipping security scan"
fi

print_status $GREEN "All linting checks passed! 🎉"
