package tagger

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"poc/processor/tagger/internal"
)

type resourceDetectionProcessor struct {
	provider           *internal.ResourceProvider
	resource           pcommon.Resource
	schemaURL          string
	httpClientSettings confighttp.HTTPClientSettings
	telemetrySettings  component.TelemetrySettings
}

// Start is invoked during service startup.
func (rdp *resourceDetectionProcessor) Start(ctx context.Context, host component.Host) error {
	client, _ := rdp.httpClientSettings.ToClient(host, rdp.telemetrySettings)
	ctx = internal.ContextWithClient(ctx, client)
	var err error
	rdp.resource, rdp.schemaURL, err = rdp.provider.Get(ctx, client)
	return err
}

// processMetrics implements the ProcessMetricsFunc type.
func (rdp *resourceDetectionProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	resourceMetricsSlice := md.ResourceMetrics()
	for i := 0; i < resourceMetricsSlice.Len(); i++ {
		rm := resourceMetricsSlice.At(i)
		rm.SetSchemaUrl(internal.MergeSchemaURL(rm.SchemaUrl(), rdp.schemaURL))
		res := rm.Resource()
		internal.MergeResource(res, rdp.resource)
	}
	return md, nil
}
