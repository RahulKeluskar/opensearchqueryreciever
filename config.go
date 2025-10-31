package opensearchqueryreceiver

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
)

// Config defines the configuration for the OpenSearch Query Receiver.
// It supports two operational modes:
//
// 1. Direct Mode: Connects directly to OpenSearch with optional authentication
//   - Use this for testing and development environments
//   - Requires direct network access to OpenSearch
//
// 2. Proxy Mode: Connects through an OAuth2 authentication proxy
//   - Use this for production environments requiring OAuth2
//   - The proxy handles token management and renewal
type Config struct {
	// ClientConfig configures the HTTP client (timeouts, TLS, headers, etc.)
	confighttp.ClientConfig `mapstructure:",squash"`

	// CollectionInterval defines how often to collect metrics
	CollectionInterval time.Duration `mapstructure:"collection_interval"`

	// InitialDelay is the delay before first collection
	InitialDelay time.Duration `mapstructure:"initial_delay"`

	// Mode specifies the operational mode: "direct" or "proxy"
	// - "direct": Connect directly to OpenSearch (uses Endpoint, Username, Password)
	// - "proxy": Connect through OAuth2 proxy (uses ProxyEndpoint, requires proxy setup)
	Mode string `mapstructure:"mode"`

	// Username for basic authentication (direct mode only)
	Username string `mapstructure:"username"`

	// Password for basic authentication (direct mode only)
	Password string `mapstructure:"password"`

	// ProxyEndpoint is the URL of the OAuth2 authentication proxy (proxy mode only)
	// Example: http://localhost:8080
	ProxyEndpoint string `mapstructure:"proxy_endpoint"`

	// Queries is a list of OpenSearch queries to execute
	// Each query should have a name and a query body
	Queries []QueryConfig `mapstructure:"queries"`

	// IndexPattern is the OpenSearch index pattern to query
	// Example: "logs-*" or "metrics-2024.01.*"
	IndexPattern string `mapstructure:"index_pattern"`

	// TimeField is the field name used for time-based queries
	// Default: "@timestamp"
	TimeField string `mapstructure:"time_field"`

	// LookbackPeriod defines how far back to query for data
	// Default: 5m (5 minutes)
	LookbackPeriod time.Duration `mapstructure:"lookback_period"`
}

// QueryConfig defines a single query to execute against OpenSearch
type QueryConfig struct {
	// Name is a unique identifier for this query
	Name string `mapstructure:"name"`

	// Description provides context about what this query does
	Description string `mapstructure:"description"`

	// Query is the OpenSearch query DSL in JSON format
	// This will be sent as the request body to the _search endpoint
	Query map[string]interface{} `mapstructure:"query"`

	// MetricName is the name to use for the resulting metric
	// If empty, defaults to "opensearch.query.{name}"
	MetricName string `mapstructure:"metric_name"`

	// Labels are additional key-value pairs to attach to the metric
	Labels map[string]string `mapstructure:"labels"`
}

// Validate checks if the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Mode == "" {
		return errors.New("mode must be specified (direct or proxy)")
	}

	if cfg.Mode != "direct" && cfg.Mode != "proxy" {
		return fmt.Errorf("invalid mode '%s': must be 'direct' or 'proxy'", cfg.Mode)
	}

	// Validate direct mode configuration
	if cfg.Mode == "direct" {
		if cfg.Endpoint == "" {
			return errors.New("endpoint must be specified in direct mode")
		}
		// Username and password are optional in direct mode (for unsecured instances)
	}

	// Validate proxy mode configuration
	if cfg.Mode == "proxy" {
		if cfg.ProxyEndpoint == "" {
			return errors.New("proxy_endpoint must be specified in proxy mode")
		}
	}

	// Validate queries
	if len(cfg.Queries) == 0 {
		return errors.New("at least one query must be configured")
	}

	for i, query := range cfg.Queries {
		if query.Name == "" {
			return fmt.Errorf("query[%d]: name must be specified", i)
		}
		if query.Query == nil || len(query.Query) == 0 {
			return fmt.Errorf("query[%d] (%s): query body must be specified", i, query.Name)
		}
	}

	// Validate index pattern
	if cfg.IndexPattern == "" {
		return errors.New("index_pattern must be specified")
	}

	// Set defaults
	if cfg.TimeField == "" {
		cfg.TimeField = "@timestamp"
	}

	if cfg.LookbackPeriod == 0 {
		cfg.LookbackPeriod = 5 * time.Minute
	}

	return nil
}

// GetEndpoint returns the appropriate endpoint based on the mode
func (cfg *Config) GetEndpoint() string {
	if cfg.Mode == "proxy" {
		return cfg.ProxyEndpoint
	}
	return cfg.Endpoint
}

// UsesBasicAuth returns true if basic authentication should be used
func (cfg *Config) UsesBasicAuth() bool {
	return cfg.Mode == "direct" && cfg.Username != "" && cfg.Password != ""
}
