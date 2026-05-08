package cli

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	appconfig "github.com/tulvar/s3up/internal/config"
	"github.com/tulvar/s3up/internal/s3client"
	"github.com/tulvar/s3up/internal/upload"
)

type uploadFlags struct {
	profile      string
	region       string
	endpointURL  string
	pathStyle    bool
	recursive    bool
	dryRun       bool
	noProgress   bool
	contentType  string
	storageClass string
	acl          string
	concurrency  int
	partSize     byteSize
	metadata     metadataValues
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
	fs.BoolVar(&values.noProgress, "no-progress", false, "disable upload progress output")
	fs.StringVar(&values.contentType, "content-type", "", "content type for uploaded objects")
	fs.StringVar(&values.storageClass, "storage-class", "", "S3 storage class")
	fs.StringVar(&values.acl, "acl", "", "S3 canned ACL")
	fs.IntVar(&values.concurrency, "concurrency", 0, "multipart upload concurrency")
	fs.Var(&values.partSize, "part-size", "multipart part size, for example 8MiB or 8388608")
	fs.Var(&values.metadata, "metadata", "object metadata entry in key=value form; can be repeated")
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
		Options: upload.Options{
			ContentType:  values.contentType,
			Metadata:     values.metadata.Map(),
			StorageClass: values.storageClass,
			ACL:          values.acl,
		},
		Progress: !values.noProgress,
	}

	var uploader upload.ObjectUploader
	if !values.dryRun {
		uploader, err = s3client.NewUploader(ctx, appconfig.Config{
			Profile:     values.profile,
			Region:      values.region,
			EndpointURL: values.endpointURL,
			PathStyle:   values.pathStyle,
			Concurrency: values.concurrency,
			PartSize:    int64(values.partSize),
		})
		if err != nil {
			return err
		}
	}

	return upload.Service{Uploader: uploader, Stdout: c.stdout}.Upload(ctx, req)
}

type metadataValues map[string]string

func (m *metadataValues) String() string {
	if m == nil || len(*m) == 0 {
		return ""
	}

	parts := make([]string, 0, len(*m))
	for key, value := range *m {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func (m *metadataValues) Set(raw string) error {
	key, value, ok := strings.Cut(raw, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return fmt.Errorf("metadata must use key=value format")
	}
	if *m == nil {
		*m = make(map[string]string)
	}
	(*m)[strings.TrimSpace(key)] = value
	return nil
}

func (m metadataValues) Map() map[string]string {
	if len(m) == 0 {
		return nil
	}

	out := make(map[string]string, len(m))
	for key, value := range m {
		out[key] = value
	}
	return out
}

type byteSize int64

func (s *byteSize) String() string {
	if s == nil || *s == 0 {
		return ""
	}
	return strconv.FormatInt(int64(*s), 10)
}

func (s *byteSize) Set(raw string) error {
	size, err := parseByteSize(raw)
	if err != nil {
		return err
	}
	*s = byteSize(size)
	return nil
}

func parseByteSize(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("part size cannot be empty")
	}

	units := []struct {
		suffix string
		mult   int64
	}{
		{"gib", 1024 * 1024 * 1024},
		{"gb", 1000 * 1000 * 1000},
		{"mib", 1024 * 1024},
		{"mb", 1000 * 1000},
		{"kib", 1024},
		{"kb", 1000},
		{"b", 1},
	}

	lower := strings.ToLower(value)
	for _, unit := range units {
		if strings.HasSuffix(lower, unit.suffix) {
			number := strings.TrimSpace(value[:len(value)-len(unit.suffix)])
			parsed, err := strconv.ParseInt(number, 10, 64)
			if err != nil || parsed <= 0 {
				return 0, fmt.Errorf("invalid part size %q", raw)
			}
			return parsed * unit.mult, nil
		}
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid part size %q", raw)
	}
	return parsed, nil
}
