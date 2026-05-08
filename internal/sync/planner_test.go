package sync

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

func TestPlanUploadsMissingAndChangedObjects(t *testing.T) {
	t.Parallel()

	local := []upload.PlannedUpload{
		{Key: "site/same.css", Size: 10},
		{Key: "site/changed.js", Size: 20},
		{Key: "site/new.html", Size: 30},
	}
	remote := []list.Entry{
		{Key: "site/same.css", Size: 10},
		{Key: "site/changed.js", Size: 99},
		{Prefix: "site/assets/", IsDir: true},
	}

	got, err := Plan(local, remote, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d actions, want 3", len(got))
	}
	if got[0].Kind != ActionSkip || got[0].Reason != "same size" {
		t.Fatalf("unexpected same action: %+v", got[0])
	}
	if got[1].Kind != ActionUpload || got[1].Reason != "size differs" {
		t.Fatalf("unexpected changed action: %+v", got[1])
	}
	if got[2].Kind != ActionUpload || got[2].Reason != "missing remote object" {
		t.Fatalf("unexpected new action: %+v", got[2])
	}
}

func TestPlanChecksumUploadsSameSizeDifferentContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "same-size.txt")
	if err := os.WriteFile(localPath, []byte("bbbb"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := Plan(
		[]upload.PlannedUpload{{LocalPath: localPath, Key: "site/same-size.txt", Size: 4}},
		[]list.Entry{{Key: "site/same-size.txt", Size: 4, ETag: quote(md5Hex("aaaa"))}},
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].Kind != ActionUpload || got[0].Reason != "checksum differs" {
		t.Fatalf("unexpected action: %+v", got[0])
	}
}

func TestPlanChecksumSkipsSameMD5(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "same.txt")
	if err := os.WriteFile(localPath, []byte("same"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := Plan(
		[]upload.PlannedUpload{{LocalPath: localPath, Key: "site/same.txt", Size: 4}},
		[]list.Entry{{Key: "site/same.txt", Size: 4, ETag: quote(md5Hex("same"))}},
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].Kind != ActionSkip || got[0].Reason != "same size and checksum" {
		t.Fatalf("unexpected action: %+v", got[0])
	}
}

func TestPlanChecksumUploadsWhenETagIsNotComparable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "multipart.txt")
	if err := os.WriteFile(localPath, []byte("same"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := Plan(
		[]upload.PlannedUpload{{LocalPath: localPath, Key: "site/multipart.txt", Size: 4}},
		[]list.Entry{{Key: "site/multipart.txt", Size: 4, ETag: quote(md5Hex("same") + "-2")}},
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].Kind != ActionUpload || got[0].Reason != "etag not comparable" {
		t.Fatalf("unexpected action: %+v", got[0])
	}
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func quote(value string) string {
	return `"` + value + `"`
}
