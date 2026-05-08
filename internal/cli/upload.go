package cli

import (
	"context"
	"flag"
	"fmt"

	appconfig "github.com/tulvar/s3up/internal/config"
	"github.com/tulvar/s3up/internal/s3client"
	"github.com/tulvar/s3up/internal/upload"
)

type uploadFlags struct {
	profile     string
	region      string
	endpointURL string
	pathStyle   bool
	recursive   bool
	dryRun      bool
}

func registerUploadFlags(fs *flag.FlagSet, values *uploadFlags) {
	if values == nil {
		values = &uploadFlags{}
	}
	fs.StringVar(&values.profile, "profile", "", "AWS shared config profile")
	fs.StringVar(&values.region, "region", "", "AWS region")
	fs.StringVar(&values.endpointURL, "endpoint-url", "", "custom S3 endpoint URL")
	fs.BoolVar(&values.pathStyle, "path-style", false, "use path-style addressing")
	fs.BoolVar(&values.recursive, "recursive", false, "upload directories recursively")
	fs.BoolVar(&values.dryRun, "dry-run", false, "print planned uploads without sending objects")
}

func (c CLI) runUpload(ctx context.Context, args []string) error {
	values := &uploadFlags{}
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	registerUploadFlags(fs, values)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 2 {
		return fmt.Errorf("usage: s3up upload [flags] <local-path> <s3://bucket/key>")
	}

	dest, err := upload.ParseS3URI(fs.Arg(1))
	if err != nil {
		return err
	}

	req := upload.Request{
		Source:      fs.Arg(0),
		Destination: dest,
		Recursive:   values.recursive,
		DryRun:      values.dryRun,
	}

	var uploader upload.ObjectUploader
	if !values.dryRun {
		uploader, err = s3client.NewUploader(ctx, appconfig.Config{
			Profile:     values.profile,
			Region:      values.region,
			EndpointURL: values.endpointURL,
			PathStyle:   values.pathStyle,
		})
		if err != nil {
			return err
		}
	}

	return upload.Service{Uploader: uploader, Stdout: c.stdout}.Upload(ctx, req)
}
