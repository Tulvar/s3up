package cli

import (
	"context"
	"flag"
	"fmt"

	appconfig "github.com/tulvar/s3up/internal/config"
	deletepkg "github.com/tulvar/s3up/internal/delete"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/s3client"
)

type deleteFlags struct {
	profile               string
	region                string
	endpointURL           string
	allowInsecureEndpoint bool
	addressingStyle       string
	pathStyle             bool
	recursive             bool
	dryRun                bool
	yes                   bool
	noProgress            bool
	workers               int
	include               stringValues
	exclude               stringValues
}

func registerDeleteFlags(fs *flag.FlagSet, values *deleteFlags) {
	if values == nil {
		values = &deleteFlags{}
	}
	fs.StringVar(&values.profile, "profile", "", "AWS shared config profile")
	fs.StringVar(&values.region, "region", "", "AWS region")
	fs.StringVar(&values.endpointURL, "endpoint-url", "", "custom S3 endpoint URL")
	fs.BoolVar(&values.allowInsecureEndpoint, "allow-insecure-endpoint", false, "allow http endpoint URLs")
	fs.StringVar(&values.addressingStyle, "addressing-style", "", "S3 addressing style: auto, path, or virtual")
	fs.BoolVar(&values.pathStyle, "path-style", false, "use path-style addressing")
	fs.BoolVar(&values.recursive, "recursive", false, "delete objects recursively under the target prefix")
	fs.BoolVar(&values.dryRun, "dry-run", false, "print delete actions without deleting objects")
	fs.BoolVar(&values.yes, "yes", false, "confirm recursive delete")
	fs.BoolVar(&values.noProgress, "no-progress", false, "disable delete progress output")
	fs.IntVar(&values.workers, "workers", 1, "number of delete actions to process in parallel")
	fs.Var(&values.include, "include", "include files by glob pattern; can be repeated")
	fs.Var(&values.exclude, "exclude", "exclude files by glob pattern; can be repeated")
}

func (c CLI) runDelete(ctx context.Context, args []string) error {
	values := &deleteFlags{}
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	registerDeleteFlags(fs, values)
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: s3up delete [flags] <s3://bucket/key-or-prefix>")
	}

	target, err := list.ParseS3Prefix(fs.Arg(0))
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
	}

	var lister *s3client.Lister
	if values.recursive {
		lister, err = s3client.NewLister(ctx, cfg)
		if err != nil {
			return err
		}
	}

	var deleter *s3client.Deleter
	if !values.dryRun {
		deleter, err = s3client.NewDeleter(ctx, cfg)
		if err != nil {
			return err
		}
	}

	return deletepkg.Service{Lister: lister, Deleter: deleter, Stdout: c.stdout}.Delete(ctx, deletepkg.Request{
		Target:        target,
		Recursive:     values.recursive,
		DryRun:        values.dryRun,
		ConfirmDelete: values.yes,
		Include:       values.include.Values(),
		Exclude:       values.exclude.Values(),
		Progress:      !values.noProgress,
		Workers:       values.workers,
	})
}
