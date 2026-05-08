package delete

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

type Service struct {
	Lister  list.ObjectLister
	Deleter ObjectDeleter
	Stdout  io.Writer
}

func (s Service) Delete(ctx context.Context, req Request) error {
	if req.Target.Prefix == "" {
		return fmt.Errorf("delete target must include an object key or prefix")
	}

	workers := req.Workers
	if workers == 0 {
		workers = 1
	}
	if workers < 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	keys, err := s.plan(ctx, req)
	if err != nil {
		return err
	}

	if req.DryRun {
		for i, key := range keys {
			if s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "dry-run delete [%d/%d]: s3://%s/%s\n", i+1, len(keys), req.Target.Bucket, key)
			}
		}
		if req.Progress && s.Stdout != nil {
			fmt.Fprintf(s.Stdout, "summary: planned deletes=%d\n", len(keys))
		}
		return nil
	}

	if len(keys) > 1 && !req.ConfirmDelete {
		return fmt.Errorf("recursive delete requires --yes unless --dry-run is set")
	}
	if s.Deleter == nil {
		return fmt.Errorf("deleter is not configured")
	}

	if err := s.deleteKeys(ctx, req.Target.Bucket, keys, workers, req.Progress); err != nil {
		return err
	}
	if req.Progress && s.Stdout != nil {
		fmt.Fprintf(s.Stdout, "summary: deleted=%d\n", len(keys))
	}
	return nil
}

func (s Service) plan(ctx context.Context, req Request) ([]string, error) {
	if !req.Recursive {
		return []string{req.Target.Prefix}, nil
	}
	if !req.DryRun && !req.ConfirmDelete {
		return nil, fmt.Errorf("recursive delete requires --yes unless --dry-run is set")
	}
	if s.Lister == nil {
		return nil, fmt.Errorf("lister is not configured")
	}

	entries, err := s.Lister.List(ctx, list.ListInput{
		Bucket:    req.Target.Bucket,
		Prefix:    req.Target.Prefix,
		Recursive: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		rel := relativeKey(entry.Key, req.Target.Prefix)
		if upload.Selected(rel, req.Include, req.Exclude) {
			keys = append(keys, entry.Key)
		}
	}
	return keys, nil
}

func (s Service) deleteKeys(ctx context.Context, bucket string, keys []string, workers int, progress bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan deleteJob)
	errs := make(chan error, 1)
	var wg sync.WaitGroup
	var outputMu sync.Mutex

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if progress && s.Stdout != nil {
					outputMu.Lock()
					fmt.Fprintf(s.Stdout, "delete [%d/%d]: s3://%s/%s\n", job.Index, job.Total, bucket, job.Key)
					outputMu.Unlock()
				}
				if err := s.Deleter.Delete(ctx, DeleteInput{Bucket: bucket, Key: job.Key}); err != nil {
					select {
					case errs <- fmt.Errorf("delete %s: %w", job.Key, err):
						cancel()
					default:
					}
					return
				}
			}
		}()
	}

sendJobs:
	for i, key := range keys {
		select {
		case <-ctx.Done():
			break sendJobs
		case jobs <- deleteJob{Index: i + 1, Total: len(keys), Key: key}:
		}
	}
	close(jobs)
	wg.Wait()

	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}

type deleteJob struct {
	Index int
	Total int
	Key   string
}

func relativeKey(key, prefix string) string {
	return upload.RelativeKey(key, prefix)
}
