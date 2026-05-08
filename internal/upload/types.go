package upload

import (
	"context"
	"fmt"
	"strings"
)

type S3URI struct {
	Bucket string
	Key    string
}

func ParseS3URI(raw string) (S3URI, error) {
	const prefix = "s3://"
	if !strings.HasPrefix(raw, prefix) {
		return S3URI{}, fmt.Errorf("destination must use s3://bucket/key format")
	}

	rest := strings.TrimPrefix(raw, prefix)
	bucket, key, ok := strings.Cut(rest, "/")
	if !ok || bucket == "" {
		return S3URI{}, fmt.Errorf("destination must include a bucket and key")
	}

	return S3URI{Bucket: bucket, Key: strings.TrimLeft(key, "/")}, nil
}

type Request struct {
	Source      string
	Destination S3URI
	Recursive   bool
	DryRun      bool
	Options     Options
	Progress    bool
}

type Options struct {
	ContentType  string
	Metadata     map[string]string
	StorageClass string
	ACL          string
}

type PlannedUpload struct {
	LocalPath    string
	Bucket       string
	Key          string
	Size         int64
	ContentType  string
	Metadata     map[string]string
	StorageClass string
	ACL          string
}

type UploadInput struct {
	LocalPath    string
	Bucket       string
	Key          string
	Size         int64
	ContentType  string
	Metadata     map[string]string
	StorageClass string
	ACL          string
}

type ObjectUploader interface {
	Upload(ctx context.Context, input UploadInput) error
}
