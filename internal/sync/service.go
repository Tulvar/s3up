package sync

import (
	"context"
	"fmt"
	"io"
	stdsync "sync"

	"github.com/tulvar/s3up/internal/limits"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

type Service struct {
	Lister   list.ObjectLister
	Uploader upload.ObjectUploader
	Deleter  ObjectDeleter
	Stdout   io.Writer
}

func (s Service) Sync(ctx context.Context, req Request) error {
	if s.Lister == nil {
		return fmt.Errorf("lister is not configured")
	}
	if !req.DryRun && s.Uploader == nil {
		return fmt.Errorf("uploader is not configured")
	}
	if req.Delete && !req.DryRun && !req.ConfirmDelete {
		return fmt.Errorf("--delete requires --yes unless --dry-run is set")
	}
	if req.Delete && req.Destination.Prefix == "" {
		return fmt.Errorf("--delete requires a non-empty destination prefix")
	}
	if req.Delete && !req.DryRun && s.Deleter == nil {
		return fmt.Errorf("deleter is not configured")
	}
	workers := req.Workers
	if workers == 0 {
		workers = 1
	}
	if workers < 0 {
		return fmt.Errorf("workers must be greater than 0")
	}
	if workers > limits.MaxWorkers {
		return fmt.Errorf("workers must be less than or equal to %d", limits.MaxWorkers)
	}

	local, err := BuildLocalPlan(req)
	if err != nil {
		return err
	}

	remote, err := s.Lister.List(ctx, list.ListInput{
		Bucket:    req.Destination.Bucket,
		Prefix:    req.Destination.Prefix,
		Recursive: true,
	})
	if err != nil {
		return fmt.Errorf("list remote objects: %w", err)
	}

	actions, err := Plan(local, remote, req)
	if err != nil {
		return err
	}
	total := len(actions)
	if req.DryRun {
		for i, action := range actions {
			printDryRunAction(s.Stdout, i+1, total, action)
		}
		if req.Progress && s.Stdout != nil {
			printSummary(s.Stdout, summarize(actions), true)
		}
		return nil
	}

	if err := s.runActions(ctx, actions, workers, req); err != nil {
		return err
	}
	if req.Progress && s.Stdout != nil {
		printSummary(s.Stdout, summarize(actions), false)
	}
	return nil
}

func (s Service) runActions(ctx context.Context, actions []Action, workers int, req Request) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan actionJob)
	errs := make(chan error, 1)
	var wg stdsync.WaitGroup
	var outputMu stdsync.Mutex

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

				if err := s.runAction(ctx, job, req, &outputMu); err != nil {
					select {
					case errs <- err:
						cancel()
					default:
					}
					return
				}
			}
		}()
	}

sendJobs:
	for i, action := range actions {
		select {
		case <-ctx.Done():
			break sendJobs
		case jobs <- actionJob{Index: i + 1, Total: len(actions), Action: action}:
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

func (s Service) runAction(ctx context.Context, job actionJob, req Request, outputMu *stdsync.Mutex) error {
	action := job.Action
	switch action.Kind {
	case ActionSkip:
		if req.Progress && s.Stdout != nil {
			outputMu.Lock()
			fmt.Fprintf(s.Stdout, "skip [%d/%d]: %s (%s)\n", job.Index, job.Total, action.Local.Key, action.Reason)
			outputMu.Unlock()
		}
	case ActionUpload:
		if req.Progress && s.Stdout != nil {
			outputMu.Lock()
			fmt.Fprintf(s.Stdout, "upload [%d/%d]: %s (%s)\n", job.Index, job.Total, action.Local.Key, action.Reason)
			outputMu.Unlock()
		}
		if err := s.Uploader.Upload(ctx, upload.UploadInput(action.Local)); err != nil {
			return fmt.Errorf("upload %s: %w", action.Local.LocalPath, err)
		}
	case ActionDelete:
		if req.Progress && s.Stdout != nil {
			outputMu.Lock()
			fmt.Fprintf(s.Stdout, "delete [%d/%d]: %s (%s)\n", job.Index, job.Total, action.Remote.Key, action.Reason)
			outputMu.Unlock()
		}
		if err := s.Deleter.Delete(ctx, DeleteInput{
			Bucket: req.Destination.Bucket,
			Key:    action.Remote.Key,
		}); err != nil {
			return fmt.Errorf("delete %s: %w", action.Remote.Key, err)
		}
	}
	return nil
}

func printDryRunAction(stdout io.Writer, index, total int, action Action) {
	if stdout == nil {
		return
	}
	switch action.Kind {
	case ActionSkip:
		fmt.Fprintf(stdout, "skip [%d/%d]: %s (%s)\n", index, total, action.Local.Key, action.Reason)
	case ActionUpload:
		fmt.Fprintf(stdout, "dry-run upload [%d/%d]: %s (%s)\n", index, total, action.Local.Key, action.Reason)
	case ActionDelete:
		fmt.Fprintf(stdout, "dry-run delete [%d/%d]: %s (%s)\n", index, total, action.Remote.Key, action.Reason)
	}
}

type actionJob struct {
	Index  int
	Total  int
	Action Action
}

type summary struct {
	Uploads int
	Skips   int
	Deletes int
}

func summarize(actions []Action) summary {
	var out summary
	for _, action := range actions {
		switch action.Kind {
		case ActionUpload:
			out.Uploads++
		case ActionSkip:
			out.Skips++
		case ActionDelete:
			out.Deletes++
		}
	}
	return out
}

func printSummary(stdout io.Writer, summary summary, dryRun bool) {
	if dryRun {
		fmt.Fprintf(stdout, "summary: planned uploads=%d planned deletes=%d skipped=%d\n", summary.Uploads, summary.Deletes, summary.Skips)
		return
	}
	fmt.Fprintf(stdout, "summary: uploaded=%d deleted=%d skipped=%d\n", summary.Uploads, summary.Deletes, summary.Skips)
}
