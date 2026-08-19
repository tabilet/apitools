// Package artifactio provides confined, integrity-checked artifact reads and
// atomic writes for apitools' local cache and materialization paths.
package artifactio

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultMaxBytes          = 128 * 1024 * 1024
	defaultMaxCompareEntries = 10_000
	defaultMaxCompareBytes   = 512 * 1024 * 1024
)

var ErrCollision = errors.New("artifact destination collision")

// EnsureRoot creates a local artifact root without following symlinked path
// components and returns its absolute path.
func EnsureRoot(root string, mode fs.FileMode) (string, error) {
	if mode.Perm() == 0 {
		mode = 0o755
	}
	return ensureDirectory(root, mode.Perm())
}

// CollisionError reports that an existing destination differs from the
// proposed content and replacement was not explicitly enabled.
type CollisionError struct {
	Path           string
	ExistingSHA256 string
	IncomingSHA256 string
}

func (err *CollisionError) Error() string {
	return fmt.Sprintf("%v at %q: existing SHA256 %s differs from incoming SHA256 %s", ErrCollision, err.Path, err.ExistingSHA256, err.IncomingSHA256)
}

func (err *CollisionError) Unwrap() error { return ErrCollision }

// ReadOptions controls a confined artifact read.
type ReadOptions struct {
	MaxBytes int64
	SHA256   string
	Bytes    int64
}

// File records content and verified integrity metadata.
type File struct {
	Path   string
	Data   []byte
	SHA256 string
	Bytes  int64
}

