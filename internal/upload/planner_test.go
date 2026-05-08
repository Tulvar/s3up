package upload

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestPlanSingleFileUsesExplicitDestinationKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	got, err := Plan(Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/custom.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []PlannedUpload{{
		LocalPath:   file,
		Bucket:      "bucket",
		Key:         "uploads/custom.txt",
		Size:        5,
		ContentType: "text/plain; charset=utf-8",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPlanSingleFileAppendsNameWhenDestinationIsPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	got, err := Plan(Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].Key != "uploads/artifact.txt" {
		t.Fatalf("got key %q", got[0].Key)
	}
}

func TestPlanDirectoryRequiresRecursive(t *testing.T) {
	t.Parallel()

	_, err := Plan(Request{
		Source:      t.TempDir(),
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlanDirectoryRecursive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "home")
	writeFile(t, filepath.Join(dir, "assets", "app.css"), "css")

	got, err := Plan(Request{
		Source:      dir,
		Destination: S3URI{Bucket: "bucket", Key: "site/"},
		Recursive:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Slice(got, func(i, j int) bool {
		return got[i].Key < got[j].Key
	})

	keys := []string{got[0].Key, got[1].Key}
	want := []string{"site/assets/app.css", "site/index.html"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("got keys %+v, want %+v", keys, want)
	}
}

func TestPlanAppliesUploadOptions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.html")
	writeFile(t, file, "hello")

	got, err := Plan(Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		Options: Options{
			ContentType:  "text/custom",
			Metadata:     map[string]string{"env": "test"},
			StorageClass: "STANDARD_IA",
			ACL:          "bucket-owner-full-control",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := got[0]
	if item.ContentType != "text/custom" {
		t.Fatalf("got content type %q", item.ContentType)
	}
	if item.Metadata["env"] != "test" {
		t.Fatalf("got metadata %+v", item.Metadata)
	}
	if item.StorageClass != "STANDARD_IA" {
		t.Fatalf("got storage class %q", item.StorageClass)
	}
	if item.ACL != "bucket-owner-full-control" {
		t.Fatalf("got acl %q", item.ACL)
	}
}

func TestPlanDetectsContentTypeFromExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "index.html")
	writeFile(t, file, "hello")

	got, err := Plan(Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(got[0].ContentType, "text/html") {
		t.Fatalf("got content type %q", got[0].ContentType)
	}
}

func TestPlanClonesMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	metadata := map[string]string{"env": "test"}
	got, err := Plan(Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "uploads/"},
		Options:     Options{Metadata: metadata},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metadata["env"] = "prod"
	if got[0].Metadata["env"] != "test" {
		t.Fatalf("metadata was not cloned: %+v", got[0].Metadata)
	}
}

func TestPlanCleansDestinationKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	got, err := Plan(Request{
		Source:      file,
		Destination: S3URI{Bucket: "bucket", Key: "/uploads//nested/../artifact.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got[0].Key != "uploads/artifact.txt" {
		t.Fatalf("got key %q", got[0].Key)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
