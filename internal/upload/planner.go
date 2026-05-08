package upload

import (
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Plan(req Request) ([]PlannedUpload, error) {
	info, err := os.Stat(req.Source)
	if err != nil {
		return nil, fmt.Errorf("stat source: %w", err)
	}

	if !info.IsDir() {
		if !Selected(filepath.Base(req.Source), req.Include, req.Exclude) {
			return nil, nil
		}
		key := destinationKeyForFile(req.Destination.Key, filepath.Base(req.Source))
		return []PlannedUpload{{
			LocalPath:    req.Source,
			Bucket:       req.Destination.Bucket,
			Key:          key,
			Size:         info.Size(),
			ContentType:  contentTypeFor(req.Source, req.Options.ContentType),
			Metadata:     cloneMetadata(req.Options.Metadata),
			StorageClass: req.Options.StorageClass,
			ACL:          req.Options.ACL,
		}}, nil
	}

	if !req.Recursive {
		return nil, fmt.Errorf("source is a directory; use --recursive")
	}

	var items []PlannedUpload
	err = filepath.WalkDir(req.Source, func(localPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(req.Source, localPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !Selected(rel, req.Include, req.Exclude) {
			return nil
		}

		items = append(items, PlannedUpload{
			LocalPath:    localPath,
			Bucket:       req.Destination.Bucket,
			Key:          joinS3Key(req.Destination.Key, rel),
			Size:         info.Size(),
			ContentType:  contentTypeFor(localPath, req.Options.ContentType),
			Metadata:     cloneMetadata(req.Options.Metadata),
			StorageClass: req.Options.StorageClass,
			ACL:          req.Options.ACL,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk source: %w", err)
	}

	return items, nil
}

func destinationKeyForFile(destKey, filename string) string {
	if destKey == "" || strings.HasSuffix(destKey, "/") {
		return joinS3Key(destKey, filename)
	}
	return cleanS3Key(destKey)
}

func joinS3Key(prefix, name string) string {
	prefix = strings.Trim(prefix, "/")
	name = strings.TrimLeft(name, "/")
	if prefix == "" {
		return cleanS3Key(name)
	}
	return cleanS3Key(path.Join(prefix, name))
}

func cleanS3Key(key string) string {
	return strings.TrimLeft(path.Clean("/"+filepath.ToSlash(key)), "/")
}

func contentTypeFor(localPath, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return mime.TypeByExtension(filepath.Ext(localPath))
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func Selected(rel string, include, exclude []string) bool {
	if len(include) > 0 && !matchesAny(rel, include) {
		return false
	}
	return !matchesAny(rel, exclude)
}

func RelativeKey(key, prefix string) string {
	key = strings.TrimLeft(path.Clean("/"+key), "/")
	prefix = strings.TrimLeft(path.Clean("/"+prefix), "/")
	if prefix == "." {
		prefix = ""
	}
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.TrimPrefix(key, prefix)
}

func matchesAny(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	base := path.Base(rel)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}

		target := rel
		if !strings.Contains(pattern, "/") {
			target = base
		}
		matched, err := path.Match(pattern, target)
		if err == nil && matched {
			return true
		}
	}
	return false
}