// ReadFile reads a local relative path under root without following symlinks.
func ReadFile(root, relative string, opts ReadOptions) (File, error) {
	rootAbs, err := existingDirectory(root)
	if err != nil {
		return File{}, err
	}
	relative, err = cleanRelative(relative)
	if err != nil {
		return File{}, err
	}
	path := filepath.Join(rootAbs, relative)
	if err := verifyExistingParents(rootAbs, filepath.Dir(path)); err != nil {
		return File{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return File{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return File{}, fmt.Errorf("artifact path %q is a symlink and not a regular file", path)
	}
	if !info.Mode().IsRegular() {
		return File{}, fmt.Errorf("artifact path %q is not a regular file", path)
	}
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if info.Size() > maxBytes {
		return File{}, fmt.Errorf("artifact %q is %d bytes, over limit %d", path, info.Size(), maxBytes)
	}
	if opts.Bytes > 0 && info.Size() != opts.Bytes {
		return File{}, fmt.Errorf("artifact %q has %d bytes, want %d", path, info.Size(), opts.Bytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return File{}, err
	}
	openedInfo, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return File{}, statErr
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return File{}, fmt.Errorf("artifact path %q changed during open", path)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return File{}, readErr
	}
	if closeErr != nil {
		return File{}, closeErr
	}
	if int64(len(data)) > maxBytes {
		return File{}, fmt.Errorf("artifact %q is over limit %d", path, maxBytes)
	}
	digest := digestBytes(data)
	if expected := strings.ToLower(strings.TrimSpace(opts.SHA256)); expected != "" {
		if !validDigest(expected) {
			return File{}, fmt.Errorf("artifact %q has invalid expected SHA256 %q", path, opts.SHA256)
		}
		if digest != expected {
			return File{}, fmt.Errorf("artifact %q SHA256 is %s, want %s", path, digest, expected)
		}
	}
	return File{Path: path, Data: data, SHA256: digest, Bytes: int64(len(data))}, nil
}

// WriteOptions controls one confined atomic file write.
type WriteOptions struct {
	Mode  fs.FileMode
	Force bool
}

// WriteResult records an atomic file write or identical-content reuse.
type WriteResult struct {
	Path   string
	SHA256 string
	Bytes  int64
	Reused bool
}

// WriteFile atomically writes relative under root. Existing identical content
// is reused; differing content requires Force.
func WriteFile(root, relative string, data []byte, opts WriteOptions) (WriteResult, error) {
	rootAbs, err := ensureDirectory(root, 0o755)
	if err != nil {
		return WriteResult{}, err
	}
	relative, err = cleanRelative(relative)
	if err != nil {
		return WriteResult{}, err
	}
	path := filepath.Join(rootAbs, relative)
	if _, err := ensureDirectory(filepath.Dir(path), 0o755); err != nil {
		return WriteResult{}, err
	}
	digest := digestBytes(data)
	result := WriteResult{Path: path, SHA256: digest, Bytes: int64(len(data))}
	reused, err := checkFileCollision(rootAbs, relative, data, digest, opts.Force)
	if err != nil {
		return WriteResult{}, err
	}
	if reused {
		result.Reused = true
		return result, nil
	}
	mode := opts.Mode.Perm()
	if mode == 0 {
		mode = 0o600
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".stage-*")
	if err != nil {
		return WriteResult{}, err
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return WriteResult{}, err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return WriteResult{}, err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return WriteResult{}, err
	}
	if err := temp.Close(); err != nil {
		return WriteResult{}, err
	}
	if !opts.Force {
		if err := os.Link(tempPath, path); err != nil {
			reused, collisionErr := checkFileCollision(rootAbs, relative, data, digest, false)
			if collisionErr != nil {
				return WriteResult{}, collisionErr
			}
			if !reused {
				return WriteResult{}, err
			}
			result.Reused = true
			return result, nil
		}
		if err := os.Remove(tempPath); err != nil {
			rollbackErr := os.Remove(path)
			if rollbackErr != nil {
				return WriteResult{}, errors.Join(err, fmt.Errorf("rollback linked artifact: %w", rollbackErr))
			}
			return WriteResult{}, err
		}
		committed = true
		if err := syncDirectory(filepath.Dir(path)); err != nil {
			_ = os.Remove(path)
			return WriteResult{}, err
		}
		return result, nil
	}
	if err := replaceFileWithRollback(tempPath, path); err != nil {
		return WriteResult{}, err
	}
	committed = true
	return result, nil
}

func replaceFileWithRollback(tempPath, path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.Rename(tempPath, path); err != nil {
			return err
		}
		if err := syncDirectory(filepath.Dir(path)); err != nil {
			removeErr := os.Remove(path)
			if removeErr != nil {
				return errors.Join(err, fmt.Errorf("rollback new artifact: %w", removeErr))
			}
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("artifact destination %q is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("artifact destination %q is not a regular file", path)
	}
	backup, err := unusedSiblingPath(path, "backup")
	if err != nil {
		return err
	}
	if err := os.Rename(path, backup); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		rollbackErr := os.Rename(backup, path)
		if rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("restore artifact backup: %w", rollbackErr))
		}
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return rollbackFileReplacement(path, backup, err)
	}
	if err := os.Remove(backup); err != nil {
		return rollbackFileReplacement(path, backup, err)
	}
	return nil
}

func rollbackFileReplacement(path, backup string, cause error) error {
	removeErr := os.Remove(path)
	if removeErr != nil {
		return errors.Join(cause, fmt.Errorf("remove failed artifact replacement: %w", removeErr))
	}
	rollbackErr := os.Rename(backup, path)
	if rollbackErr != nil {
		return errors.Join(cause, fmt.Errorf("restore artifact backup: %w", rollbackErr))
	}
	return cause
}

func checkFileCollision(root, relative string, data []byte, digest string, force bool) (bool, error) {
	path := filepath.Join(root, relative)
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("artifact destination %q is a symlink and not a regular file", path)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("artifact destination %q is not a regular file", path)
	}
	if info.Size() == int64(len(data)) {
		existing, err := ReadFile(root, relative, ReadOptions{MaxBytes: int64(len(data))})
		if err != nil {
			return false, err
		}
		if existing.SHA256 == digest {
			return true, nil
		}
		if !force {
			return false, &CollisionError{Path: path, ExistingSHA256: existing.SHA256, IncomingSHA256: digest}
		}
		return false, nil
	}
	if !force {
		return false, &CollisionError{Path: path, ExistingSHA256: "different-size", IncomingSHA256: digest}
	}
	return false, nil
}

