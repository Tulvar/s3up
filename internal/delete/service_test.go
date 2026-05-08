package delete

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/tulvar/s3up/internal/list"
)

type recordingLister struct {
	input   list.ListInput
	entries []list.Entry
}

func (l *recordingLister) List(_ context.Context, input list.ListInput) ([]list.Entry, error) {
	l.input = input
	return l.entries, nil
}

type recordingDeleter struct {
	mu     sync.Mutex
	inputs []DeleteInput
}

func (d *recordingDeleter) Delete(_ context.Context, input DeleteInput) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inputs = append(d.inputs, input)
	return nil
}

func (d *recordingDeleter) len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.inputs)
}

func TestServiceDeleteSingleObject(t *testing.T) {
	t.Parallel()

	deleter := &recordingDeleter{}
	var out bytes.Buffer
	err := Service{Deleter: deleter, Stdout: &out}.Delete(context.Background(), Request{
		Target:   list.S3Prefix{Bucket: "bucket", Prefix: "site/index.html"},
		Progress: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleter.len() != 1 || deleter.inputs[0].Key != "site/index.html" {
		t.Fatalf("unexpected deletes: %+v", deleter.inputs)
	}
	if !strings.Contains(out.String(), "summary: deleted=1") {
		t.Fatalf("expected summary, got %q", out.String())
	}
}

func TestServiceDeleteRecursiveRequiresConfirmation(t *testing.T) {
	t.Parallel()

	err := Service{Lister: &recordingLister{}, Deleter: &recordingDeleter{}}.Delete(context.Background(), Request{
		Target:    list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Recursive: true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeleteRecursiveDryRun(t *testing.T) {
	t.Parallel()

	lister := &recordingLister{
		entries: []list.Entry{
			{Key: "site/index.html"},
			{Key: "site/app.js"},
			{Key: "site/app.js.map"},
		},
	}
	var out bytes.Buffer
	err := Service{Lister: lister, Stdout: &out}.Delete(context.Background(), Request{
		Target:    list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Recursive: true,
		DryRun:    true,
		Include:   []string{"*.js", "*.html"},
		Exclude:   []string{"*.map"},
		Progress:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lister.input.Bucket != "bucket" || lister.input.Prefix != "site/" || !lister.input.Recursive {
		t.Fatalf("unexpected list input: %+v", lister.input)
	}
	if strings.Contains(out.String(), "app.js.map") {
		t.Fatalf("excluded object appears in output: %q", out.String())
	}
	if !strings.Contains(out.String(), "summary: planned deletes=2") {
		t.Fatalf("expected summary, got %q", out.String())
	}
}

func TestServiceDeleteRecursiveWithWorkers(t *testing.T) {
	t.Parallel()

	lister := &recordingLister{
		entries: []list.Entry{
			{Key: "site/a.txt"},
			{Key: "site/b.txt"},
			{Key: "site/c.txt"},
		},
	}
	deleter := &recordingDeleter{}
	err := Service{Lister: lister, Deleter: deleter}.Delete(context.Background(), Request{
		Target:        list.S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Recursive:     true,
		ConfirmDelete: true,
		Workers:       3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleter.len() != 3 {
		t.Fatalf("got %d deletes, want 3", deleter.len())
	}
}

func TestServiceDeleteRejectsNegativeWorkers(t *testing.T) {
	t.Parallel()

	err := Service{Deleter: &recordingDeleter{}}.Delete(context.Background(), Request{
		Target:  list.S3Prefix{Bucket: "bucket", Prefix: "site/index.html"},
		Workers: -1,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestServiceDeleteRejectsTooManyWorkers(t *testing.T) {
	t.Parallel()

	err := Service{Deleter: &recordingDeleter{}}.Delete(context.Background(), Request{
		Target:  list.S3Prefix{Bucket: "bucket", Prefix: "site/index.html"},
		Workers: 65,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
