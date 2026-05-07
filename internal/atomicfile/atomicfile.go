// Copyright (c) Greetingland LLC

// Package atomicfile provides atomic file write helpers shared by apitools
// subpackages. Writes go to a sibling temp file and are renamed into place
// so concurrent readers never observe a partial file.
package atomicfile

import (
	"os"
	"path/filepath"
)

// Write writes data to path atomically with the given permission. The data
// is first written to a temp file in the same directory, then renamed.
func Write(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
