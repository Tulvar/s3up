//go:build integration

package s3client_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tulvar/s3up/internal/config"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/s3client"
	"github.com/tulvar/s3up/internal/upload"
)

func TestUploadToMinIO(t *testing.T) {
	endpoint := envOrDefault("S3UP_INTEGRATION_ENDPOINT", "http://localhost:9000")
	accessKey := envOrDefault("S3UP_INTEGRATION_ACCESS_KEY_ID", "minioadmin")
	secretKey := envOrDefault("S3UP_INTEGRATION_SECRET_ACCESS_KEY", "minioadmin")
	region := envOrDefault("S3UP_INTEGRATION_REGION", "us-east-1")

	t.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	t.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	t.Setenv("AWS_REGION", region)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := s3.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	waitForS3(t, ctx, client)

	bucket := fmt.Sprintf("s3up-it-%d", time.Now().UnixNano())
	key := "site/index.html"

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		_, _ = client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		_, _ = client.DeleteBucket(cleanupCtx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
	})

	dir := t.TempDir()
	localFile := filepath.Join(dir, "index.html")
	if err := os.WriteFile(localFile, []byte("<h1>s3up</h1>"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	uploader, err := s3client.NewUploader(ctx, config.Config{
		Region:      region,
		EndpointURL: endpoint,
		PathStyle:   true,
		Concurrency: 2,
		PartSize:    5 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("new uploader: %v", err)
	}

	var out bytes.Buffer
	err = upload.Service{Uploader: uploader, Stdout: &out}.Upload(ctx, upload.Request{
		Source:      localFile,
		Destination: upload.S3URI{Bucket: bucket, Key: key},
		Options: upload.Options{
			ContentType: "text/html",
			Metadata:    map[string]string{"app": "s3up"},
		},
		Progress: true,
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}

	if head.ContentLength == nil || *head.ContentLength != int64(len("<h1>s3up</h1>")) {
		t.Fatalf("got content length %v", head.ContentLength)
	}
	if head.ContentType == nil || *head.ContentType != "text/html" {
		t.Fatalf("got content type %v", head.ContentType)
	}
	if head.Metadata["app"] != "s3up" {
		t.Fatalf("got metadata %+v", head.Metadata)
	}
	if out.String() == "" {
		t.Fatalf("expected progress output")
	}

	lister, err := s3client.NewLister(ctx, config.Config{
		Region:      region,
		EndpointURL: endpoint,
		PathStyle:   true,
	})
	if err != nil {
		t.Fatalf("new lister: %v", err)
	}

	entries, err := lister.List(ctx, list.ListInput{
		Bucket:    bucket,
		Prefix:    "site/",
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}

	if !hasObject(entries, key) {
		t.Fatalf("list entries do not include %q: %+v", key, entries)
	}
}

func waitForS3(t *testing.T, ctx context.Context, client *s3.Client) {
	t.Helper()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		_, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
		if err == nil {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for S3 endpoint: %v", err)
		case <-ticker.C:
		}
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func hasObject(entries []list.Entry, key string) bool {
	for _, entry := range entries {
		if !entry.IsDir && entry.Key == key {
			return true
		}
	}
	return false
}