// DirTransaction stages a complete directory tree beside its destination and
// publishes it with one rename. Rollback removes only the private stage.
type DirTransaction struct {
	target string
	stage  string
	force  bool
	done   bool
}

// BeginDir starts a directory transaction. The target itself must not be a
// filesystem root or symlink.
func BeginDir(target string, force bool) (*DirTransaction, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("artifact transaction target is required")
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	targetAbs = filepath.Clean(targetAbs)
	if isFilesystemRoot(targetAbs) {
		return nil, fmt.Errorf("artifact transaction target must not be a filesystem root")
	}
	parent, err := ensureDirectory(filepath.Dir(targetAbs), 0o755)
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkTarget(targetAbs); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(targetAbs); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("artifact transaction target %q is not a directory", targetAbs)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	stage, err := os.MkdirTemp(parent, "."+filepath.Base(targetAbs)+".stage-*")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(stage, 0o700); err != nil {
		_ = os.RemoveAll(stage)
		return nil, err
	}
	return &DirTransaction{target: targetAbs, stage: stage, force: force}, nil
}

// Stage writes one relative file into the private transaction tree.
func (tx *DirTransaction) Stage(relative string, data []byte, mode fs.FileMode) (WriteResult, error) {
	if tx == nil || tx.done {
		return WriteResult{}, fmt.Errorf("artifact transaction is closed")
	}
	return WriteFile(tx.stage, relative, data, WriteOptions{Mode: mode})
}

// TargetPath returns the final confined path for a staged relative file.
func (tx *DirTransaction) TargetPath(relative string) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("artifact transaction is nil")
	}
	relative, err := cleanRelative(relative)
	if err != nil {
		return "", err
	}
	return filepath.Join(tx.target, relative), nil
}

// Commit publishes the complete staged tree. An identical destination is
// reused. A differing destination requires Force and is backed up until the
// staged rename succeeds.
func (tx *DirTransaction) Commit() (bool, error) {
	if tx == nil || tx.done {
		return false, fmt.Errorf("artifact transaction is closed")
	}
	defer func() { tx.done = true }()
	info, err := os.Lstat(tx.target)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.Rename(tx.stage, tx.target); err != nil {
			_ = os.RemoveAll(tx.stage)
			return false, err
		}
		if err := syncDirectory(filepath.Dir(tx.target)); err != nil {
			removeErr := os.RemoveAll(tx.target)
			if removeErr != nil {
				return false, errors.Join(err, fmt.Errorf("rollback new artifact directory: %w", removeErr))
			}
			return false, err
		}
		return false, nil
	}
	if err != nil {
		_ = os.RemoveAll(tx.stage)
		return false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		_ = os.RemoveAll(tx.stage)
		return false, fmt.Errorf("artifact transaction target %q is not a regular directory", tx.target)
	}
	equal, err := equalTrees(tx.target, tx.stage)
	if err != nil {
		_ = os.RemoveAll(tx.stage)
		return false, err
	}
	if equal {
		if err := os.RemoveAll(tx.stage); err != nil {
			return false, err
		}
		return true, nil
	}
	if !tx.force {
		_ = os.RemoveAll(tx.stage)
		return false, fmt.Errorf("%w at directory %q", ErrCollision, tx.target)
	}
	backup, err := unusedSiblingPath(tx.target, "backup")
	if err != nil {
		_ = os.RemoveAll(tx.stage)
		return false, err
	}
	if err := os.Rename(tx.target, backup); err != nil {
		_ = os.RemoveAll(tx.stage)
		return false, err
	}
	if err := os.Rename(tx.stage, tx.target); err != nil {
		rollbackErr := os.Rename(backup, tx.target)
		_ = os.RemoveAll(tx.stage)
		if rollbackErr != nil {
			return false, errors.Join(err, fmt.Errorf("restore artifact backup: %w", rollbackErr))
		}
		return false, err
	}
	if err := syncDirectory(filepath.Dir(tx.target)); err != nil {
		removeErr := os.RemoveAll(tx.target)
		if removeErr != nil {
			return false, errors.Join(err, fmt.Errorf("remove failed artifact replacement: %w", removeErr))
		}
		rollbackErr := os.Rename(backup, tx.target)
		if rollbackErr != nil {
			return false, errors.Join(err, fmt.Errorf("restore artifact backup: %w", rollbackErr))
		}
		return false, err
	}
	if err := os.RemoveAll(backup); err != nil {
		return false, err
	}
	return false, nil
}

