// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resourcedetectionprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"

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

	"poc/processor/resourcedetectionprocessor/internal"
	"poc/processor/resourcedetectionprocessor/internal/aws/ec2"
)

const (
	// The value of "type" key in configuration.
	typeStr = "resourcedetection"
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
		ec2.TypeStr:              ec2.NewDetector,
	})

	f := &factory{
		resourceProviderFactory: resourceProviderFactory,
		providers:               map[config.ComponentID]*internal.ResourceProvider{},
	}

	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsProcessor(f.createMetricsProcessor, stability)
}

// Type gets the type of the Option config created by this factory.
func (*factory) Type() config.Type {
	return typeStr
}

func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings:  config.NewProcessorSettings(config.NewComponentID(typeStr)),
		HTTPClientSettings: defaultHTTPClientSettings()
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

	provider, err := f.getResourceProvider(params, cfg.ID(), oCfg.HTTPClientSettings.Timeout)
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
) (*internal.ResourceProvider, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if provider, ok := f.providers[processorName]; ok {
		return provider, nil
	}

	provider, err := f.resourceProviderFactory.CreateResourceProvider(params, timeout)
	if err != nil {
		return nil, err
	}

	f.providers[processorName] = provider
	return provider, nil
}
