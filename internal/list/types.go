package list

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type S3Prefix struct {
	Bucket string
	Prefix string
}

func ParseS3Prefix(raw string) (S3Prefix, error) {
	const scheme = "s3://"
	if !strings.HasPrefix(raw, scheme) {
		return S3Prefix{}, fmt.Errorf("target must use s3://bucket or s3://bucket/prefix format")
	}

	rest := strings.TrimPrefix(raw, scheme)
	if rest == "" {
		return S3Prefix{}, fmt.Errorf("target must include a bucket")
	}

	bucket, prefix, ok := strings.Cut(rest, "/")
	if bucket == "" {
		return S3Prefix{}, fmt.Errorf("target must include a bucket")
	}
	if !ok {
		return S3Prefix{Bucket: bucket}, nil
	}
	return S3Prefix{Bucket: bucket, Prefix: strings.TrimLeft(prefix, "/")}, nil
}

type Request struct {
	Target    S3Prefix
	Recursive bool
	Human     bool
	JSON      bool
}

type ListInput struct {
	Bucket    string
	Prefix    string
	Recursive bool
}

type Entry struct {
	Key          string
	Prefix       string
	IsDir        bool
	Size         int64
	ETag         string
	LastModified time.Time
}

type ObjectLister interface {
	List(ctx context.Context, input ListInput) ([]Entry, error)
}
