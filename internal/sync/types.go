package sync

import (
	deletepkg "github.com/tulvar/s3up/internal/delete"
	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

type Request struct {
	Source        string
	Destination   list.S3Prefix
	DryRun        bool
	FollowLinks   bool
	Checksum      bool
	Delete        bool
	ConfirmDelete bool
	Include       []string
	Exclude       []string
	Options       upload.Options
	Progress      bool
	Workers       int
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

type DeleteInput = deletepkg.DeleteInput
type ObjectDeleter = deletepkg.ObjectDeleter
