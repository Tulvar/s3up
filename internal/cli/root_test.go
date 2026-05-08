package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPrintsUsageWithoutArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "s3up upload") {
		t.Fatalf("usage output does not mention upload: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"wat"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}
