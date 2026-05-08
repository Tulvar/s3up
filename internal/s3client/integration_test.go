//go:build integration

package s3client_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tulvar/s3up/internal/config"
	deletepkg "github.com/tulvar/s3up/internal/delete"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/s3client"
	syncer "github.com/tulvar/s3up/internal/sync"
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

func TestSyncToMinIOUploadsOnlyChangedAndNewObjects(t *testing.T) {
	endpoint := envOrDefault("S3UP_INTEGRATION_ENDPOINT", "http://localhost:9000")
	accessKey := envOrDefault("S3UP_INTEGRATION_ACCESS_KEY_ID", "minioadmin")
	secretKey := envOrDefault("S3UP_INTEGRATION_SECRET_ACCESS_KEY", "minioadmin")
	region := envOrDefault("S3UP_INTEGRATION_REGION", "us-east-1")

	t.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	t.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	t.Setenv("AWS_REGION", region)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.Config{
		Region:      region,
		EndpointURL: endpoint,
		PathStyle:   true,
	}
	client := s3.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	waitForS3(t, ctx, client)

	bucket := fmt.Sprintf("s3up-sync-it-%d", time.Now().UnixNano())
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		for _, key := range []string{"site/same.txt", "site/changed.txt", "site/new.txt", "site/old.txt", "site/protected.map", "site/ignored.map"} {
			_, _ = client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
		}
		_, _ = client.DeleteBucket(cleanupCtx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
	})

	putObject(t, ctx, client, bucket, "site/same.txt", "same")
	putObject(t, ctx, client, bucket, "site/changed.txt", "old")
	putObject(t, ctx, client, bucket, "site/old.txt", "old")
	putObject(t, ctx, client, bucket, "site/protected.map", "protected")

	dir := t.TempDir()
	writeLocalFile(t, filepath.Join(dir, "same.txt"), "DIFF")
	writeLocalFile(t, filepath.Join(dir, "changed.txt"), "changed")
	writeLocalFile(t, filepath.Join(dir, "new.txt"), "new")
	writeLocalFile(t, filepath.Join(dir, "ignored.map"), "ignored")

	lister, err := s3client.NewLister(ctx, cfg)
	if err != nil {
		t.Fatalf("new lister: %v", err)
	}
	uploader, err := s3client.NewUploader(ctx, cfg)
	if err != nil {
		t.Fatalf("new uploader: %v", err)
	}

	var out bytes.Buffer
	err = syncer.Service{Lister: lister, Uploader: uploader, Stdout: &out}.Sync(ctx, syncer.Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: bucket, Prefix: "site/"},
		Include:     []string{"*.txt"},
		Exclude:     []string{"*.map"},
		Progress:    true,
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if body := getObjectBody(t, ctx, client, bucket, "site/same.txt"); body != "same" {
		t.Fatalf("same object should be skipped, got body %q", body)
	}
	if body := getObjectBody(t, ctx, client, bucket, "site/changed.txt"); body != "changed" {
		t.Fatalf("changed object body = %q", body)
	}
	if body := getObjectBody(t, ctx, client, bucket, "site/new.txt"); body != "new" {
		t.Fatalf("new object body = %q", body)
	}
	if objectExists(t, ctx, client, bucket, "site/ignored.map") {
		t.Fatalf("excluded object was uploaded")
	}
	if !objectExists(t, ctx, client, bucket, "site/old.txt") {
		t.Fatalf("plain sync should not delete extra remote object")
	}

	out.Reset()
	err = syncer.Service{Lister: lister, Uploader: uploader, Stdout: &out}.Sync(ctx, syncer.Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: bucket, Prefix: "site/"},
		Checksum:    true,
		Include:     []string{"*.txt"},
		Exclude:     []string{"*.map"},
		Progress:    true,
	})
	if err != nil {
		t.Fatalf("checksum sync: %v", err)
	}

	if body := getObjectBody(t, ctx, client, bucket, "site/same.txt"); body != "DIFF" {
		t.Fatalf("checksum sync should update same-size changed object, got body %q", body)
	}

	deleter, err := s3client.NewDeleter(ctx, cfg)
	if err != nil {
		t.Fatalf("new deleter: %v", err)
	}

	out.Reset()
	err = syncer.Service{Lister: lister, Uploader: uploader, Stdout: &out}.Sync(ctx, syncer.Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: bucket, Prefix: "site/"},
		Delete:      true,
		DryRun:      true,
		Include:     []string{"*.txt"},
		Exclude:     []string{"*.map"},
	})
	if err != nil {
		t.Fatalf("dry-run delete sync: %v", err)
	}
	if !objectExists(t, ctx, client, bucket, "site/old.txt") {
		t.Fatalf("dry-run delete should not delete extra remote object")
	}

	out.Reset()
	err = syncer.Service{Lister: lister, Uploader: uploader, Deleter: deleter, Stdout: &out}.Sync(ctx, syncer.Request{
		Source:        dir,
		Destination:   list.S3Prefix{Bucket: bucket, Prefix: "site/"},
		Delete:        true,
		ConfirmDelete: true,
		Include:       []string{"*.txt"},
		Exclude:       []string{"*.map"},
		Progress:      true,
	})
	if err != nil {
		t.Fatalf("delete sync: %v", err)
	}
	if objectExists(t, ctx, client, bucket, "site/old.txt") {
		t.Fatalf("delete sync should delete extra remote object")
	}
	if !objectExists(t, ctx, client, bucket, "site/protected.map") {
		t.Fatalf("delete sync should respect exclude filters")
	}
}

