package sync

import (
	"context"
	"fmt"
	"io"

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
	if req.Delete && !req.DryRun && s.Deleter == nil {
		return fmt.Errorf("deleter is not configured")
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
	for i, action := range actions {
		index := i + 1
		switch action.Kind {
		case ActionSkip:
			if req.Progress && s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "skip [%d/%d]: %s (%s)\n", index, total, action.Local.Key, action.Reason)
			}
		case ActionUpload:
			if req.DryRun {
				if s.Stdout != nil {
					fmt.Fprintf(s.Stdout, "dry-run upload [%d/%d]: %s (%s)\n", index, total, action.Local.Key, action.Reason)
				}
				continue
			}
			if req.Progress && s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "upload [%d/%d]: %s (%s)\n", index, total, action.Local.Key, action.Reason)
			}
			if err := s.Uploader.Upload(ctx, upload.UploadInput(action.Local)); err != nil {
				return fmt.Errorf("upload %s: %w", action.Local.LocalPath, err)
			}
		case ActionDelete:
			if req.DryRun {
				if s.Stdout != nil {
					fmt.Fprintf(s.Stdout, "dry-run delete [%d/%d]: %s (%s)\n", index, total, action.Remote.Key, action.Reason)
				}
				continue
			}
			if req.Progress && s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "delete [%d/%d]: %s (%s)\n", index, total, action.Remote.Key, action.Reason)
			}
			if err := s.Deleter.Delete(ctx, DeleteInput{
				Bucket: req.Destination.Bucket,
				Key:    action.Remote.Key,
			}); err != nil {
				return fmt.Errorf("delete %s: %w", action.Remote.Key, err)
			}
		}
	}

	return nil
}
