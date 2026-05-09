package cli

import (
	"context"
	"flag"
	"fmt"

	appconfig "github.com/tulvar/s3up/internal/config"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/s3client"
)

type lsFlags struct {
	profile               string
	region                string
	endpointURL           string
	allowInsecureEndpoint bool
	pathStyle             bool
	recursive             bool
	human                 bool
	json                  bool
}

func registerLSFlags(fs *flag.FlagSet, values *lsFlags) {
	if values == nil {
		values = &lsFlags{}
	}
	fs.StringVar(&values.profile, "profile", "", "AWS shared config profile")
	fs.StringVar(&values.region, "region", "", "AWS region")
	fs.StringVar(&values.endpointURL, "endpoint-url", "", "custom S3 endpoint URL")
	fs.BoolVar(&values.allowInsecureEndpoint, "allow-insecure-endpoint", false, "allow http endpoint URLs")
	fs.BoolVar(&values.pathStyle, "path-style", false, "use path-style addressing")
	fs.BoolVar(&values.recursive, "recursive", false, "list objects recursively")
	fs.BoolVar(&values.human, "human", false, "print object sizes in human-readable units")
	fs.BoolVar(&values.json, "json", false, "print entries as JSON")
}

func (c CLI) runLS(ctx context.Context, args []string) error {
	values := &lsFlags{}
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	registerLSFlags(fs, values)
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: s3up ls [flags] <s3://bucket/prefix>")
	}
	if values.human && values.json {
		return fmt.Errorf("--human and --json cannot be used together")
	}

	target, err := list.ParseS3Prefix(fs.Arg(0))
	if err != nil {
		return err
	}

	lister, err := s3client.NewLister(ctx, appconfig.Config{
		Profile:               values.profile,
		Region:                values.region,
		EndpointURL:           values.endpointURL,
		AllowInsecureEndpoint: values.allowInsecureEndpoint,
		PathStyle:             values.pathStyle,
	})
	if err != nil {
		return err
	}

	return list.Service{Lister: lister, Stdout: c.stdout}.List(ctx, list.Request{
		Target:    target,
		Recursive: values.recursive,
		Human:     values.human,
		JSON:      values.json,
	})
}
