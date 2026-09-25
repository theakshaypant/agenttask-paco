// Package artifact defines the workspace file names shared with the upstream
// paco-cli (github.com/pipelines-as-code/paco-cli) "diff" and "post" steps.
//
// agenttask-paco does not vendor paco-cli's internal packages (they are
// unexported from the module), so the file names below are re-declared here
// to keep this package in lock-step with paco-cli's own
// internal/artifact/artifact.go. If paco-cli changes these names, this file
// must be updated to match.
package artifact

import (
	"os"
	"path/filepath"
)

// File names written/read by paco-cli's "diff" and "post" steps that
// agenttask-paco's "build-request" and "translate-result" steps also need to
// read or write, so the real paco-cli binary can be used unmodified for the
// diff and post steps of the pipeline.
const (
	FileDiff             = ".pr.diff"
	FileValidLines       = ".valid-lines.json"
	FileExistingInline   = ".existing-inline.json"
	FileExistingFeedback = ".existing-feedback.txt"
	FileHeadSHA          = ".head_sha"
	FileError            = ".paco-error"
	FileReview           = ".paco-review.json"
	FileMode             = ".paco-mode"
	FileFailed           = ".paco-failed"
	FileSecurityBlock    = ".paco-security-block"
	FileReviewRules      = ".tekton/ai/REVIEW.md"
	FileToolchains       = ".toolchain-versions"
)

// Workspace is a thin helper around a workspace directory shared between
// Tekton Task steps, mirroring paco-cli's own internal/artifact.Workspace.
type Workspace struct {
	Dir string
}

// Path returns the absolute path of a named artifact inside the workspace.
func (w *Workspace) Path(name string) string {
	return filepath.Join(w.Dir, name)
}

// Write writes an artifact file, creating parent directories as needed.
func (w *Workspace) Write(name string, data []byte) error {
	path := w.Path(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Read reads an artifact file's contents.
func (w *Workspace) Read(name string) ([]byte, error) {
	return os.ReadFile(w.Path(name))
}

// Exists reports whether an artifact file is present.
func (w *Workspace) Exists(name string) bool {
	_, err := os.Stat(w.Path(name))
	return err == nil
}

// ReadOptional reads an artifact file, returning an empty string when the
// file does not exist or cannot be read (many artifacts are best-effort).
func (w *Workspace) ReadOptional(name string) string {
	data, err := w.Read(name)
	if err != nil {
		return ""
	}
	return string(data)
}
