package sync

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

func Plan(local []upload.PlannedUpload, remote []list.Entry, req Request) ([]Action, error) {
	if err := upload.ValidateFilters(req.Include, req.Exclude); err != nil {
		return nil, err
	}
	remoteByKey := make(map[string]list.Entry, len(remote))
	for _, entry := range remote {
		if entry.IsDir {
			continue
		}
		remoteByKey[entry.Key] = entry
	}

	actions := make([]Action, 0, len(local))
	localKeys := make(map[string]struct{}, len(local))
	for _, item := range local {
		localKeys[item.Key] = struct{}{}
		remoteEntry, ok := remoteByKey[item.Key]
		if !ok {
			actions = append(actions, Action{
				Kind:   ActionUpload,
				Reason: "missing remote object",
				Local:  item,
			})
			continue
		}

		if remoteEntry.Size != item.Size {
			actions = append(actions, Action{
				Kind:   ActionUpload,
				Reason: "size differs",
				Local:  item,
				Remote: remoteEntry,
			})
			continue
		}

		if req.Checksum {
			remoteMD5, ok := comparableETag(remoteEntry.ETag)
			if !ok {
				actions = append(actions, Action{
					Kind:   ActionUpload,
					Reason: "etag not comparable",
					Local:  item,
					Remote: remoteEntry,
				})
				continue
			}

			localMD5, err := fileMD5(upload.UploadInput(item))
			if err != nil {
				return nil, err
			}
			if localMD5 != remoteMD5 {
				actions = append(actions, Action{
					Kind:   ActionUpload,
					Reason: "checksum differs",
					Local:  item,
					Remote: remoteEntry,
				})
				continue
			}
		}

		actions = append(actions, Action{
			Kind:   ActionSkip,
			Reason: skipReason(req.Checksum),
			Local:  item,
			Remote: remoteEntry,
		})
	}

	if req.Delete {
		for _, entry := range remote {
			if entry.IsDir {
				continue
			}
			if _, ok := localKeys[entry.Key]; ok {
				continue
			}
			rel, inPrefix := strings.CutPrefix(entry.Key, req.Destination.Prefix)
			if !inPrefix || !upload.Selected(rel, req.Include, req.Exclude) {
				continue
			}
			actions = append(actions, Action{
				Kind:   ActionDelete,
				Reason: "missing local file",
				Remote: entry,
			})
		}
	}

	return actions, nil
}

func BuildLocalPlan(req Request) ([]upload.PlannedUpload, error) {
	return upload.Plan(upload.Request{
		Source: req.Source,
		Destination: upload.S3URI{
			Bucket: req.Destination.Bucket,
			Key:    req.Destination.Prefix,
		},
		Recursive:           true,
		FollowLinks:         req.FollowLinks,
		RejectSourceSymlink: true,
		Include:             req.Include,
		Exclude:             req.Exclude,
		Options:             req.Options,
	})
}

func skipReason(checksum bool) string {
	if checksum {
		return "same size and checksum"
	}
	return "same size"
}

func comparableETag(etag string) (string, bool) {
	clean := strings.Trim(strings.TrimSpace(etag), `"`)
	if len(clean) != md5.Size*2 || strings.Contains(clean, "-") {
		return "", false
	}
	for _, char := range clean {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return "", false
		}
	}
	return strings.ToLower(clean), true
}

func fileMD5(input upload.UploadInput) (string, error) {
	file, err := input.OpenSource()
	if err != nil {
		return "", fmt.Errorf("open for checksum: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("read for checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
