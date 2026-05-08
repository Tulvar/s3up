package delete

import (
	"context"

	"github.com/tulvar/s3up/internal/list"
)

type Request struct {
	Target        list.S3Prefix
	Recursive     bool
	DryRun        bool
	ConfirmDelete bool
	Include       []string
	Exclude       []string
	Progress      bool
	Workers       int
}

type DeleteInput struct {
	Bucket string
	Key    string
}

type ObjectDeleter interface {
	Delete(ctx context.Context, input DeleteInput) error
}
