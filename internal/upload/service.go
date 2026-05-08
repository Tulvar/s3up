package upload

import (
	"context"
	"fmt"
	"io"
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

	for _, item := range planned {
		if req.DryRun {
			if s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "dry-run: %s -> s3://%s/%s\n", item.LocalPath, item.Bucket, item.Key)
			}
			continue
		}
		if s.Uploader == nil {
			return fmt.Errorf("uploader is not configured")
		}
		if err := s.Uploader.Upload(ctx, UploadInput(item)); err != nil {
			return fmt.Errorf("upload %s: %w", item.LocalPath, err)
		}
		if s.Stdout != nil {
			fmt.Fprintf(s.Stdout, "uploaded: %s -> s3://%s/%s\n", item.LocalPath, item.Bucket, item.Key)
		}
	}

	return nil
}
