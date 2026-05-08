package upload

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type Service struct {
	Uploader ObjectUploader
	Stdout   io.Writer
}

func (s Service) Upload(ctx context.Context, req Request) error {
	planned, err := Plan(req)
	if err != nil {
		return err
	}
	workers := req.Workers
	if workers == 0 {
		workers = 1
	}
	if workers < 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	if req.DryRun {
		for i, item := range planned {
			if s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "dry-run [%d/%d]: %s -> s3://%s/%s (%s)\n", i+1, len(planned), item.LocalPath, item.Bucket, item.Key, formatSize(item.Size))
			}
		}
		if req.Progress && s.Stdout != nil {
			fmt.Fprintf(s.Stdout, "summary: planned uploads=%d\n", len(planned))
		}
		return nil
	}

	if s.Uploader == nil {
		return fmt.Errorf("uploader is not configured")
	}

	if err := runUploadWorkers(ctx, planned, workers, req.Progress, s.Stdout, s.Uploader); err != nil {
		return err
	}
	if req.Progress && s.Stdout != nil {
		fmt.Fprintf(s.Stdout, "summary: uploaded=%d\n", len(planned))
	}
	return nil
}

func runUploadWorkers(ctx context.Context, planned []PlannedUpload, workers int, progress bool, stdout io.Writer, uploader ObjectUploader) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan uploadJob)
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

				if progress && stdout != nil {
					outputMu.Lock()
					fmt.Fprintf(stdout, "uploading [%d/%d]: %s -> s3://%s/%s (%s)\n", job.Index, job.Total, job.Item.LocalPath, job.Item.Bucket, job.Item.Key, formatSize(job.Item.Size))
					outputMu.Unlock()
				}
				if err := uploader.Upload(ctx, UploadInput(job.Item)); err != nil {
					select {
					case errs <- fmt.Errorf("upload %s: %w", job.Item.LocalPath, err):
						cancel()
					default:
					}
					return
				}
				if progress && stdout != nil {
					outputMu.Lock()
					fmt.Fprintf(stdout, "uploaded [%d/%d]: %s -> s3://%s/%s\n", job.Index, job.Total, job.Item.LocalPath, job.Item.Bucket, job.Item.Key)
					outputMu.Unlock()
				}
			}
		}()
	}

sendJobs:
	for i, item := range planned {
		select {
		case <-ctx.Done():
			break sendJobs
		case jobs <- uploadJob{Index: i + 1, Total: len(planned), Item: item}:
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

type uploadJob struct {
	Index int
	Total int
	Item  PlannedUpload
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
