// Copyright (c) Greetingland LLC

// Package icot provides on-disk persistence helpers for interactive
// chain-of-thought (icot) sessions.
//
// The conversation orchestration itself lives in the parent `apitools`
// package as `RunProgressiveICOT[S, D, A]` and `ProgressiveLoopHooks[S, D, A]`,
// which already implement the bound-runtime pattern: a generic structural
// loop bound at execution time to a concrete engine's domain hooks. This
// package only contributes draft-on-disk plumbing that both engines reuse.
package icot

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/OpenUdon/apitools/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

// DraftPath returns the canonical `.icot/session.yaml` path under exampleDir.
// Engines that want a different on-disk layout can ignore this helper and
// pass their own path to LoadDraft / SaveDraft.
func DraftPath(exampleDir string) string {
	return filepath.Join(exampleDir, ".icot", "session.yaml")
}

// LoadDraft reads a YAML-encoded session of type T from path. It returns
// (zero, false, nil) when the file does not exist, (decoded, true, nil) on
// success, and propagates other errors.
func LoadDraft[T any](path string) (T, bool, error) {
	var zero T
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var session T
	if err := yaml.Unmarshal(data, &session); err != nil {
		return zero, false, err
	}
	return session, true, nil
}

// SaveDraft writes session as YAML to path atomically. The parent directory
// is created with 0o755 if it does not exist; the file is written 0o600.
// A no-op when path is empty.
func SaveDraft[T any](path string, session T) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(session)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 0o600)
}

// DeleteDraft removes the draft file and prunes the enclosing `.icot/`
// directory if it becomes empty. Missing files are not an error.
func DeleteDraft(path string) error {
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return err
	}
	draftDir := filepath.Dir(path)
	_ = os.Remove(draftDir)
	return nil
}
