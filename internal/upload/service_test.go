package upload

import (
	"bytes"
	"context"
	"path/filepath"
	"sync"
	"testing"
)

type recordingUploader struct {
	mu     sync.Mutex
	inputs []UploadInput
}

func (u *recordingUploader) Upload(_ context.Context, input UploadInput) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.inputs = append(u.inputs, input)
	return nil
}

func (u *recordingUploader) len() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.inputs)
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
	if !bytes.Contains(out.Bytes(), []byte("summary: uploaded=1")) {
		t.Fatalf("expected summary output, got %q", out.String())
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
	if bytes.Contains(out.Bytes(), []byte("summary:")) {
		t.Fatalf("did not expect summary without progress, got %q", out.String())
	}
}

func TestServiceDryRunPrintsSummaryWithProgress(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	var out bytes.Buffer
	err := Service{Stdout: &out}.Upload(context.Background(), Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		DryRun:      true,
		Progress:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("summary: planned uploads=1")) {
		t.Fatalf("expected summary output, got %q", out.String())
	}
}

func TestServiceUploadWithWorkersUploadsAllFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "a")
	writeFile(t, filepath.Join(dir, "b.txt"), "b")
	writeFile(t, filepath.Join(dir, "c.txt"), "c")

	uploader := &recordingUploader{}
	err := Service{Uploader: uploader}.Upload(context.Background(), Request{
		Source:      dir,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		Recursive:   true,
		Workers:     3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uploader.len() != 3 {
		t.Fatalf("got %d uploads, want 3", uploader.len())
	}
}

func TestServiceUploadRejectsNegativeWorkers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	err := Service{Uploader: &recordingUploader{}}.Upload(context.Background(), Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		Workers:     -1,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
