package opensearchqueryreceiver

// This file serves as the main package documentation and exports

// OpenSearch Query Receiver is a custom OpenTelemetry Collector receiver
// that queries OpenSearch indices and converts the results into metrics.
//
// Key Features:
// - Dual operational modes: direct and proxy
// - Support for complex OpenSearch queries with aggregations
// - Automatic time-range filtering based on lookback period
// - Conversion of query results to OpenTelemetry metrics
// - Configurable collection intervals
//
// Operational Modes:
//
// 1. Direct Mode:
//    - Connects directly to OpenSearch cluster
//    - Supports basic authentication (username/password)
//    - Best for development and testing environments
//    - Example: mode: "direct", endpoint: "https://opensearch.example.com:9200"
//
// 2. Proxy Mode:
//    - Connects through an OAuth2 authentication proxy
//    - The proxy handles OAuth2 token management and renewal
//    - Best for production environments requiring OAuth2
//    - Example: mode: "proxy", proxy_endpoint: "http://oauth-proxy:8080"
//
// Configuration Example:
//
//	receivers:
//	  opensearchquery:
//	    mode: direct
//	    endpoint: https://localhost:9200
//	    username: admin
//	    password: admin
//	    index_pattern: "logs-*"
//	    collection_interval: 60s
//	    lookback_period: 5m
//	    queries:
//	      - name: error_count
//	        description: "Count of error logs"
//	        metric_name: "logs.errors"
//	        query:
//	          match:
//	            level: "ERROR"
//	        labels:
//	          severity: "error"
//
// The receiver will execute each configured query at the specified interval
// and emit metrics with the query results. Each query produces multiple metrics:
// - {metric_name}.count: Number of documents matching the query
// - {metric_name}.took_ms: Query execution time
// - {metric_name}.shards.*: Shard statistics
// - {metric_name}.agg.*: Aggregation results (if present)
