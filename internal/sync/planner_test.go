package sync

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
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

	got, err := Plan(local, remote, Request{})
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
		Request{Checksum: true},
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
		Request{Checksum: true},
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
		Request{Checksum: true},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].Kind != ActionUpload || got[0].Reason != "etag not comparable" {
		t.Fatalf("unexpected action: %+v", got[0])
	}
}

func TestPlanDeleteUploadsRemoteOnlyObjects(t *testing.T) {
	t.Parallel()

	got, err := Plan(
		[]upload.PlannedUpload{{Key: "site/index.html", Size: 4}},
		[]list.Entry{
			{Key: "site/index.html", Size: 4},
			{Key: "site/old.html", Size: 3},
		},
		Request{
			Delete:      true,
			Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d actions, want 2", len(got))
	}
	if got[1].Kind != ActionDelete || got[1].Remote.Key != "site/old.html" || got[1].Reason != "missing local file" {
		t.Fatalf("unexpected delete action: %+v", got[1])
	}
}

func TestPlanDeleteRespectsIncludeAndExclude(t *testing.T) {
	t.Parallel()

	got, err := Plan(
		nil,
		[]list.Entry{
			{Key: "site/old.html", Size: 3},
			{Key: "site/old.js", Size: 3},
			{Key: "site/draft.html", Size: 3},
		},
		Request{
			Delete:      true,
			Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
			Include:     []string{"*.html"},
			Exclude:     []string{"draft.html"},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got actions %+v, want one delete", got)
	}
	if got[0].Kind != ActionDelete || got[0].Remote.Key != "site/old.html" {
		t.Fatalf("unexpected action: %+v", got[0])
	}
}

func TestBuildLocalPlanRejectsRootSymlinkByDefault(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "index.html"), []byte("home"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	link := filepath.Join(t.TempDir(), "source")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := BuildLocalPlan(Request{
		Source:      link,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "--follow-symlinks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLocalPlanFollowsRootSymlinkWhenEnabled(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	localPath := filepath.Join(target, "index.html")
	if err := os.WriteFile(localPath, []byte("home"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	link := filepath.Join(t.TempDir(), "source")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := BuildLocalPlan(Request{
		Source:      link,
		Destination: list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		FollowLinks: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Key != "site/index.html" {
		t.Fatalf("unexpected plan: %+v", got)
	}
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func quote(value string) string {
	return `"` + value + `"`
}
