package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunDeleteRequiresTarget(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"delete"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsageMentionsDelete(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "s3up delete") {
		t.Fatalf("usage output does not mention delete: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-recursive") {
		t.Fatalf("usage output does not mention recursive: %q", stdout.String())
	}
}
