package simpleprocessor

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	ec2provider "poc/internal/ec2metadata"
)

type SimpleProcessor struct {
	logger           *zap.Logger
	cancelFunc       context.CancelFunc
	metadataProvider ec2provider.Provider
}

func newSimpleProcessor(config *Config, logger *zap.Logger) *SimpleProcessor {

	sess, err := session.NewSession()
	if err != nil {
		return nil
	}

	_, cancel := context.WithCancel(context.Background())
	p := &SimpleProcessor{
		logger:           logger,
		cancelFunc:       cancel,
		metadataProvider: ec2provider.NewProvider(sess),
	}
	return p
}

func (simple *SimpleProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		res := rm.At(i).Resource()
		attr := res.Attributes()

		meta, err := simple.metadataProvider.Get(ctx)
		if err != nil {
			fmt.Printf("Failed getting identity document: %v\n", err)
			return md, err
		}

		attr.InsertString(mdKeyInstanceId, meta.InstanceID)
		attr.InsertString(mdKeyImageId, meta.ImageID)
		attr.InsertString(mdKeyInstanceType, meta.InstanceType)
	}
	return md, nil
}

func (simple *SimpleProcessor) shutdown(context.Context) error {
	simple.cancelFunc()
	return nil
}
