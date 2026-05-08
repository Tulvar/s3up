package sync

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tulvar/s3up/internal/list"
	"github.com/tulvar/s3up/internal/upload"
)

func Plan(local []upload.PlannedUpload, remote []list.Entry, checksum bool) ([]Action, error) {
	remoteByKey := make(map[string]list.Entry, len(remote))
	for _, entry := range remote {
		if entry.IsDir {
			continue
		}
		remoteByKey[entry.Key] = entry
	}

	actions := make([]Action, 0, len(local))
	for _, item := range local {
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

		if checksum {
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

			localMD5, err := fileMD5(item.LocalPath)
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
			Reason: skipReason(checksum),
			Local:  item,
			Remote: remoteEntry,
		})
	}

	return actions, nil
}

func BuildLocalPlan(req Request) ([]upload.PlannedUpload, error) {
	info, err := os.Stat(req.Source)
	if err != nil {
		return nil, fmt.Errorf("stat source: %w", err)
	}

	return upload.Plan(upload.Request{
		Source: req.Source,
		Destination: upload.S3URI{
			Bucket: req.Destination.Bucket,
			Key:    req.Destination.Prefix,
		},
		Recursive: info.IsDir(),
		Options:   req.Options,
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

func fileMD5(path string) (string, error) {
	file, err := os.Open(path)
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
