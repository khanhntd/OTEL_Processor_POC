package ec2tagger

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	typeStr   = "simple"
	stability = component.StabilityLevelStable
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

type Config struct {
	config.ProcessorSettings
}

func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewComponentID(typeStr)),
	}
}

func NewFactory() component.ProcessorFactory {
	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsProcessor(createMetricsProcessor, stability))
}

func createMetricsProcessor(
	_ context.Context,
	params component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Metrics,
) (component.MetricsProcessor, error) {
	processorConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("configuration parsing error")
	}

	metricsProcessor := newEC2TaggerProcessor(processorConfig, params.Logger)

	return processorhelper.NewMetricsProcessor(
		cfg,
		nextConsumer,
		metricsProcessor.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(metricsProcessor.shutdown))
}
