package opensearchqueryreceiver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.uber.org/zap"
)

func TestNewOpenSearchClient(t *testing.T) {
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "https://localhost:9200",
			Timeout:  30 * time.Second,
		},
		Mode:           "direct",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 5 * time.Minute,
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)

	if err != nil {
		t.Fatalf("NewOpenSearchClient() failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewOpenSearchClient() returned nil client")
	}

	if client.config != config {
		t.Error("Client config doesn't match provided config")
	}

	if client.httpClient == nil {
		t.Error("Client httpClient is nil")
	}
}

func TestExecuteQuery(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Check for basic auth
		username, password, ok := r.BasicAuth()
		if ok {
			if username != "admin" || password != "admin" {
				t.Errorf("Expected basic auth admin/admin, got %s/%s", username, password)
			}
		}

		// Return mock response
		response := SearchResponse{
			Took:     42,
			TimedOut: false,
			Shards: ShardInfo{
				Total:      5,
				Successful: 5,
				Skipped:    0,
				Failed:     0,
			},
			Hits: Hits{
				Total: HitsTotal{
					Value:    100,
					Relation: "eq",
				},
				Hits: []Hit{
					{
						Index: "logs-2024.01.01",
						ID:    "1",
						Source: map[string]interface{}{
							"message": "test log",
							"level":   "INFO",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: server.URL,
			Timeout:  30 * time.Second,
		},
		Mode:           "direct",
		Username:       "admin",
		Password:       "admin",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 5 * time.Minute,
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("NewOpenSearchClient() failed: %v", err)
	}

	// Execute query
	queryConfig := QueryConfig{
		Name: "test_query",
		Query: map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	ctx := context.Background()
	response, err := client.ExecuteQuery(ctx, queryConfig)

	if err != nil {
		t.Fatalf("ExecuteQuery() failed: %v", err)
	}

	// Verify response
	if response.Took != 42 {
		t.Errorf("Expected took=42, got %d", response.Took)
	}

	if response.Hits.Total.Value != 100 {
		t.Errorf("Expected total hits=100, got %d", response.Hits.Total.Value)
	}

	if len(response.Hits.Hits) != 1 {
		t.Errorf("Expected 1 hit, got %d", len(response.Hits.Hits))
	}

	if response.Shards.Total != 5 {
		t.Errorf("Expected 5 total shards, got %d", response.Shards.Total)
	}

	if response.Shards.Successful != 5 {
		t.Errorf("Expected 5 successful shards, got %d", response.Shards.Successful)
	}
}

func TestExecuteQueryError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	// Create client
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: server.URL,
			Timeout:  30 * time.Second,
		},
		Mode:           "direct",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 5 * time.Minute,
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("NewOpenSearchClient() failed: %v", err)
	}

	// Execute query
	queryConfig := QueryConfig{
		Name: "test_query",
		Query: map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	ctx := context.Background()
	_, err = client.ExecuteQuery(ctx, queryConfig)

	if err == nil {
		t.Fatal("ExecuteQuery() expected error, got nil")
	}

	// Error should mention status code
	if !contains(err.Error(), "500") {
		t.Errorf("Error should mention status code 500: %v", err)
	}
}

func TestPing(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/_cluster/health" {
			t.Errorf("Expected path /_cluster/health, got %s", r.URL.Path)
		}

		// Return mock response
		response := map[string]interface{}{
			"cluster_name": "test-cluster",
			"status":       "green",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: server.URL,
			Timeout:  30 * time.Second,
		},
		Mode:           "direct",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 5 * time.Minute,
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("NewOpenSearchClient() failed: %v", err)
	}

	// Ping
	ctx := context.Background()
	err = client.Ping(ctx)

	if err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}
}

func TestPingError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": "Service unavailable"}`))
	}))
	defer server.Close()

	// Create client
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: server.URL,
			Timeout:  30 * time.Second,
		},
		Mode:           "direct",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 5 * time.Minute,
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("NewOpenSearchClient() failed: %v", err)
	}

	// Ping
	ctx := context.Background()
	err = client.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() expected error, got nil")
	}

	// Error should mention status code
	if !contains(err.Error(), "503") && !contains(err.Error(), "failed") {
		t.Errorf("Error should mention failure or status code: %v", err)
	}
}

func TestAddTimeRangeFilter(t *testing.T) {
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "https://localhost:9200",
			Timeout:  30 * time.Second,
		},
		Mode:           "direct",
		IndexPattern:   "logs-*",
		TimeField:      "@timestamp",
		LookbackPeriod: 5 * time.Minute,
	}

	logger := zap.NewNop()
	client, err := NewOpenSearchClient(config, logger)
	if err != nil {
		t.Fatalf("NewOpenSearchClient() failed: %v", err)
	}

	// Create a simple query
	query := map[string]interface{}{
		"match": map[string]interface{}{
			"level": "ERROR",
		},
	}

	// Add time range filter
	filteredQuery := client.addTimeRangeFilter(query)

	// Verify the structure
	boolQuery, ok := filteredQuery["bool"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected filtered query to have 'bool' key")
	}

	must, ok := boolQuery["must"].([]interface{})
	if !ok || len(must) == 0 {
		t.Fatal("Expected 'must' clause with original query")
	}

	filter, ok := boolQuery["filter"].([]interface{})
	if !ok || len(filter) == 0 {
		t.Fatal("Expected 'filter' clause with time range")
	}

	// Verify time range filter
	timeFilter, ok := filter[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected time filter to be a map")
	}

	rangeQuery, ok := timeFilter["range"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'range' in time filter")
	}

	timeRange, ok := rangeQuery["@timestamp"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected '@timestamp' range")
	}

	if _, ok := timeRange["gte"]; !ok {
		t.Error("Expected 'gte' in time range")
	}

	if _, ok := timeRange["lte"]; !ok {
		t.Error("Expected 'lte' in time range")
	}
}
