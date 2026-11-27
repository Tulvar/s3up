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
	info, err := statSource(req)
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
			source:       localSource{fileInfo: info, followLinks: req.FollowLinks},
		}}, nil
	}

	if !req.Recursive {
		return nil, fmt.Errorf("source is a directory; use --recursive")
	}

	var root *os.Root
	if !req.FollowLinks {
		root, err = openVerifiedRoot(req.Source, info)
		if err != nil {
			return nil, fmt.Errorf("open source directory: %w", err)
		}
		defer root.Close()
	}

	var items []PlannedUpload
	err = walkUploadDir(".", req, root, info, make(map[string]struct{}), &items)
	if err != nil {
		return nil, fmt.Errorf("walk source: %w", err)
	}

	return items, nil
}

func statSource(req Request) (fs.FileInfo, error) {
	if req.FollowLinks {
		return os.Stat(req.Source)
	}
	// A trailing separator makes Lstat follow a symlink to a directory.
	localPath := strings.TrimRightFunc(req.Source, func(char rune) bool {
		return char <= 127 && os.IsPathSeparator(byte(char))
	})
	if len(localPath) == len(filepath.VolumeName(req.Source)) {
		localPath = req.Source
	}
	info, err := os.Lstat(localPath)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if req.RejectSourceSymlink {
			return nil, fmt.Errorf("source is a symlink; use --follow-symlinks to follow it")
		}
		return nil, nil
	}
	return info, nil
}

func followedEntryInfo(localPath string, entry fs.DirEntry) (fs.FileInfo, error) {
	if entry.Type()&os.ModeSymlink == 0 {
		return entry.Info()
	}
	return os.Stat(localPath)
}

func walkUploadDir(dir string, req Request, root *os.Root, rootInfo fs.FileInfo, visited map[string]struct{}, items *[]PlannedUpload) error {
	var entries []fs.DirEntry
	var err error
	if root != nil {
		entries, err = fs.ReadDir(root.FS(), ".")
	} else {
		localDir := filepath.Join(req.Source, dir)
		if realDir, evalErr := filepath.EvalSymlinks(localDir); evalErr == nil {
			if _, ok := visited[realDir]; ok {
				return nil
			}
			visited[realDir] = struct{}{}
		}
		entries, err = os.ReadDir(localDir)
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		rel := filepath.Join(dir, entry.Name())
		localPath := filepath.Join(req.Source, rel)
		var info fs.FileInfo
		if root != nil {
			info, err = root.Lstat(entry.Name())
		} else {
			info, err = followedEntryInfo(localPath, entry)
		}
		if err != nil {
			return err
		}
		if root != nil && info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			var child *os.Root
			if root != nil {
				child, err = root.OpenRoot(entry.Name())
				if err != nil {
					return err
				}
				if err := verifyRoot(child, info); err != nil {
					child.Close()
					return err
				}
			}
			err = walkUploadDir(rel, req, child, rootInfo, visited, items)
			if child != nil {
				child.Close()
			}
			if err != nil {
				return err
			}
			continue
		}

		keySuffix := filepath.ToSlash(rel)
		if !Selected(keySuffix, req.Include, req.Exclude) {
			continue
		}

		*items = append(*items, PlannedUpload{
			LocalPath:    localPath,
			Bucket:       req.Destination.Bucket,
			Key:          joinS3Key(req.Destination.Key, keySuffix),
			Size:         info.Size(),
			ContentType:  contentTypeFor(localPath, req.Options.ContentType),
			Metadata:     cloneMetadata(req.Options.Metadata),
			StorageClass: req.Options.StorageClass,
			ACL:          req.Options.ACL,
			source:       localSource{rootPath: req.Source, rootInfo: rootInfo, relativePath: rel, fileInfo: info, followLinks: req.FollowLinks},
		})
	}

	return nil
}

func destinationKeyForFile(destKey, filename string) string {
	if destKey == "" || strings.HasSuffix(destKey, "/") {
		return destKey + filename
	}
	return destKey
}

func joinS3Key(prefix, name string) string {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + name
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
