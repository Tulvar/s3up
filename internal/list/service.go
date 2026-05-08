package list

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"
)

type Service struct {
	Lister ObjectLister
	Stdout io.Writer
}

func (s Service) List(ctx context.Context, req Request) error {
	if s.Lister == nil {
		return fmt.Errorf("lister is not configured")
	}

	entries, err := s.Lister.List(ctx, ListInput{
		Bucket:    req.Target.Bucket,
		Prefix:    req.Target.Prefix,
		Recursive: req.Recursive,
	})
	if err != nil {
		return err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entryName(entries[i]) < entryName(entries[j])
	})

	if req.JSON {
		return writeJSON(s.Stdout, entries)
	}

	for _, entry := range entries {
		if entry.IsDir {
			fmt.Fprintf(s.Stdout, "PRE %s\n", entry.Prefix)
			continue
		}
		size := fmt.Sprintf("%d", entry.Size)
		if req.Human {
			size = formatSize(entry.Size)
		}
		fmt.Fprintf(s.Stdout, "%12s %s\n", size, entry.Key)
	}

	return nil
}

type jsonEntry struct {
	Type         string     `json:"type"`
	Key          string     `json:"key,omitempty"`
	Prefix       string     `json:"prefix,omitempty"`
	Size         int64      `json:"size,omitempty"`
	LastModified *time.Time `json:"last_modified,omitempty"`
}

func writeJSON(stdout io.Writer, entries []Entry) error {
	out := make([]jsonEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			out = append(out, jsonEntry{
				Type:   "prefix",
				Prefix: entry.Prefix,
			})
			continue
		}

		item := jsonEntry{
			Type: "object",
			Key:  entry.Key,
			Size: entry.Size,
		}
		if !entry.LastModified.IsZero() {
			item.LastModified = &entry.LastModified
		}
		out = append(out, item)
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func entryName(entry Entry) string {
	if entry.IsDir {
		return entry.Prefix
	}
	return entry.Key
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
