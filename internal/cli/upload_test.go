package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUploadDryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "artifact.txt")
	writeFile(t, file, "hello")

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{
		"upload",
		"--dry-run",
		file,
		"s3://bucket/uploads/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout.String(), "dry-run:") {
		t.Fatalf("expected dry-run output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "s3://bucket/uploads/artifact.txt") {
		t.Fatalf("expected destination in output, got %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunUploadRequiresTwoArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"upload", "--dry-run"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("unexpected error: %v", err)
	}
}
