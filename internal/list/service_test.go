package list

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type recordingLister struct {
	input   ListInput
	entries []Entry
}

func (l *recordingLister) List(_ context.Context, input ListInput) ([]Entry, error) {
	l.input = input
	return l.entries, nil
}

func TestServiceListPrintsEntries(t *testing.T) {
	t.Parallel()

	lister := &recordingLister{
		entries: []Entry{
			{Key: "site/index.html", Size: 12},
			{Prefix: "site/assets/", IsDir: true},
		},
	}
	var out bytes.Buffer

	err := Service{Lister: lister, Stdout: &out}.List(context.Background(), Request{
		Target:    S3Prefix{Bucket: "bucket", Prefix: "site/"},
		Recursive: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lister.input.Bucket != "bucket" || lister.input.Prefix != "site/" || lister.input.Recursive {
		t.Fatalf("unexpected input: %+v", lister.input)
	}
	got := out.String()
	if !strings.Contains(got, "PRE site/assets/") {
		t.Fatalf("missing prefix line: %q", got)
	}
	if !strings.Contains(got, "12 site/index.html") {
		t.Fatalf("missing object line: %q", got)
	}
}

func TestServiceListPrintsHumanSizes(t *testing.T) {
	t.Parallel()

	lister := &recordingLister{
		entries: []Entry{{Key: "backup.tar", Size: 2 * 1024 * 1024}},
	}
	var out bytes.Buffer

	err := Service{Lister: lister, Stdout: &out}.List(context.Background(), Request{
		Target: S3Prefix{Bucket: "bucket"},
		Human:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "2.0 MiB backup.tar") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestServiceListPrintsJSON(t *testing.T) {
	t.Parallel()

	modified := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	lister := &recordingLister{
		entries: []Entry{
			{Prefix: "site/assets/", IsDir: true},
			{Key: "site/index.html", Size: 12, LastModified: modified},
		},
	}
	var out bytes.Buffer

	err := Service{Lister: lister, Stdout: &out}.List(context.Background(), Request{
		Target: S3Prefix{Bucket: "bucket", Prefix: "site/"},
		JSON:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v\n%s", err, out.String())
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0]["type"] != "prefix" || got[0]["prefix"] != "site/assets/" {
		t.Fatalf("unexpected prefix entry: %+v", got[0])
	}
	if got[1]["type"] != "object" || got[1]["key"] != "site/index.html" || got[1]["size"] != float64(12) {
		t.Fatalf("unexpected object entry: %+v", got[1])
	}
	if got[1]["last_modified"] == "" {
		t.Fatalf("expected last_modified in object entry: %+v", got[1])
	}
}

func TestServiceListRequiresLister(t *testing.T) {
	t.Parallel()

	err := Service{Stdout: &bytes.Buffer{}}.List(context.Background(), Request{
		Target: S3Prefix{Bucket: "bucket"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
