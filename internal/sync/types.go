package sync

import (
	"context"

	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

type Request struct {
	Source        string
	Destination   list.S3Prefix
	DryRun        bool
	Checksum      bool
	Delete        bool
	ConfirmDelete bool
	Include       []string
	Exclude       []string
	Options       upload.Options
	Progress      bool
}

type ActionKind string

const (
	ActionUpload ActionKind = "upload"
	ActionSkip   ActionKind = "skip"
	ActionDelete ActionKind = "delete"
)

type Action struct {
	Kind   ActionKind
	Reason string
	Local  upload.PlannedUpload
	Remote list.Entry
}

type DeleteInput struct {
	Bucket string
	Key    string
}

type ObjectDeleter interface {
	Delete(ctx context.Context, input DeleteInput) error
}
