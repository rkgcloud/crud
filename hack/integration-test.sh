#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
APP_PORT=${PORT:-8080}
APP_HOST=${HOST:-localhost}
BASE_URL="http://${APP_HOST}:${APP_PORT}"
TIMEOUT=30

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to wait for application to be ready
wait_for_app() {
    print_status $YELLOW "Waiting for application to be ready..."
    local count=0
    while [ $count -lt $TIMEOUT ]; do
        if curl -s -f "${BASE_URL}/health/live" > /dev/null 2>&1; then
            print_status $GREEN "Application is ready!"
            return 0
        fi
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    print_status $RED "Application failed to start within ${TIMEOUT} seconds"
    return 1
}

# Function to test an endpoint
test_endpoint() {
    local endpoint=$1
    local description=$2
    local expected_status=${3:-200}
    
    print_status $YELLOW "Testing ${description}..."
    
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}${endpoint}")
    
    if [ "$response_code" -eq "$expected_status" ]; then
        print_status $GREEN "✓ ${description} - Status: ${response_code}"
        return 0
    else
        print_status $RED "✗ ${description} - Expected: ${expected_status}, Got: ${response_code}"
        return 1
    fi
}

# Function to test JSON endpoint
test_json_endpoint() {
    local endpoint=$1
    local description=$2
    local expected_status=${3:-200}
    
    print_status $YELLOW "Testing ${description}..."
    
    local response
    response=$(curl -s -w "\n%{http_code}" "${BASE_URL}${endpoint}")
    local response_code=$(echo "$response" | tail -n1)
    local response_body=$(echo "$response" | head -n -1)
    
    if [ "$response_code" -eq "$expected_status" ]; then
        # Try to parse JSON to verify it's valid
        if echo "$response_body" | python3 -m json.tool > /dev/null 2>&1; then
            print_status $GREEN "✓ ${description} - Status: ${response_code}, Valid JSON"
            return 0
        else
            print_status $YELLOW "✓ ${description} - Status: ${response_code}, but response is not valid JSON"
            return 0
        fi
    else
        print_status $RED "✗ ${description} - Expected: ${expected_status}, Got: ${response_code}"
        return 1
    fi
}

# Main test function
run_tests() {
    local failed=0
    
    print_status $YELLOW "Starting integration tests for CRUD application..."
    print_status $YELLOW "Base URL: ${BASE_URL}"
    
    # Wait for application to be ready
    if ! wait_for_app; then
        return 1
    fi
    
    # Test health endpoints
    test_json_endpoint "/health/live" "Liveness probe" || failed=$((failed + 1))
    test_json_endpoint "/health/ready" "Readiness probe" || failed=$((failed + 1))
    test_json_endpoint "/health/" "Health check" || failed=$((failed + 1))
    test_json_endpoint "/health/metrics" "Metrics endpoint" || failed=$((failed + 1))
    
    # Test public endpoints
    test_endpoint "/login" "Login page" || failed=$((failed + 1))
    
    # Test rate limiting (this might fail if rate limits are low)
    print_status $YELLOW "Testing rate limiting..."
    local rate_limit_failed=0
    for i in {1..70}; do
        if ! curl -s -f "${BASE_URL}/health/live" > /dev/null 2>&1; then
            print_status $GREEN "✓ Rate limiting is working (request $i was blocked)"
            break
        fi
        if [ $i -eq 70 ]; then
            print_status $YELLOW "⚠ Rate limiting may not be working (all 70 requests succeeded)"
        fi
    done
    
    # Test CORS headers
    print_status $YELLOW "Testing CORS headers..."
    local cors_headers
    cors_headers=$(curl -s -I "${BASE_URL}/health/live" | grep -i "access-control" || true)
    if [ -n "$cors_headers" ]; then
        print_status $GREEN "✓ CORS headers are present"
    else
        print_status $YELLOW "⚠ CORS headers not found (this may be expected depending on configuration)"
    fi
    
    # Test security headers
    print_status $YELLOW "Testing security headers..."
    local security_headers
    security_headers=$(curl -s -I "${BASE_URL}/health/live")
    
    local security_tests=(
        "X-Content-Type-Options:nosniff"
        "X-Frame-Options:DENY"
        "X-XSS-Protection:1; mode=block"
        "Content-Security-Policy"
    )
    
    for header in "${security_tests[@]}"; do
        if echo "$security_headers" | grep -qi "$header"; then
            print_status $GREEN "✓ Security header found: $header"
        else
            print_status $YELLOW "⚠ Security header missing: $header"
        fi
    done
    
    # Summary
    print_status $YELLOW "Integration test summary:"
    if [ $failed -eq 0 ]; then
        print_status $GREEN "All critical tests passed! 🎉"
        return 0
    else
        print_status $RED "$failed critical tests failed ❌"
        return 1
    fi
}

# Help function
show_help() {
    echo "Integration test script for CRUD application"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -p, --port     Application port (default: 8080)"
    echo "  -H, --host     Application host (default: localhost)"
    echo ""
    echo "Environment variables:"
    echo "  PORT           Application port"
    echo "  HOST           Application host"
    echo ""
    echo "Example:"
    echo "  $0 --port 3000 --host 127.0.0.1"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -p|--port)
            APP_PORT="$2"
            shift 2
            ;;
        -H|--host)
            APP_HOST="$2"
            shift 2
            ;;
        *)
            print_status $RED "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Update BASE_URL with parsed arguments
BASE_URL="http://${APP_HOST}:${APP_PORT}"

# Run the tests
run_tests
