package s3client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tulvar/s3up/internal/config"
)

func NewUploader(ctx context.Context, cfg config.Config) (*Uploader, error) {
	client, err := newClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

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

func NewLister(ctx context.Context, cfg config.Config) (*Lister, error) {
	client, err := newClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Lister{client: client}, nil
}

func NewDeleter(ctx context.Context, cfg config.Config) (*Deleter, error) {
	client, err := newClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Deleter{client: client}, nil
}

func newClient(ctx context.Context, cfg config.Config) (*s3.Client, error) {
	if err := validateEndpoint(cfg.EndpointURL, cfg.AllowInsecureEndpoint); err != nil {
		return nil, err
	}
	usePathStyle, err := usePathStyle(cfg)
	if err != nil {
		return nil, err
	}

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
	awsCfg.Region = defaultRegion(awsCfg.Region)

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
		}
		o.UsePathStyle = usePathStyle
	})

	return client, nil
}

func usePathStyle(cfg config.Config) (bool, error) {
	switch cfg.AddressingStyle {
	case "", "auto":
		if cfg.PathStyle {
			return true, nil
		}
		return cfg.EndpointURL != "", nil
	case "path":
		return true, nil
	case "virtual":
		if cfg.PathStyle {
			return false, fmt.Errorf("--path-style cannot be combined with --addressing-style=virtual")
		}
		return false, nil
	default:
		return false, fmt.Errorf("addressing style must be one of auto, path, or virtual")
	}
}

func defaultRegion(region string) string {
	if region != "" {
		return region
	}
	return "us-east-1"
}

func validateEndpoint(endpoint string, allowInsecure bool) error {
	if endpoint == "" {
		return nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("parse endpoint url: %w", err)
	}
	if parsed.Scheme == "http" && !allowInsecure {
		return fmt.Errorf("insecure endpoint %q requires --allow-insecure-endpoint", endpoint)
	}
	return nil
}
