package upload

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

type recordingUploader struct {
	inputs []UploadInput
}

func (u *recordingUploader) Upload(_ context.Context, input UploadInput) error {
	u.inputs = append(u.inputs, input)
	return nil
}

func TestServiceUploadUsesPlannedUploads(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	uploader := &recordingUploader{}
	var out bytes.Buffer

	err := Service{Uploader: uploader, Stdout: &out}.Upload(context.Background(), Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		Progress:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(uploader.inputs) != 1 {
		t.Fatalf("got %d uploads, want 1", len(uploader.inputs))
	}
	if uploader.inputs[0].Key != "uploads/artifact.txt" {
		t.Fatalf("got key %q", uploader.inputs[0].Key)
	}
	if out.String() == "" {
		t.Fatalf("expected progress output")
	}
}

func TestServiceUploadCanDisableProgress(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	uploader := &recordingUploader{}
	var out bytes.Buffer

	err := Service{Uploader: uploader, Stdout: &out}.Upload(context.Background(), Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		Progress:    false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("expected no progress output, got %q", out.String())
	}
}

func TestServiceDryRunDoesNotRequireUploader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	var out bytes.Buffer
	err := Service{Stdout: &out}.Upload(context.Background(), Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected dry-run output")
	}
}
