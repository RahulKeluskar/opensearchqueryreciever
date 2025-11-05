package opensearchqueryreceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// OpenSearchClient handles communication with OpenSearch
// Supports both direct connections and proxy-based connections
type OpenSearchClient struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewOpenSearchClient creates a new OpenSearch client
func NewOpenSearchClient(cfg *Config, logger *zap.Logger) (*OpenSearchClient, error) {
	// Create HTTP client with configured timeout
	httpClient := &http.Client{
		Timeout: cfg.ClientConfig.Timeout,
	}

	// Always configure TLS settings if using HTTPS endpoints
	// LoadTLSConfig handles insecure_skip_verify even without cert files
	tlsConfig, err := cfg.ClientConfig.TLS.LoadTLSConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	httpClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &OpenSearchClient{
		config:     cfg,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// SearchRequest represents the request payload for OpenSearch queries
type SearchRequest struct {
	Query map[string]interface{} `json:"query"`
	Size  int                    `json:"size,omitempty"`
	Aggs  map[string]interface{} `json:"aggs,omitempty"`
}

// SearchResponse represents the response from OpenSearch
type SearchResponse struct {
	Took         int                    `json:"took"`
	TimedOut     bool                   `json:"timed_out"`
	Shards       ShardInfo              `json:"_shards"`
	Hits         Hits                   `json:"hits"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// ShardInfo contains information about shard success/failure
type ShardInfo struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

// Hits contains the search results
type Hits struct {
	Total    HitsTotal `json:"total"`
	MaxScore *float64  `json:"max_score"`
	Hits     []Hit     `json:"hits"`
}

// HitsTotal contains the total number of hits
type HitsTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// Hit represents a single search result
type Hit struct {
	Index  string                 `json:"_index"`
	ID     string                 `json:"_id"`
	Score  *float64               `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

// ExecuteQuery executes a query against OpenSearch and returns the response
func (c *OpenSearchClient) ExecuteQuery(ctx context.Context, query QueryConfig) (*SearchResponse, error) {
	// Build the search request
	searchReq := SearchRequest{
		Query: query.Query,
		Size:  10000, // Maximum results per query
	}

	// Add time range filter if needed
	searchReq.Query = c.addTimeRangeFilter(searchReq.Query)

	// Marshal request body
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	c.logger.Debug("Executing OpenSearch query",
		zap.String("query_name", query.Name),
		zap.String("index_pattern", c.config.IndexPattern),
		zap.String("request_body", string(reqBody)),
	)

	// Build the URL
	url := fmt.Sprintf("%s/%s/_search", c.config.GetEndpoint(), c.config.IndexPattern)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Add authentication if in direct mode
	if c.config.UsesBasicAuth() {
		req.SetBasicAuth(c.config.Username, c.config.Password)
		c.logger.Debug("Using basic authentication")
	} else {
		c.logger.Debug("No authentication configured")
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Debug("Query executed successfully",
		zap.String("query_name", query.Name),
		zap.Int64("total_hits", searchResp.Hits.Total.Value),
		zap.Int("returned_hits", len(searchResp.Hits.Hits)),
		zap.Int("took_ms", searchResp.Took),
	)

	return &searchResp, nil
}

// addTimeRangeFilter adds a time range filter to the query based on lookback period
func (c *OpenSearchClient) addTimeRangeFilter(query map[string]interface{}) map[string]interface{} {
	now := time.Now()
	startTime := now.Add(-c.config.LookbackPeriod)

	// Create time range filter
	timeFilter := map[string]interface{}{
		"range": map[string]interface{}{
			c.config.TimeField: map[string]interface{}{
				"gte": startTime.Format(time.RFC3339),
				"lte": now.Format(time.RFC3339),
			},
		},
	}

	// Wrap the query in a bool query with the time filter
	wrappedQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				query,
			},
			"filter": []interface{}{
				timeFilter,
			},
		},
	}

	return wrappedQuery
}

// Ping checks if the OpenSearch cluster is reachable
func (c *OpenSearchClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/_cluster/health", c.config.GetEndpoint())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	// Add authentication if in direct mode
	if c.config.UsesBasicAuth() {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping OpenSearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ping failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Successfully connected to OpenSearch",
		zap.String("mode", c.config.Mode),
		zap.String("endpoint", c.config.GetEndpoint()),
	)

	return nil
}
