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

	if !strings.Contains(stdout.String(), "dry-run [1/1]:") {
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

func TestMetadataValues(t *testing.T) {
	t.Parallel()

	var values metadataValues
	if err := values.Set("env=test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := values.Set("owner=platform"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := values.Map()
	if got["env"] != "test" || got["owner"] != "platform" {
		t.Fatalf("got metadata %+v", got)
	}

	got["env"] = "prod"
	if values["env"] != "test" {
		t.Fatalf("metadata map was not cloned")
	}
}

func TestMetadataValuesRejectInvalidFormat(t *testing.T) {
	t.Parallel()

	var values metadataValues
	if err := values.Set("env"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseByteSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want int64
	}{
		{raw: "8388608", want: 8388608},
		{raw: "8MiB", want: 8 * 1024 * 1024},
		{raw: "8MB", want: 8 * 1000 * 1000},
		{raw: "2GiB", want: 2 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()

			got, err := parseByteSize(tt.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseByteSizeRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, err := parseByteSize("0MiB"); err == nil {
		t.Fatalf("expected error")
	}
}
