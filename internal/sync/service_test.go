package sync

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

type recordingLister struct {
	input   list.ListInput
	entries []list.Entry
}

func (l *recordingLister) List(_ context.Context, input list.ListInput) ([]list.Entry, error) {
	l.input = input
	return l.entries, nil
}

type recordingUploader struct {
	inputs []upload.UploadInput
}

func (u *recordingUploader) Upload(_ context.Context, input upload.UploadInput) error {
	u.inputs = append(u.inputs, input)
	return nil
}

func TestServiceSyncUploadsOnlyChangedAndMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "same.txt"), "same")
	writeFile(t, filepath.Join(dir, "changed.txt"), "changed")
	writeFile(t, filepath.Join(dir, "new.txt"), "new")

	lister := &recordingLister{
		entries: []list.Entry{
			{Key: "site/same.txt", Size: 4},
			{Key: "site/changed.txt", Size: 1},
		},
	}
	uploader := &recordingUploader{}
	var out bytes.Buffer

	err := Service{Lister: lister, Uploader: uploader, Stdout: &out}.Sync(context.Background(), Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Progress:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lister.input.Bucket != "bucket" || lister.input.Prefix != "site/" || !lister.input.Recursive {
		t.Fatalf("unexpected list input: %+v", lister.input)
	}
	if len(uploader.inputs) != 2 {
		t.Fatalf("got %d uploads, want 2", len(uploader.inputs))
	}
	gotKeys := []string{uploader.inputs[0].Key, uploader.inputs[1].Key}
	if !contains(gotKeys, "site/changed.txt") || !contains(gotKeys, "site/new.txt") {
		t.Fatalf("unexpected uploads: %+v", gotKeys)
	}
	if !strings.Contains(out.String(), "skip") || !strings.Contains(out.String(), "upload") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestServiceSyncDryRunDoesNotRequireUploader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "new.txt"), "new")

	lister := &recordingLister{}
	var out bytes.Buffer

	err := Service{Lister: lister, Stdout: &out}.Sync(context.Background(), Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "dry-run upload") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestServiceSyncPassesExcludeToLocalPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "home")
	writeFile(t, filepath.Join(dir, "index.html.map"), "map")

	lister := &recordingLister{}
	uploader := &recordingUploader{}

	err := Service{Lister: lister, Uploader: uploader, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Exclude:     []string{"*.map"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(uploader.inputs) != 1 || uploader.inputs[0].Key != "site/index.html" {
		t.Fatalf("unexpected uploads: %+v", uploader.inputs)
	}
}

func TestServiceSyncPassesIncludeToLocalPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "home")
	writeFile(t, filepath.Join(dir, "app.js"), "js")

	lister := &recordingLister{}
	uploader := &recordingUploader{}

	err := Service{Lister: lister, Uploader: uploader, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Include:     []string{"*.html"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(uploader.inputs) != 1 || uploader.inputs[0].Key != "site/index.html" {
		t.Fatalf("unexpected uploads: %+v", uploader.inputs)
	}
}

func TestServiceSyncRequiresLister(t *testing.T) {
	t.Parallel()

	err := Service{Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:      t.TempDir(),
		Destination: list.S3Prefix{Bucket: "bucket"},
		DryRun:      true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()

	if err := uploadTestWriteFile(path, body); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
