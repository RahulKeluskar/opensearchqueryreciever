// +build integration

package opensearchqueryreceiver

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.uber.org/zap"
)

// Integration tests require OpenSearch to be running
// Run with: go test -tags=integration ./...

func TestIntegrationDirectMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "https://localhost:9200",
			Timeout:  30 * time.Second,
		},
		CollectionInterval: 60 * time.Second,
		Mode:           "direct",
		Username:       "admin",
		Password:       "admin",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 24 * time.Hour, // Look back 24 hours for test data
		Queries: []QueryConfig{
			{
				Name:        "all_logs",
				Description: "Count all logs",
				MetricName:  "logs.total",
				Query: map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test ping
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Test query execution
	resp, err := client.ExecuteQuery(ctx, config.Queries[0])
	if err != nil {
		t.Fatalf("ExecuteQuery failed: %v", err)
	}

	t.Logf("Query executed successfully:")
	t.Logf("  Total hits: %d", resp.Hits.Total.Value)
	t.Logf("  Returned hits: %d", len(resp.Hits.Hits))
	t.Logf("  Took: %dms", resp.Took)
	t.Logf("  Shards - Total: %d, Successful: %d, Failed: %d",
		resp.Shards.Total, resp.Shards.Successful, resp.Shards.Failed)

	// Verify response structure
	if resp.Shards.Total == 0 {
		t.Error("Expected at least one shard")
	}

	if resp.Shards.Failed > 0 {
		t.Errorf("Expected no failed shards, got %d", resp.Shards.Failed)
	}
}

func TestIntegrationProxyMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout: 30 * time.Second,
		},
		CollectionInterval: 60 * time.Second,
		Mode:           "proxy",
		ProxyEndpoint:  "http://localhost:8080",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 24 * time.Hour,
		Queries: []QueryConfig{
			{
				Name:        "all_logs",
				Description: "Count all logs via proxy",
				MetricName:  "logs.total",
				Query: map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			},
		},
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test ping through proxy
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping through proxy failed: %v", err)
	}

	// Test query execution through proxy
	resp, err := client.ExecuteQuery(ctx, config.Queries[0])
	if err != nil {
		t.Fatalf("ExecuteQuery through proxy failed: %v", err)
	}

	t.Logf("Query executed through proxy successfully:")
	t.Logf("  Total hits: %d", resp.Hits.Total.Value)
	t.Logf("  Took: %dms", resp.Took)
}

func TestIntegrationAggregations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "https://localhost:9200",
			Timeout:  30 * time.Second,
		},
		CollectionInterval: 60 * time.Second,
		Mode:           "direct",
		Username:       "admin",
		Password:       "admin",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 24 * time.Hour,
		Queries: []QueryConfig{
			{
				Name:        "logs_by_level",
				Description: "Count logs grouped by level",
				MetricName:  "logs.by_level",
				Query: map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				Labels: map[string]string{
					"source": "integration_test",
				},
			},
		},
	}

	// Add aggregation to the query
	config.Queries[0].Query = map[string]interface{}{
		"match_all": map[string]interface{}{},
		"aggs": map[string]interface{}{
			"levels": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "level.keyword",
					"size":  10,
				},
			},
		},
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	resp, err := client.ExecuteQuery(ctx, config.Queries[0])
	if err != nil {
		t.Fatalf("ExecuteQuery with aggregations failed: %v", err)
	}

	t.Logf("Query with aggregations executed successfully:")
	t.Logf("  Total hits: %d", resp.Hits.Total.Value)

	if resp.Aggregations != nil {
		t.Logf("  Aggregations found: %d", len(resp.Aggregations))
		for aggName := range resp.Aggregations {
			t.Logf("    - %s", aggName)
		}
	} else {
		t.Log("  No aggregations in response")
	}
}