func TestDeleteFromMinIO(t *testing.T) {
	endpoint := envOrDefault("S3UP_INTEGRATION_ENDPOINT", "http://localhost:9000")
	accessKey := envOrDefault("S3UP_INTEGRATION_ACCESS_KEY_ID", "minioadmin")
	secretKey := envOrDefault("S3UP_INTEGRATION_SECRET_ACCESS_KEY", "minioadmin")
	region := envOrDefault("S3UP_INTEGRATION_REGION", "us-east-1")

	t.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	t.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	t.Setenv("AWS_REGION", region)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.Config{
		Region:      region,
		EndpointURL: endpoint,
		PathStyle:   true,
	}
	client := s3.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	waitForS3(t, ctx, client)

	bucket := fmt.Sprintf("s3up-delete-it-%d", time.Now().UnixNano())
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		for _, key := range []string{"site/a.txt", "site/b.txt", "site/protected.map"} {
			_, _ = client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
		}
		_, _ = client.DeleteBucket(cleanupCtx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
	})

	putObject(t, ctx, client, bucket, "site/a.txt", "a")
	putObject(t, ctx, client, bucket, "site/b.txt", "b")
	putObject(t, ctx, client, bucket, "site/protected.map", "map")

	lister, err := s3client.NewLister(ctx, cfg)
	if err != nil {
		t.Fatalf("new lister: %v", err)
	}
	deleter, err := s3client.NewDeleter(ctx, cfg)
	if err != nil {
		t.Fatalf("new deleter: %v", err)
	}

	var out bytes.Buffer
	err = deletepkg.Service{Lister: lister, Stdout: &out}.Delete(ctx, deletepkg.Request{
		Target:    list.S3Prefix{Bucket: bucket, Prefix: "site/"},
		Recursive: true,
		DryRun:    true,
		Exclude:   []string{"*.map"},
		Progress:  true,
	})
	if err != nil {
		t.Fatalf("dry-run delete: %v", err)
	}
	if !objectExists(t, ctx, client, bucket, "site/a.txt") || !objectExists(t, ctx, client, bucket, "site/b.txt") {
		t.Fatalf("dry-run delete removed objects")
	}

	out.Reset()
	err = deletepkg.Service{Lister: lister, Deleter: deleter, Stdout: &out}.Delete(ctx, deletepkg.Request{
		Target:        list.S3Prefix{Bucket: bucket, Prefix: "site/"},
		Recursive:     true,
		ConfirmDelete: true,
		Exclude:       []string{"*.map"},
		Progress:      true,
		Workers:       2,
	})
	if err != nil {
		t.Fatalf("recursive delete: %v", err)
	}
	if objectExists(t, ctx, client, bucket, "site/a.txt") || objectExists(t, ctx, client, bucket, "site/b.txt") {
		t.Fatalf("recursive delete left matching objects")
	}
	if !objectExists(t, ctx, client, bucket, "site/protected.map") {
		t.Fatalf("recursive delete should respect exclude filters")
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

func putObject(t *testing.T, ctx context.Context, client *s3.Client, bucket, key, body string) {
	t.Helper()

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(body)),
	})
	if err != nil {
		t.Fatalf("put object %s: %v", key, err)
	}
}

func getObjectBody(t *testing.T, ctx context.Context, client *s3.Client, bucket, key string) string {
	t.Helper()

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("get object %s: %v", key, err)
	}
	defer out.Body.Close()

	body, err := io.ReadAll(out.Body)
	if err != nil {
		t.Fatalf("read object %s: %v", key, err)
	}
	return string(body)
}

func objectExists(t *testing.T, ctx context.Context, client *s3.Client, bucket, key string) bool {
	t.Helper()

	_, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

func writeLocalFile(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
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
