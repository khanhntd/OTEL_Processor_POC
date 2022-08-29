package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsemfexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/service"
	"go.uber.org/multierr"
	"log"
	"path/filepath"
	"poc/processor/simpleprocessor"
	"poc/processor/taggerprocessor"
)

func main() {
	factories, err := Components()
	if err != nil {
		log.Fatalf("Failed to build factories : %v\n", err)
	}

	info := component.BuildInfo{
		Command:     "OTEL_Processor_POC",
		Description: "OTEL Migration for Processor",
		Version:     "1.0",
	}

	cfgProvider, err := service.NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("config", "config.yaml")}))
	if err != nil {
		log.Fatalf("Failed to build Config Provider : %v\n", err)
	}

	params := service.CollectorSettings{
		Factories:      factories,
		BuildInfo:      info,
		ConfigProvider: cfgProvider,
	}

	if err = service.NewCommand(params).Execute(); err != nil {
		log.Fatalf("Error starting OTEL Processor: %v\n", err)
	}
}

func Components() (component.Factories, error) {
	var errs error

	factories := component.Factories{}

	receivers, err := component.MakeReceiverFactoryMap(
		hostmetricsreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	processors, err := component.MakeProcessorFactoryMap(
		simpleprocessor.NewFactory(),
		taggerprocessor.NewFactory(),
		batchprocessor.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	exporters, err := component.MakeExporterFactoryMap(
		loggingexporter.NewFactory(),
		awsemfexporter.NewFactory(),
		awscloudwatchlogsexporter.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	factories = component.Factories{
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, errs
}

func newDefaultConfigProviderSettings(locations []string) service.ConfigProviderSettings {
	return service.ConfigProviderSettings{
		Locations:     locations,
		MapProviders:  makeMapProvidersMap(fileprovider.New(), envprovider.New(), yamlprovider.New()),
		MapConverters: []confmap.Converter{expandconverter.New()},
	}
}

func makeMapProvidersMap(providers ...confmap.Provider) map[string]confmap.Provider {
	ret := make(map[string]confmap.Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