// Rollback discards the private stage and never mutates the destination.
func (tx *DirTransaction) Rollback() error {
	if tx == nil || tx.done {
		return nil
	}
	tx.done = true
	return os.RemoveAll(tx.stage)
}

func equalTrees(left, right string) (bool, error) {
	leftEntries, err := treeEntries(left)
	if err != nil {
		return false, err
	}
	rightEntries, err := treeEntries(right)
	if err != nil {
		return false, err
	}
	if len(leftEntries) != len(rightEntries) {
		return false, nil
	}
	keys := make([]string, 0, len(leftEntries))
	for key := range leftEntries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if leftEntries[key] != rightEntries[key] {
			return false, nil
		}
	}
	return true, nil
}

func treeEntries(root string) (map[string]string, error) {
	entries := map[string]string{}
	var totalBytes int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if len(entries) >= defaultMaxCompareEntries {
			return fmt.Errorf("artifact tree %q exceeds %d entries", root, defaultMaxCompareEntries)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact tree %q contains symlink %q", root, relative)
		}
		if entry.IsDir() {
			entries[relative] = "dir"
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("artifact tree %q contains non-regular path %q", root, relative)
		}
		totalBytes += info.Size()
		if totalBytes > defaultMaxCompareBytes {
			return fmt.Errorf("artifact tree %q exceeds %d comparison bytes", root, defaultMaxCompareBytes)
		}
		file, err := ReadFile(root, relative, ReadOptions{MaxBytes: info.Size() + 1})
		if err != nil {
			return err
		}
		entries[relative] = "file:" + file.SHA256
		return nil
	})
	return entries, err
}

func cleanRelative(relative string) (string, error) {
	relative = filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if relative == "." || !filepath.IsLocal(relative) {
		return "", fmt.Errorf("artifact path %q must be a local relative path", relative)
	}
	return relative, nil
}

func existingDirectory(path string) (string, error) {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if isFilesystemRoot(abs) {
		return "", fmt.Errorf("artifact root must not be a filesystem root")
	}
	if err := verifyDirectoryChain(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func ensureDirectory(path string, mode fs.FileMode) (string, error) {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if isFilesystemRoot(abs) {
		return "", fmt.Errorf("artifact root must not be a filesystem root")
	}
	volume := filepath.VolumeName(abs)
	current := volume + string(filepath.Separator)
	remainder := strings.TrimPrefix(abs, current)
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.Mkdir(current, mode); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("artifact directory component %q is not a regular directory", current)
		}
	}
	return abs, nil
}

func verifyDirectoryChain(path string) error {
	volume := filepath.VolumeName(path)
	current := volume + string(filepath.Separator)
	remainder := strings.TrimPrefix(path, current)
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact directory component %q is not a regular directory", current)
		}
	}
	return nil
}

func verifyExistingParents(root, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	relative, err := filepath.Rel(root, path)
	if err != nil || (relative != "." && !filepath.IsLocal(relative)) {
		return fmt.Errorf("artifact path %q escapes root %q", path, root)
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact directory component %q is not a regular directory", current)
		}
	}
	return nil
}

func rejectSymlinkTarget(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("artifact destination %q is a symlink", path)
	}
	return nil
}

func unusedSiblingPath(target, label string) (string, error) {
	dir, err := os.MkdirTemp(filepath.Dir(target), "."+filepath.Base(target)+"."+label+"-*")
	if err != nil {
		return "", err
	}
	if err := os.Remove(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr := dir.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func isFilesystemRoot(path string) bool {
	volume := filepath.VolumeName(path)
	return filepath.Clean(path) == filepath.Clean(volume+string(filepath.Separator))
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
