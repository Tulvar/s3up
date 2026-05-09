package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunLSRequiresTarget(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"ls"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsageMentionsLS(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "s3up ls") {
		t.Fatalf("usage output does not mention ls: %q", stdout.String())
	}
}

func TestRunLSRejectsHumanAndJSONTogether(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{
		"ls",
		"--human",
		"--json",
		"s3://bucket/site/",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLSAcceptsFlagsAfterTarget(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := New(&stdout, &stderr).Run(context.Background(), []string{
		"ls",
		"s3://bucket/site/",
		"--human",
		"--json",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}
