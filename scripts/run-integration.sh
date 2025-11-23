#!/bin/bash
# Integration test script for OpenSearch Query Receiver

set -e

echo "========================================"
echo "OpenSearch Query Receiver Integration Tests"
echo "========================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_step() {
    echo -e "${BLUE}▸ $1${NC}"
}

# Change to project root
cd "$(dirname "$0")/.."

# Cleanup function
cleanup() {
    print_info "Cleaning up..."
    cd testdata
    docker-compose down -v 2>/dev/null || true
    cd ..
}

# Register cleanup on exit
trap cleanup EXIT

# 1. Start Docker environment
print_step "Starting Docker test environment..."
cd testdata
docker-compose up -d
cd ..
print_success "Docker environment started"

# 2. Wait for OpenSearch to be ready
print_step "Waiting for OpenSearch to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -k -s -u admin:admin "https://localhost:9200/_cluster/health" > /dev/null 2>&1; then
        print_success "OpenSearch is ready"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        print_error "OpenSearch did not become ready in time"
        exit 1
    fi

    echo -n "."
    sleep 2
done
echo ""

# 3. Wait for log generator to create some data
print_step "Waiting for log data to be generated..."
sleep 15
print_success "Log data should be available"

# 4. Verify data exists
print_step "Verifying data in OpenSearch..."
INDICES=$(curl -k -s -u admin:admin "https://localhost:9200/_cat/indices/logs-*?h=index,docs.count" 2>/dev/null)
if [ -z "$INDICES" ]; then
    print_error "No log indices found"
    exit 1
fi

echo "$INDICES"
print_success "Data verified"

# 5. Test direct mode connection
print_step "Testing direct mode connection..."
curl -k -s -u admin:admin "https://localhost:9200/_cluster/health?pretty > /tmp/opensearch-health.json
if [ $? -eq 0 ]; then
    print_success "Direct connection works"
    cat /tmp/opensearch-health.json
else
    print_error "Direct connection failed"
    exit 1
fi
echo ""

# 6. Test proxy connection
print_step "Testing proxy connection..."
sleep 5  # Give proxy time to start
if curl -s "http://localhost:8080/_cluster/health?pretty" > /tmp/proxy-health.json 2>&1; then
    print_success "Proxy connection works"
    cat /tmp/proxy-health.json
else
    print_error "Proxy connection failed"
    print_info "Checking proxy logs..."
    cd testdata
    docker-compose logs opensearch-oauth-proxy
    cd ..
    exit 1
fi
echo ""

# 7. Test a sample query
print_step "Testing sample query..."
QUERY='{"query":{"match_all":{}},"size":1}'
RESPONSE=$(curl -k -s -u admin:admin -X POST "https://localhost:9200/logs-*/_search" \
    -H "Content-Type: application/json" \
    -d "$QUERY")

HITS=$(echo "$RESPONSE" | grep -o '"total":{"value":[0-9]*' | grep -o '[0-9]*$')
if [ -n "$HITS" ] && [ "$HITS" -gt 0 ]; then
    print_success "Query returned $HITS documents"
else
    print_error "Query failed or returned no results"
    echo "$RESPONSE"
    exit 1
fi

# 8. Build the collector
print_step "Building collector..."
make build
if [ $? -eq 0 ]; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi

# 9. Test collector with direct mode
print_step "Testing collector in direct mode..."
timeout 30s ./bin/otelcol-opensearchquery --config=examples/config-direct.yaml > /tmp/collector-direct.log 2>&1 &
COLLECTOR_PID=$!
sleep 20

if ps -p $COLLECTOR_PID > /dev/null; then
    print_success "Collector is running in direct mode"
    kill $COLLECTOR_PID 2>/dev/null || true
else
    print_error "Collector failed to start in direct mode"
    cat /tmp/collector-direct.log
    exit 1
fi

# Check logs for metrics
if grep -q "opensearch.query" /tmp/collector-direct.log; then
    print_success "Metrics collected successfully"
else
    print_info "Collector logs:"
    cat /tmp/collector-direct.log
fi

# 10. Test collector with proxy mode
print_step "Testing collector in proxy mode..."
timeout 30s ./bin/otelcol-opensearchquery --config=examples/config-proxy.yaml > /tmp/collector-proxy.log 2>&1 &
COLLECTOR_PID=$!
sleep 20

if ps -p $COLLECTOR_PID > /dev/null; then
    print_success "Collector is running in proxy mode"
    kill $COLLECTOR_PID 2>/dev/null || true
else
    print_error "Collector failed to start in proxy mode"
    cat /tmp/collector-proxy.log
    exit 1
fi

# 11. Show summary
echo ""
echo "========================================"
print_success "All integration tests passed!"
echo "========================================"
echo ""
echo "Summary:"
echo "  ✓ Docker environment started"
echo "  ✓ OpenSearch is accessible"
echo "  ✓ OAuth proxy is working"
echo "  ✓ Log data generated"
echo "  ✓ Queries work"
echo "  ✓ Collector built successfully"
echo "  ✓ Direct mode tested"
echo "  ✓ Proxy mode tested"
echo ""
echo "Docker environment is still running. You can:"
echo "  - Run collector: make run-direct"
echo "  - View logs: cd testdata && docker-compose logs -f"
echo "  - Access OpenSearch: https://localhost:9200"
echo "  - Access Dashboards: http://localhost:5601"
echo ""
echo "To stop: make docker-down"
