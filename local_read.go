package apitools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func resolvedLocalMaxBytes(maxBytes int64) int64 {
	if maxBytes <= 0 {
		return DefaultMaxBytes
	}
	return maxBytes
}

func validateLocalScanRoot(path string) error {
	info, err := lstatLocalPathNoSymlinks(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("local OpenAPI directory %q is not a directory", path)
	}
	return nil
}

func readLocalSpecFile(path string, maxBytes int64) ([]byte, error) {
	maxBytes = resolvedLocalMaxBytes(maxBytes)
	info, err := lstatLocalPathNoSymlinks(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("local OpenAPI document %q is a directory", path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("local OpenAPI document %q is not a regular file", path)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("local OpenAPI document %q is larger than %d bytes", path, maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, openedInfo) {
		return nil, fmt.Errorf("local OpenAPI document %q changed while opening", path)
	}
	if openedInfo.IsDir() {
		return nil, fmt.Errorf("local OpenAPI document %q is a directory", path)
	}
	if !openedInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("local OpenAPI document %q is not a regular file", path)
	}
	if openedInfo.Size() > maxBytes {
		return nil, fmt.Errorf("local OpenAPI document %q is larger than %d bytes", path, maxBytes)
	}
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("local OpenAPI document %q is larger than %d bytes", path, maxBytes)
	}
	return content, nil
}

func lstatLocalPathNoSymlinks(path string) (os.FileInfo, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("local OpenAPI path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(abs)
	root := string(filepath.Separator)
	if volume := filepath.VolumeName(clean); volume != "" {
		root = volume + string(filepath.Separator)
	}
	rel, err := filepath.Rel(root, clean)
	if err != nil {
		return nil, err
	}
	current := root
	var info os.FileInfo
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err = os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("local OpenAPI path %q contains symlink component %q", path, current)
		}
	}
	if info == nil {
		info, err = os.Lstat(clean)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("local OpenAPI path %q contains symlink component %q", path, clean)
		}
	}
	return info, nil
}
