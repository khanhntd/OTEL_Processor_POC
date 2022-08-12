package ec2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"

	ec2provider "poc/internal/ec2metadata"
	"poc/processor/taggerprocessor/internal"
)

const (
	TypeStr                = "ec2metadata"
	MetadataKeyInstanceId  = "InstanceId"
	MetadataKeyInstaceType = "InstanceType"
	MetadataKeyImageId     = "ImageId"
)

var _ internal.Detector = (*Detector)(nil)

type Detector struct {
	metadataProvider ec2provider.Provider
	logger           *zap.Logger
}

func NewDetector(set component.ProcessorCreateSettings) (internal.Detector, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return &Detector{
		metadataProvider: ec2provider.NewProvider(sess),
		logger:           set.Logger,
	}, nil
}

func (d *Detector) Detect(ctx context.Context) (resource pcommon.Resource, schemaURL string, err error) {
	res := pcommon.NewResource()
	if _, err = d.metadataProvider.InstanceID(ctx); err != nil {
		d.logger.Debug("EC2 metadata unavailable", zap.Error(err))
		return res, "", nil
	}

	meta, err := d.metadataProvider.Get(ctx)
	if err != nil {
		return res, "", fmt.Errorf("failed getting identity document: %w", err)
	}

	attr := res.Attributes()
	attr.InsertString(MetadataKeyInstanceId, meta.InstanceID)
	attr.InsertString(MetadataKeyImageId, meta.ImageID)
	attr.InsertString(MetadataKeyInstaceType, meta.InstanceType)

	client := getHTTPClientSettings(ctx, d.logger)
	tagsAndVolumes, err := connectAndFetchEc2TagsandEcsVolume(meta.Region, meta.InstanceID, client)

	if err != nil {
		return res, "", fmt.Errorf("failed fetching ec2 instance tags: %w", err)
	}
	for key, val := range tagsAndVolumes {
		attr.InsertString(key, val)
	}

	return res, conventions.SchemaURL, nil
}

func getHTTPClientSettings(ctx context.Context, logger *zap.Logger) *http.Client {
	client, err := internal.ClientFromContext(ctx)
	if err != nil {
		client = http.DefaultClient
		logger.Debug("Error retrieving client from context thus creating default", zap.Error(err))
	}
	return client
}

func connectAndFetchEc2TagsandEcsVolume(region string, instanceID string, client *http.Client) (map[string]string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:     aws.String(region),
		HTTPClient: client},
	)
	if err != nil {
		return nil, err
	}
	e := ec2.New(sess)

	return fetchEC2TagsAndVolumes(e, instanceID)
}

func fetchEC2TagsAndVolumes(svc ec2iface.EC2API, instanceID string) (map[string]string, error) {
	ec2Tags, err := svc.DescribeTags(&ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{{
			Name:   aws.String("resource-id"),
			Values: aws.StringSlice([]string{instanceID}),
		}},
	})
	if err != nil {
		return nil, err
	}
	tagsAndVolumes := make(map[string]string)
	for _, tag := range ec2Tags.Tags {
		tagsAndVolumes[*tag.Key] = *tag.Value
	}

	ec2Volumes, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.instance-id"),
				Values: aws.StringSlice([]string{instanceID}),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for _, volume := range ec2Volumes.Volumes {
		for _, attachment := range volume.Attachments {
			tagsAndVolumes[*attachment.VolumeId] = fmt.Sprintf("aws://%s/%s", *volume.AvailabilityZone, *attachment.VolumeId)
		}
	}

	return tagsAndVolumes, nil
}
