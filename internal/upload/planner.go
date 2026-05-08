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
	info, err := statSource(req.Source, req.FollowLinks)
	if err != nil {
		return nil, fmt.Errorf("stat source: %w", err)
	}
	if info == nil {
		return nil, nil
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
	err = walkUploadDir(req.Source, req.Source, req, make(map[string]struct{}), &items)
	if err != nil {
		return nil, fmt.Errorf("walk source: %w", err)
	}

	return items, nil
}

func statSource(localPath string, followLinks bool) (fs.FileInfo, error) {
	if followLinks {
		return os.Stat(localPath)
	}
	info, err := os.Lstat(localPath)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}
	return info, nil
}

func entryInfo(localPath string, entry fs.DirEntry, followLinks bool) (fs.FileInfo, error) {
	if entry.Type()&os.ModeSymlink == 0 {
		return entry.Info()
	}
	if !followLinks {
		return nil, nil
	}
	return os.Stat(localPath)
}

func walkUploadDir(root, dir string, req Request, visited map[string]struct{}, items *[]PlannedUpload) error {
	realDir, err := filepath.EvalSymlinks(dir)
	if err == nil {
		if _, ok := visited[realDir]; ok {
			return nil
		}
		visited[realDir] = struct{}{}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		localPath := filepath.Join(dir, entry.Name())
		info, err := entryInfo(localPath, entry, req.FollowLinks)
		if err != nil {
			return err
		}
		if info == nil {
			continue
		}
		if info.IsDir() {
			if err := walkUploadDir(root, localPath, req, visited, items); err != nil {
				return err
			}
			continue
		}

		rel, err := filepath.Rel(root, localPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !Selected(rel, req.Include, req.Exclude) {
			continue
		}

		*items = append(*items, PlannedUpload{
			LocalPath:    localPath,
			Bucket:       req.Destination.Bucket,
			Key:          joinS3Key(req.Destination.Key, rel),
			Size:         info.Size(),
			ContentType:  contentTypeFor(localPath, req.Options.ContentType),
			Metadata:     cloneMetadata(req.Options.Metadata),
			StorageClass: req.Options.StorageClass,
			ACL:          req.Options.ACL,
		})
	}

	return nil
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
