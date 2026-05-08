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

	for i, item := range planned {
		index := i + 1
		total := len(planned)
		if req.DryRun {
			if s.Stdout != nil {
				fmt.Fprintf(s.Stdout, "dry-run [%d/%d]: %s -> s3://%s/%s (%s)\n", index, total, item.LocalPath, item.Bucket, item.Key, formatSize(item.Size))
			}
			continue
		}
		if s.Uploader == nil {
			return fmt.Errorf("uploader is not configured")
		}
		if req.Progress && s.Stdout != nil {
			fmt.Fprintf(s.Stdout, "uploading [%d/%d]: %s -> s3://%s/%s (%s)\n", index, total, item.LocalPath, item.Bucket, item.Key, formatSize(item.Size))
		}
		if err := s.Uploader.Upload(ctx, UploadInput(item)); err != nil {
			return fmt.Errorf("upload %s: %w", item.LocalPath, err)
		}
		if req.Progress && s.Stdout != nil {
			fmt.Fprintf(s.Stdout, "uploaded [%d/%d]: %s -> s3://%s/%s\n", index, total, item.LocalPath, item.Bucket, item.Key)
		}
	}

	return nil
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
