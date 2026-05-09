package cli

import (
	"context"
	"flag"
	"fmt"

	appconfig "github.com/tulvar/s3up/internal/config"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/s3client"
	syncer "github.com/tulvar/s3up/internal/sync"
	"github.com/tulvar/s3up/internal/upload"
)

type syncFlags struct {
	profile               string
	region                string
	endpointURL           string
	allowInsecureEndpoint bool
	addressingStyle       string
	pathStyle             bool
	followLinks           bool
	dryRun                bool
	checksum              bool
	delete                bool
	yes                   bool
	noProgress            bool
	contentType           string
	storageClass          string
	acl                   string
	concurrency           int
	partSize              byteSize
	workers               int
	metadata              metadataValues
	include               stringValues
	exclude               stringValues
}

func registerSyncFlags(fs *flag.FlagSet, values *syncFlags) {
	if values == nil {
		values = &syncFlags{}
	}
	fs.StringVar(&values.profile, "profile", "", "AWS shared config profile")
	fs.StringVar(&values.region, "region", "", "AWS region")
	fs.StringVar(&values.endpointURL, "endpoint-url", "", "custom S3 endpoint URL")
	fs.BoolVar(&values.allowInsecureEndpoint, "allow-insecure-endpoint", false, "allow http endpoint URLs")
	fs.StringVar(&values.addressingStyle, "addressing-style", "", "S3 addressing style: auto, path, or virtual")
	fs.BoolVar(&values.pathStyle, "path-style", false, "use path-style addressing")
	fs.BoolVar(&values.followLinks, "follow-symlinks", false, "follow symlinked files")
	fs.BoolVar(&values.dryRun, "dry-run", false, "print sync actions without uploading objects")
	fs.BoolVar(&values.checksum, "checksum", false, "compare same-size objects using local MD5 and remote ETag")
	fs.BoolVar(&values.delete, "delete", false, "delete remote objects missing from the local plan")
	fs.BoolVar(&values.yes, "yes", false, "confirm destructive sync actions such as --delete")
	fs.BoolVar(&values.noProgress, "no-progress", false, "disable sync progress output")
	fs.StringVar(&values.contentType, "content-type", "", "content type for uploaded objects")
	fs.StringVar(&values.storageClass, "storage-class", "", "S3 storage class")
	fs.StringVar(&values.acl, "acl", "", "S3 canned ACL")
	fs.IntVar(&values.concurrency, "concurrency", 0, "multipart upload concurrency")
	fs.IntVar(&values.workers, "workers", 1, "number of sync actions to process in parallel")
	fs.Var(&values.partSize, "part-size", "multipart part size, for example 8MiB or 8388608")
	fs.Var(&values.metadata, "metadata", "object metadata entry in key=value form; can be repeated")
	fs.Var(&values.include, "include", "include files by glob pattern; can be repeated")
	fs.Var(&values.exclude, "exclude", "exclude files by glob pattern; can be repeated")
}

func (c CLI) runSync(ctx context.Context, args []string) error {
	values := &syncFlags{}
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	registerSyncFlags(fs, values)
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() != 2 {
		return fmt.Errorf("usage: s3up sync [flags] <local-path> <s3://bucket/prefix>")
	}

	dest, err := list.ParseS3Prefix(fs.Arg(1))
	if err != nil {
		return err
	}

	cfg := appconfig.Config{
		Profile:               values.profile,
		Region:                values.region,
		EndpointURL:           values.endpointURL,
		AllowInsecureEndpoint: values.allowInsecureEndpoint,
		AddressingStyle:       values.addressingStyle,
		PathStyle:             values.pathStyle,
		Concurrency:           values.concurrency,
		PartSize:              int64(values.partSize),
	}

	lister, err := s3client.NewLister(ctx, cfg)
	if err != nil {
		return err
	}

	var uploader upload.ObjectUploader
	var deleter syncer.ObjectDeleter
	if !values.dryRun {
		uploader, err = s3client.NewUploader(ctx, cfg)
		if err != nil {
			return err
		}
		if values.delete {
			deleter, err = s3client.NewDeleter(ctx, cfg)
			if err != nil {
				return err
			}
		}
	}

	return syncer.Service{Lister: lister, Uploader: uploader, Deleter: deleter, Stdout: c.stdout}.Sync(ctx, syncer.Request{
		Source:        fs.Arg(0),
		Destination:   dest,
		DryRun:        values.dryRun,
		FollowLinks:   values.followLinks,
		Checksum:      values.checksum,
		Delete:        values.delete,
		ConfirmDelete: values.yes,
		Include:       values.include.Values(),
		Exclude:       values.exclude.Values(),
		Options: upload.Options{
			ContentType:  values.contentType,
			Metadata:     values.metadata.Map(),
			StorageClass: values.storageClass,
			ACL:          values.acl,
		},
		Progress: !values.noProgress,
		Workers:  values.workers,
	})
}
