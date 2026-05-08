package s3client

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tulvar/s3up/internal/config"
)

func NewUploader(ctx context.Context, cfg config.Config) (*Uploader, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	if cfg.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.Profile))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
		}
		o.UsePathStyle = cfg.PathStyle
	})

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		if cfg.Concurrency > 0 {
			u.Concurrency = cfg.Concurrency
		}
		if cfg.PartSize > 0 {
			u.PartSize = cfg.PartSize
		}
	})

	return &Uploader{uploader: uploader}, nil
}
