package sync

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	stdsync "sync"
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
	mu     stdsync.Mutex
	inputs []upload.UploadInput
}

func (u *recordingUploader) Upload(_ context.Context, input upload.UploadInput) error {
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

type recordingDeleter struct {
	mu     stdsync.Mutex
	inputs []DeleteInput
}

func (d *recordingDeleter) Delete(_ context.Context, input DeleteInput) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inputs = append(d.inputs, input)
	return nil
}

func (d *recordingDeleter) len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.inputs)
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
	if !strings.Contains(out.String(), "summary: uploaded=2 deleted=0 skipped=1") {
		t.Fatalf("expected summary output, got %q", out.String())
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
	if strings.Contains(out.String(), "summary:") {
		t.Fatalf("did not expect summary without progress, got %q", out.String())
	}
}

func TestServiceSyncDryRunPrintsSummaryWithProgress(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "new.txt"), "new")

	lister := &recordingLister{
		entries: []list.Entry{{Key: "site/old.txt", Size: 3}},
	}
	var out bytes.Buffer

	err := Service{Lister: lister, Stdout: &out}.Sync(context.Background(), Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Delete:      true,
		DryRun:      true,
		Progress:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "summary: planned uploads=1 planned deletes=1 skipped=0") {
		t.Fatalf("expected summary output, got %q", out.String())
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

func TestServiceSyncDeleteRemovesRemoteOnlyObjects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "home")

	lister := &recordingLister{
		entries: []list.Entry{
			{Key: "site/index.html", Size: 4},
			{Key: "site/old.html", Size: 3},
		},
	}
	uploader := &recordingUploader{}
	deleter := &recordingDeleter{}
	var out bytes.Buffer

	err := Service{Lister: lister, Uploader: uploader, Deleter: deleter, Stdout: &out}.Sync(context.Background(), Request{
		Source:        dir,
		Destination:   list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Delete:        true,
		ConfirmDelete: true,
		Progress:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deleter.inputs) != 1 || deleter.inputs[0].Key != "site/old.html" {
		t.Fatalf("unexpected deletes: %+v", deleter.inputs)
	}
	if !strings.Contains(out.String(), "delete") {
		t.Fatalf("expected delete output, got %q", out.String())
	}
}

func TestServiceSyncDeleteRejectsRootSymlinkBeforeListingOrDeleting(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	writeFile(t, filepath.Join(target, "index.html"), "home")
	link := filepath.Join(t.TempDir(), "source")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	lister := &recordingLister{
		entries: []list.Entry{{Key: "site/keep.html", Size: 4}},
	}
	deleter := &recordingDeleter{}
	err := Service{
		Lister:   lister,
		Uploader: &recordingUploader{},
		Deleter:  deleter,
		Stdout:   &bytes.Buffer{},
	}.Sync(context.Background(), Request{
		Source:        link,
		Destination:   list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Delete:        true,
		ConfirmDelete: true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "--follow-symlinks") {
		t.Fatalf("unexpected error: %v", err)
	}
	if lister.input != (list.ListInput{}) {
		t.Fatalf("remote objects were listed: %+v", lister.input)
	}
	if deleter.len() != 0 {
		t.Fatalf("got %d deletes, want 0", deleter.len())
	}
}

func TestServiceSyncDeleteDryRunDoesNotRequireDeleter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "home")

	lister := &recordingLister{
		entries: []list.Entry{{Key: "site/old.html", Size: 3}},
	}
	var out bytes.Buffer

	err := Service{Lister: lister, Stdout: &out}.Sync(context.Background(), Request{
		Source:      dir,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Delete:      true,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "dry-run delete") {
		t.Fatalf("expected dry-run delete output, got %q", out.String())
	}
}

func TestServiceSyncDeleteRequiresDeleter(t *testing.T) {
	t.Parallel()

	err := Service{Lister: &recordingLister{}, Uploader: &recordingUploader{}, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:        t.TempDir(),
		Destination:   list.S3Prefix{Bucket: "bucket"},
		Delete:        true,
		ConfirmDelete: true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestServiceSyncDeleteRequiresConfirmation(t *testing.T) {
	t.Parallel()

	err := Service{Lister: &recordingLister{}, Uploader: &recordingUploader{}, Deleter: &recordingDeleter{}, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:      t.TempDir(),
		Destination: list.S3Prefix{Bucket: "bucket"},
		Delete:      true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSyncDeleteRequiresDelimitedDestinationPrefix(t *testing.T) {
	t.Parallel()

	for _, dryRun := range []bool{false, true} {
		dryRun := dryRun
		t.Run(fmt.Sprintf("dry-run=%t", dryRun), func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "index.html"), "home")

			lister := &recordingLister{
				entries: []list.Entry{{Key: "site-backup/keep.html", Size: 4}},
			}
			deleter := &recordingDeleter{}
			err := Service{
				Lister:   lister,
				Uploader: &recordingUploader{},
				Deleter:  deleter,
				Stdout:   &bytes.Buffer{},
			}.Sync(context.Background(), Request{
				Source:        dir,
				Destination:   list.S3Prefix{Bucket: "bucket", Prefix: "site"},
				Delete:        true,
				ConfirmDelete: true,
				DryRun:        dryRun,
			})
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), "end with /") {
				t.Fatalf("unexpected error: %v", err)
			}
			if lister.input != (list.ListInput{}) {
				t.Fatalf("remote objects were listed: %+v", lister.input)
			}
			if deleter.len() != 0 {
				t.Fatalf("got %d deletes, want 0", deleter.len())
			}
		})
	}
}

func TestServiceSyncDeleteRejectsEmptyDestinationPrefix(t *testing.T) {
	t.Parallel()

	err := Service{Lister: &recordingLister{}, Uploader: &recordingUploader{}, Deleter: &recordingDeleter{}, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:        t.TempDir(),
		Destination:   list.S3Prefix{Bucket: "bucket"},
		Delete:        true,
		ConfirmDelete: true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "non-empty destination prefix") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSyncWithWorkersProcessesUploadAndDeleteActions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "new-a.txt"), "a")
	writeFile(t, filepath.Join(dir, "new-b.txt"), "b")

	lister := &recordingLister{
		entries: []list.Entry{
			{Key: "site/old-a.txt", Size: 1},
			{Key: "site/old-b.txt", Size: 1},
		},
	}
	uploader := &recordingUploader{}
	deleter := &recordingDeleter{}

	err := Service{Lister: lister, Uploader: uploader, Deleter: deleter, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:        dir,
		Destination:   list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Delete:        true,
		ConfirmDelete: true,
		Workers:       4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uploader.len() != 2 {
		t.Fatalf("got %d uploads, want 2", uploader.len())
	}
	if deleter.len() != 2 {
		t.Fatalf("got %d deletes, want 2", deleter.len())
	}
}

func TestServiceSyncRejectsNegativeWorkers(t *testing.T) {
	t.Parallel()

	err := Service{Lister: &recordingLister{}, Uploader: &recordingUploader{}, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:      t.TempDir(),
		Destination: list.S3Prefix{Bucket: "bucket"},
		Workers:     -1,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestServiceSyncRejectsTooManyWorkers(t *testing.T) {
	t.Parallel()

	err := Service{Lister: &recordingLister{}, Uploader: &recordingUploader{}, Stdout: &bytes.Buffer{}}.Sync(context.Background(), Request{
		Source:      t.TempDir(),
		Destination: list.S3Prefix{Bucket: "bucket"},
		Workers:     65,
	})
	if err == nil {
		t.Fatalf("expected error")
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
