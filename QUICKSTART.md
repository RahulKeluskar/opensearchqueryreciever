# Quick Start Guide

Get the OpenSearch Query Receiver running in 5 minutes!

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, but recommended)

## Step 1: Navigate to Project

```bash
cd /Users/I524916/Documents/monitoring/telegraf-stuff/telegraf-iaas-metrics-monika-boshrelease/opensearchqueryreceiver
```

## Step 2: Fix Dependencies (Important!)

The project needs a quick dependency fix. Run:

```bash
# This fixes the OpenTelemetry API version
go mod edit -require=go.opentelemetry.io/collector/component@v0.95.0
go mod edit -require=go.opentelemetry.io/collector/config/confighttp@v0.95.0
go mod edit -require=go.opentelemetry.io/collector/confmap@v0.95.0
go mod edit -require=go.opentelemetry.io/collector/consumer@v0.95.0
go mod edit -require=go.opentelemetry.io/collector/pdata@v1.2.0
go mod edit -require=go.opentelemetry.io/collector/receiver@v0.95.0

# Download dependencies
go mod tidy
```

## Step 3: Start Test Environment

```bash
# Start OpenSearch and supporting services
make docker-up

# Or manually:
cd testdata
docker-compose up -d
cd ..
```

Wait about 30-60 seconds for OpenSearch to start.

## Step 4: Verify OpenSearch is Running

```bash
curl -k -u admin:admin https://localhost:9200

# You should see OpenSearch cluster info
```

## Step 5: Build the Receiver

```bash
make build

# Or manually:
go build ./...
```

## Step 6: Run the Collector

```bash
make run-direct

# Or manually:
./bin/otelcol-opensearchquery --config=examples/config-direct.yaml
```

## What You'll See

The collector will:
1. Connect to OpenSearch at https://localhost:9200
2. Execute configured queries every 60 seconds
3. Print metrics to stdout

Example output:
```
Metric #0
Descriptor:
     -> Name: opensearch.query.error_count.count
     -> Description: Number of documents matching the query
     -> Unit: hits
NumberDataPoints #0
Data point attributes:
     -> query.name: Str(error_count)
     -> severity: Str(error)
Value: 42
```

## What's Running

After `make docker-up`, you have:

- **OpenSearch**: https://localhost:9200 (admin/admin)
- **OpenSearch Dashboards**: http://localhost:5601
- **OAuth Proxy**: http://localhost:8080
- **Log Generator**: Continuously creating test logs

## Test the Setup

### Check OpenSearch
```bash
curl -k -u admin:admin 'https://localhost:9200/_cat/indices?v'
```

### Check Logs are Being Generated
```bash
curl -k -u admin:admin 'https://localhost:9200/logs-*/_count?pretty'
```

### Test a Query
```bash
curl -k -u admin:admin -X POST 'https://localhost:9200/logs-*/_search?pretty' \
  -H 'Content-Type: application/json' \
  -d '{"query":{"match":{"level":"ERROR"}}}'
```

## Common Issues

### Can't Connect to OpenSearch

**Problem**: Connection refused or timeout

**Solution**:
```bash
# Check if OpenSearch is running
docker ps | grep opensearch

# Check logs
docker logs opensearch-test

# Restart
make docker-restart
```

### Build Errors

**Problem**: Undefined types or imports

**Solution**:
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Port Already in Use

**Problem**: Port 9200, 8080, or 5601 already in use

**Solution**:
```bash
# Find what's using the port
lsof -i :9200

# Either stop that service or edit testdata/docker-compose.yaml
# to use different ports
```

## Next Steps

### 1. Explore the Example Configurations

```bash
# View direct mode config
cat examples/config-direct.yaml

# View proxy mode config
cat examples/config-proxy.yaml

# View complete reference
cat examples/config-full.yaml
```

### 2. Customize Your Queries

Edit `examples/config-direct.yaml`:

```yaml
queries:
  # Add your custom query
  - name: my_query
    description: "My custom OpenSearch query"
    metric_name: "my.custom.metric"
    query:
      bool:
        must:
          - match:
              field_name: "search_value"
    labels:
      team: "my-team"
```

### 3. Run Tests

```bash
# Run unit tests
make test

# Run integration tests (requires Docker)
./scripts/run-integration.sh
```

### 4. View Dashboards

Open http://localhost:5601 in your browser to:
- View indexed logs
- Create visualizations
- Test queries before adding to config

### 5. Try Proxy Mode

```bash
# Stop direct mode (Ctrl+C)

# Run proxy mode
make run-proxy

# This connects through the OAuth proxy at localhost:8080
```

## Understanding the Output

Each query generates multiple metrics:

```
logs.errors.count          # Number of matching documents
logs.errors.took_ms        # Query execution time
logs.errors.shards.total   # Total shards queried
logs.errors.shards.successful  # Successful shards
logs.errors.shards.failed      # Failed shards
```

## Cleanup

When you're done:

```bash
# Stop the collector (Ctrl+C)

# Stop Docker services
make docker-down

# Or manually:
cd testdata
docker-compose down -v
cd ..
```

## Full Development Workflow

```bash
# 1. Start environment
make docker-up

# 2. Make changes to code
# ... edit files ...

# 3. Test changes
make test

# 4. Build
make build

# 5. Run
make run-direct

# 6. Stop environment when done
make docker-down
```

## Useful Make Commands

```bash
make help           # Show all available commands
make build          # Build the receiver
make test           # Run tests
make test-short     # Run tests (skip integration)
make coverage       # Generate coverage report
make docker-up      # Start Docker environment
make docker-down    # Stop Docker environment
make docker-logs    # View Docker logs
make run-direct     # Run in direct mode
make run-proxy      # Run in proxy mode
make clean          # Clean build artifacts
```

## Getting Help

1. **Check Documentation**
   - `README.md` - Comprehensive guide
   - `INSTALL.md` - Installation details
   - `PROJECT_SUMMARY.md` - Project overview

2. **View Logs**
   ```bash
   make docker-logs
   ```

3. **Check Service Health**
   ```bash
   # OpenSearch
   curl -k -u admin:admin https://localhost:9200/_cluster/health

   # Proxy
   curl http://localhost:8080/health
   ```

4. **Troubleshooting Section**
   See README.md for detailed troubleshooting steps

## Summary

You now have:
- ✅ OpenSearch running with test data
- ✅ OpenSearch Query Receiver built
- ✅ Example queries collecting metrics
- ✅ Full Docker test environment
- ✅ OAuth proxy for testing

**Time to get metrics**: < 5 minutes

**Next**: Customize queries for your use case!

---

**Questions?** Check README.md or INSTALL.md for detailed information.
