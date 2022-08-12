package tagger

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"poc/processor/tagger/internal"
	"poc/processor/tagger/internal/ec2"
)

const (
	// The value of "type" key in configuration.
	typeStr = "tagger"
	// The stability level of the processor.
	stability = component.StabilityLevelBeta
)

var consumerCapabilities = consumer.Capabilities{MutatesData: true}

type factory struct {
	resourceProviderFactory *internal.ResourceProviderFactory

	// providers stores a provider for each named processor that
	// may a different set of detectors configured.
	providers map[config.ComponentID]*internal.ResourceProvider
	lock      sync.Mutex
}

// NewFactory creates a new factory for ResourceDetection processor.
func NewFactory() component.ProcessorFactory {
	resourceProviderFactory := internal.NewProviderFactory(map[internal.DetectorType]internal.DetectorFactory{
		ec2.TypeStr: ec2.NewDetector,
	})

	f := &factory{
		resourceProviderFactory: resourceProviderFactory,
		providers:               map[config.ComponentID]*internal.ResourceProvider{},
	}

	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsProcessor(f.createMetricsProcessor, stability))
}

// Type gets the type of the Option config created by this factory.
func (*factory) Type() config.Type {
	return typeStr
}

func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings:  config.NewProcessorSettings(config.NewComponentID(typeStr)),
		Detectors:          []string{},
		HTTPClientSettings: defaultHTTPClientSettings(),
	}
}

func defaultHTTPClientSettings() confighttp.HTTPClientSettings {
	httpClientSettings := confighttp.NewDefaultHTTPClientSettings()
	httpClientSettings.Timeout = 5 * time.Second
	return httpClientSettings
}

func (f *factory) createMetricsProcessor(
	_ context.Context,
	params component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Metrics,
) (component.MetricsProcessor, error) {
	rdp, err := f.getResourceDetectionProcessor(params, cfg)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewMetricsProcessor(
		cfg,
		nextConsumer,
		rdp.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(rdp.Start))
}

func (f *factory) getResourceDetectionProcessor(
	params component.ProcessorCreateSettings,
	cfg config.Processor,
) (*resourceDetectionProcessor, error) {
	oCfg := cfg.(*Config)

	provider, err := f.getResourceProvider(params, cfg.ID(), oCfg.HTTPClientSettings.Timeout, oCfg.Detectors)
	if err != nil {
		return nil, err
	}

	return &resourceDetectionProcessor{
		provider:           provider,
		httpClientSettings: oCfg.HTTPClientSettings,
		telemetrySettings:  params.TelemetrySettings,
	}, nil
}

func (f *factory) getResourceProvider(
	params component.ProcessorCreateSettings,
	processorName config.ComponentID,
	timeout time.Duration,
	configuredDetectors []string,
) (*internal.ResourceProvider, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if provider, ok := f.providers[processorName]; ok {
		return provider, nil
	}

	detectorTypes := make([]internal.DetectorType, 0, len(configuredDetectors))
	for _, key := range configuredDetectors {
		detectorTypes = append(detectorTypes, internal.DetectorType(strings.TrimSpace(key)))
	}

	provider, err := f.resourceProviderFactory.CreateResourceProvider(params, timeout, detectorTypes...)
	if err != nil {
		return nil, err
	}

	f.providers[processorName] = provider
	return provider, nil
}
