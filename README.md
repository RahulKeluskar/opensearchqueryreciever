# OpenSearch Query Receiver

An OpenTelemetry Collector receiver that queries OpenSearch and converts results into metrics.

## Features

- Execute OpenSearch queries and generate metrics from results
- Support for both direct and OAuth2 proxy connections
- Automatic time-based filtering with configurable lookback periods
- Complex query support using OpenSearch Query DSL
- Multiple queries per receiver instance

## Quick Start

See [QUICKSTART.md](QUICKSTART.md) for a 5-minute setup guide.

## Installation

```bash
# Build from source
make build

# Or use OpenTelemetry Collector Builder
make builder

# Install to $GOPATH/bin
make install
```

## Configuration

### Direct Mode (Development/Testing)

```yaml
receivers:
  opensearchquery:
    mode: direct
    endpoint: https://localhost:9200
    username: admin
    password: admin
    index_pattern: "logs-*"
    collection_interval: 60s
    queries:
      - name: error_count
        metric_name: "logs.errors"
        query:
          match:
            level: "ERROR"
        labels:
          severity: "error"
```

### Proxy Mode (Production)

```yaml
receivers:
  opensearchquery:
    mode: proxy
    proxy_endpoint: http://oauth-proxy:8080
    index_pattern: "logs-*"
    collection_interval: 60s
    queries:
      - name: app_errors
        metric_name: "app.errors"
        query:
          bool:
            must:
              - match:
                  level: "ERROR"
```

## Generated Metrics

For each query, the receiver generates:

- `{metric_name}.count` - Number of matching documents
- `{metric_name}.took_ms` - Query execution time
- `{metric_name}.shards.total` - Total shards queried
- `{metric_name}.shards.successful` - Successful shards
- `{metric_name}.shards.failed` - Failed shards

## Building

```bash
# Build the collector
make build

# Run tests
make test

# Start test environment
make docker-up

# Run the collector
make run-direct
```

## Usage Example

1. Start test environment:
   ```bash
   make docker-up
   ```

2. Build and run:
   ```bash
   make build
   make run-direct
   ```

3. View metrics in stdout

## Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `mode` | string | Yes | - | Operation mode: `direct` or `proxy` |
| `endpoint` | string | Yes (direct) | - | OpenSearch endpoint URL |
| `proxy_endpoint` | string | Yes (proxy) | - | OAuth2 proxy endpoint |
| `index_pattern` | string | Yes | - | Index pattern (e.g., `logs-*`) |
| `time_field` | string | No | `@timestamp` | Time field for filtering |
| `lookback_period` | duration | No | `5m` | Query time range |
| `collection_interval` | duration | No | `60s` | Collection frequency |
| `timeout` | duration | No | `30s` | Request timeout |

## Query Configuration

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique query identifier |
| `description` | string | No | - | Query description |
| `metric_name` | string | No | `opensearch.query.{name}` | Base metric name |
| `query` | object | Yes | - | OpenSearch Query DSL |
| `labels` | map | No | - | Additional metric labels |

## Examples

### Count Errors
```yaml
queries:
  - name: error_count
    metric_name: "logs.errors"
    query:
      match:
        level: "ERROR"
```

### Complex Boolean Query
```yaml
queries:
  - name: critical_errors
    metric_name: "production.critical"
    query:
      bool:
        must:
          - match:
              level: "ERROR"
          - match:
              environment: "production"
        filter:
          - term:
              severity.keyword: "critical"
    labels:
      alert: "true"
```

### Range Query
```yaml
queries:
  - name: slow_requests
    metric_name: "api.slow"
    query:
      bool:
        must:
          - match:
              service: "api"
        filter:
          - range:
              response_time_ms:
                gte: 1000
```

## Testing

```bash
# Unit tests
make test

# Integration tests (requires Docker)
./scripts/run-integration.sh

# Start test environment
make docker-up

# Test query
make test-query
```

## Architecture

```
Collector → Client → [Direct/Proxy] → OpenSearch
              ↓
          Scraper → Metrics Generator
```

### Components

- **Config**: Validates and stores configuration
- **Client**: HTTP client supporting direct and proxy modes
- **Scraper**: Executes queries periodically
- **Metrics Generator**: Converts results to OpenTelemetry metrics

## Troubleshooting

### Cannot connect to OpenSearch

```bash
# Verify OpenSearch is running
curl -k -u admin:admin https://localhost:9200

# Check Docker services
make docker-up
docker ps | grep opensearch
```

### No data returned

- Verify index pattern matches: `curl -k -u admin:admin 'https://localhost:9200/_cat/indices'`
- Increase `lookback_period`
- Test query in OpenSearch Dashboards

### Build errors

```bash
make clean
go mod tidy
make build
```

## Development

```bash
# Make changes
vim config.go

# Format and test
make fmt
make test

# Build and run
make build
make run-direct
```

## License

Apache License 2.0

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run `make ci` to verify
5. Submit a pull request
