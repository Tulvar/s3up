package upload

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
		LocalPath: file,
		Bucket:    "bucket",
		Key:       "uploads/custom.txt",
		Size:      5,
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
