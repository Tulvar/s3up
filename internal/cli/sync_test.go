package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunSyncRequiresArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"sync", "--dry-run"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsageMentionsSync(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "s3up sync") {
		t.Fatalf("usage output does not mention sync: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-checksum") {
		t.Fatalf("usage output does not mention checksum: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-delete") {
		t.Fatalf("usage output does not mention delete: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-yes") {
		t.Fatalf("usage output does not mention yes: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-exclude") {
		t.Fatalf("usage output does not mention exclude: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-include") {
		t.Fatalf("usage output does not mention include: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-workers") {
		t.Fatalf("usage output does not mention workers: %q", stdout.String())
	}
}
