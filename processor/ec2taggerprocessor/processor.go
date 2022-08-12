package ec2taggerprocessor

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type EC2TaggerProcessor struct {
	logger     *zap.Logger
	cancelFunc context.CancelFunc
	metadata   *ec2metadata.EC2Metadata
}

func newEC2TaggerProcessor(config *Config, logger *zap.Logger) *EC2TaggerProcessor {

	sess, err := session.NewSession()
	if err != nil {
		return nil
	}

	_, cancel := context.WithCancel(context.Background())
	p := &EC2TaggerProcessor{
		logger:     logger,
		cancelFunc: cancel,
		metadata:   ec2metadata.New(sess),
	}
	return p
}

func (ec2tagger *EC2TaggerProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	var err error
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		res := rm.At(i).Resource()
		attr := res.Attributes()

		if _, err = ec2tagger.metadata.GetMetadataWithContext(ctx, "instance-id"); err != nil {
			fmt.Printf("EC2 metadata unavailable %v\n", err)
			return md, err
		}

		meta, err := ec2tagger.metadata.GetInstanceIdentityDocumentWithContext(ctx)
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

func (ctdp *EC2TaggerProcessor) shutdown(context.Context) error {
	ctdp.cancelFunc()
	return nil
}
