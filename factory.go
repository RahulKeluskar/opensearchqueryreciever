package opensearchqueryreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// Type identifier for the OpenSearch Query Receiver
var typeStr = component.MustNewType("opensearchquery")

// NewFactory creates a new factory for the OpenSearch Query Receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the receiver
func createDefaultConfig() component.Config {
	return &Config{
		ClientConfig: confighttp.ClientConfig{
			Timeout: 30 * time.Second,
		},
		CollectionInterval: 60 * time.Second,
		InitialDelay:       time.Second,
		Mode:               "direct",
		TimeField:          "@timestamp",
		LookbackPeriod:     5 * time.Minute,
		Queries:            []QueryConfig{},
	}
}

// createMetricsReceiver creates a metrics receiver based on the provided config
func createMetricsReceiver(
	ctx context.Context,
	settings receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	rCfg := cfg.(*Config)

	// Create and return the metrics receiver
	return newMetricsReceiver(rCfg, consumer, settings)
}
