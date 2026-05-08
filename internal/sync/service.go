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
	Stdout   io.Writer
}

func (s Service) Sync(ctx context.Context, req Request) error {
	if s.Lister == nil {
		return fmt.Errorf("lister is not configured")
	}
	if !req.DryRun && s.Uploader == nil {
		return fmt.Errorf("uploader is not configured")
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

	actions, err := Plan(local, remote, req.Checksum)
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
		}
	}

	return nil
}
