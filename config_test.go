package opensearchqueryreceiver

import (
	"testing"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid direct mode config",
			config: &Config{
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
				LookbackPeriod: 5 * time.Minute,
				Queries: []QueryConfig{
					{
						Name: "test_query",
						Query: map[string]interface{}{
							"match_all": map[string]interface{}{},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid proxy mode config",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Timeout: 30 * time.Second,
				},
				CollectionInterval: 60 * time.Second,
				Mode:           "proxy",
				ProxyEndpoint:  "http://localhost:8080",
				IndexPattern:   "logs-*",
				TimeField:      "@timestamp",
				LookbackPeriod: 5 * time.Minute,
				Queries: []QueryConfig{
					{
						Name: "test_query",
						Query: map[string]interface{}{
							"match_all": map[string]interface{}{},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing mode",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://localhost:9200",
				},
				IndexPattern: "logs-*",
				Queries: []QueryConfig{
					{
						Name:  "test",
						Query: map[string]interface{}{"match_all": map[string]interface{}{}},
					},
				},
			},
			wantErr: true,
			errMsg:  "mode must be specified",
		},
		{
			name: "invalid mode",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://localhost:9200",
				},
				Mode:         "invalid",
				IndexPattern: "logs-*",
				Queries: []QueryConfig{
					{
						Name:  "test",
						Query: map[string]interface{}{"match_all": map[string]interface{}{}},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid mode",
		},
		{
			name: "direct mode missing endpoint",
			config: &Config{
				Mode:         "direct",
				IndexPattern: "logs-*",
				Queries: []QueryConfig{
					{
						Name:  "test",
						Query: map[string]interface{}{"match_all": map[string]interface{}{}},
					},
				},
			},
			wantErr: true,
			errMsg:  "endpoint must be specified in direct mode",
		},
		{
			name: "proxy mode missing proxy_endpoint",
			config: &Config{
				Mode:         "proxy",
				IndexPattern: "logs-*",
				Queries: []QueryConfig{
					{
						Name:  "test",
						Query: map[string]interface{}{"match_all": map[string]interface{}{}},
					},
				},
			},
			wantErr: true,
			errMsg:  "proxy_endpoint must be specified in proxy mode",
		},
		{
			name: "missing queries",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://localhost:9200",
				},
				Mode:         "direct",
				IndexPattern: "logs-*",
				Queries:      []QueryConfig{},
			},
			wantErr: true,
			errMsg:  "at least one query must be configured",
		},
		{
			name: "query missing name",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://localhost:9200",
				},
				Mode:         "direct",
				IndexPattern: "logs-*",
				Queries: []QueryConfig{
					{
						Query: map[string]interface{}{"match_all": map[string]interface{}{}},
					},
				},
			},
			wantErr: true,
			errMsg:  "name must be specified",
		},
		{
			name: "query missing body",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://localhost:9200",
				},
				Mode:         "direct",
				IndexPattern: "logs-*",
				Queries: []QueryConfig{
					{
						Name: "test",
					},
				},
			},
			wantErr: true,
			errMsg:  "query body must be specified",
		},
		{
			name: "missing index_pattern",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://localhost:9200",
				},
				Mode: "direct",
				Queries: []QueryConfig{
					{
						Name:  "test",
						Query: map[string]interface{}{"match_all": map[string]interface{}{}},
					},
				},
			},
			wantErr: true,
			errMsg:  "index_pattern must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, should contain '%s'", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "https://localhost:9200",
		},
		Mode:         "direct",
		IndexPattern: "logs-*",
		Queries: []QueryConfig{
			{
				Name:  "test",
				Query: map[string]interface{}{"match_all": map[string]interface{}{}},
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Check defaults
	if config.TimeField != "@timestamp" {
		t.Errorf("Expected default time_field '@timestamp', got '%s'", config.TimeField)
	}

	if config.LookbackPeriod != 5*time.Minute {
		t.Errorf("Expected default lookback_period 5m, got %v", config.LookbackPeriod)
	}
}

func TestConfigGetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "direct mode",
			config: &Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "https://opensearch.example.com:9200",
				},
				Mode: "direct",
			},
			expected: "https://opensearch.example.com:9200",
		},
		{
			name: "proxy mode",
			config: &Config{
				Mode:          "proxy",
				ProxyEndpoint: "http://proxy.example.com:8080",
			},
			expected: "http://proxy.example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := tt.config.GetEndpoint()
			if endpoint != tt.expected {
				t.Errorf("GetEndpoint() = %v, want %v", endpoint, tt.expected)
			}
		})
	}
}

func TestConfigUsesBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "direct mode with auth",
			config: &Config{
				Mode:     "direct",
				Username: "admin",
				Password: "admin",
			},
			expected: true,
		},
		{
			name: "direct mode without auth",
			config: &Config{
				Mode: "direct",
			},
			expected: false,
		},
		{
			name: "proxy mode with credentials",
			config: &Config{
				Mode:     "proxy",
				Username: "admin",
				Password: "admin",
			},
			expected: false,
		},
		{
			name: "direct mode missing password",
			config: &Config{
				Mode:     "direct",
				Username: "admin",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usesAuth := tt.config.UsesBasicAuth()
			if usesAuth != tt.expected {
				t.Errorf("UsesBasicAuth() = %v, want %v", usesAuth, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
