#!/bin/bash
# Test script for OpenSearch Query Receiver

set -e

echo "================================"
echo "OpenSearch Query Receiver Tests"
echo "================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Change to project root
cd "$(dirname "$0")/.."

# 1. Check dependencies
print_info "Checking dependencies..."
if ! command -v go &> /dev/null; then
    print_error "Go is not installed"
    exit 1
fi
print_success "Go is installed"

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi
print_success "Docker is installed"

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed"
    exit 1
fi
print_success "Docker Compose is installed"

# 2. Download dependencies
print_info "Downloading Go dependencies..."
go mod download
go mod tidy
print_success "Dependencies downloaded"

# 3. Format check
print_info "Checking code formatting..."
if [ -n "$(gofmt -l .)" ]; then
    print_error "Code is not formatted. Run 'make fmt'"
    exit 1
fi
print_success "Code is properly formatted"

# 4. Run go vet
print_info "Running go vet..."
go vet ./...
print_success "go vet passed"

# 5. Run unit tests
print_info "Running unit tests..."
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -eq 0 ]; then
    print_success "Unit tests passed"
else
    print_error "Unit tests failed"
    exit $TEST_EXIT_CODE
fi

# 6. Generate coverage report
print_info "Generating coverage report..."
go tool cover -func=coverage.out
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
print_success "Coverage: ${COVERAGE}%"

# 7. Build binary
print_info "Building binary..."
make build
if [ $? -eq 0 ]; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi

echo ""
echo "================================"
print_success "All tests passed!"
echo "================================"
echo ""
echo "Next steps:"
echo "  1. Start test environment: make docker-up"
echo "  2. Run integration tests: make test-integration"
echo "  3. Run collector: make run-direct"
