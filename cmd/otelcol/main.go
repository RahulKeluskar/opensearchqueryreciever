package main

import (
	"fmt"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver"

	opensearchqueryreceiver "github.com/opensearchqueryreceiver"
)

func main() {
	// Create collector settings
	info := component.BuildInfo{
		Command:     "otelcol-opensearchquery",
		Description: "OpenTelemetry Collector with OpenSearch Query Receiver",
		Version:     "0.1.0",
	}

	// Create and run collector
	cmd := otelcol.NewCommand(otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{getConfigFilePath()},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
				DefaultScheme: "file",
			},
		},
	})

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "collector server run finished with error: %v\n", err)
		os.Exit(1)
	}
}

// components builds the set of components for the collector
func components() (otelcol.Factories, error) {
	factories := otelcol.Factories{}

	// Build receiver factories map
	receiverFactory := opensearchqueryreceiver.NewFactory()
	factories.Receivers = map[component.Type]receiver.Factory{
		receiverFactory.Type(): receiverFactory,
	}

	// Build processor factories map
	batchProcessorFactory := batchprocessor.NewFactory()
	factories.Processors = map[component.Type]processor.Factory{
		batchProcessorFactory.Type(): batchProcessorFactory,
	}

	// Build exporter factories map
	debugExporterFactory := debugexporter.NewFactory()
	factories.Exporters = map[component.Type]exporter.Factory{
		debugExporterFactory.Type(): debugExporterFactory,
	}

	// Initialize empty maps for other component types
	factories.Extensions = make(map[component.Type]extension.Factory)
	factories.Connectors = make(map[component.Type]connector.Factory)

	return factories, nil
}

// getConfigFilePath returns the config file path from environment or default
func getConfigFilePath() string {
	configPath := os.Getenv("OTEL_CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yaml"
	}
	return configPath
}
