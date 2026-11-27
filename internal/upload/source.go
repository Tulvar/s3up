package upload

import (
	"fmt"
	"os"
	"path/filepath"
)

type localSource struct {
	rootPath     string
	rootInfo     os.FileInfo
	relativePath string
	fileInfo     os.FileInfo
	followLinks  bool
}

// OpenSource opens the local file for either upload or checksum calculation.
func (input UploadInput) OpenSource() (*os.File, error) {
	source := input.source
	if source.followLinks {
		return os.Open(input.LocalPath)
	}

	rootPath, name := source.rootPath, source.relativePath
	if rootPath == "" {
		rootPath, name = filepath.Split(input.LocalPath)
		if rootPath == "" {
			rootPath = "."
		}
	}
	root, err := openVerifiedRoot(rootPath, source.rootInfo)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	// The root confines path resolution. Checking the opened descriptor binds
	// the read to the file inspected before opening it.
	expected, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !expected.Mode().IsRegular() || (source.fileInfo != nil && !os.SameFile(expected, source.fileInfo)) {
		return nil, fmt.Errorf("source is no longer the planned regular file: %q", input.LocalPath)
	}

	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	actual, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(expected, actual) {
		file.Close()
		return nil, fmt.Errorf("source changed while opening: %q", input.LocalPath)
	}
	return file, nil
}

func openVerifiedRoot(path string, expected os.FileInfo) (*os.Root, error) {
	root, err := os.OpenRoot(path)
	if err != nil || expected == nil {
		return root, err
	}
	if err := verifyRoot(root, expected); err != nil {
		root.Close()
		return nil, err
	}
	return root, nil
}

func verifyRoot(root *os.Root, expected os.FileInfo) error {
	actual, err := root.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(expected, actual) {
		return fmt.Errorf("source directory changed: %q", root.Name())
	}
	return nil
}
