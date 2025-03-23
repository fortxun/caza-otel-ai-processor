package main

import (
	"fmt"
	"log"
	"os"

	aiprocessor "github.com/fortxun/caza-otel-ai-processor/pkg/processor"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

func main() {
	info := component.BuildInfo{
		Command:     "caza-otel-ai-processor",
		Description: "OpenTelemetry AI Processor",
		Version:     "0.1.0",
	}

	// Explicitly add the config file path to argv if not provided
	args := os.Args
	if len(args) == 1 || (len(args) > 1 && args[1] != "--config") {
		configFile := "config/config.yaml"
		// Use the default config file
		args = append([]string{args[0], "--config=" + configFile}, args[1:]...)
		os.Args = args
	}

	// Set up the collector settings
	settings := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: func() (otelcol.Factories, error) {
			return components()
		},
        // Use the file provider for configuration
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
				DefaultScheme: "file",
			},
		},
	}

	// Create a new collector command
	cmd := otelcol.NewCommand(settings)

	// Execute the command
	if err := cmd.Execute(); err != nil {
		log.Fatalf("collector failed: %v", err)
	}
}

func components() (otelcol.Factories, error) {
	factories := otelcol.Factories{}

	// Register receivers
	receivers, err := otelcol.MakeFactoryMap(
		otlpreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create receiver factories: %w", err)
	}
	factories.Receivers = receivers

	// Register processors
	processors, err := otelcol.MakeFactoryMap(
		aiprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create processor factories: %w", err)
	}
	factories.Processors = processors

	// Register exporters
	exporters, err := otelcol.MakeFactoryMap(
		otlpexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create exporter factories: %w", err)
	}
	factories.Exporters = exporters

	return factories, nil
}