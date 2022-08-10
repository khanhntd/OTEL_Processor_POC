package ec2tagger

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type EC2TaggerProcessor struct {
	logger     *zap.Logger
	cancelFunc context.CancelFunc
}

func newEC2TaggerProcessor(config *Config, logger *zap.Logger) *EC2TaggerProcessor {
	_, cancel := context.WithCancel(context.Background())
	p := &EC2TaggerProcessor{
		logger:                  logger,
		cancelFunc:              cancel,
	}
	return p
}

func (ec2tagger *EC2TaggerProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		res := rm.At(i).Resource()
	}
	return md, nil
}

func (ctdp *EC2TaggerProcessor) shutdown(context.Context) error {
	ctdp.cancelFunc()
	return nil
}