package main

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/theakshaypant/agenttask-paco/pkg/artifact"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, ws *artifact.Workspace)
		wantSkip   string
		wantErr    string
		wantReqNot string // substring the request output must NOT be (used for empty-string checks)
	}{
		{
			name: "no .pr.diff at all is skipped",
			setup: func(t *testing.T, ws *artifact.Workspace) {
				t.Helper()
			},
			wantSkip: "true",
		},
		{
			name: "empty .pr.diff is skipped",
			setup: func(t *testing.T, ws *artifact.Workspace) {
				t.Helper()
				assert.NilError(t, ws.Write(artifact.FileDiff, []byte("")))
			},
			wantSkip: "true",
		},
		{
			name: "explicit .paco-error is skipped and reason is recorded",
			setup: func(t *testing.T, ws *artifact.Workspace) {
				t.Helper()
				assert.NilError(t, ws.Write(artifact.FileError, []byte("no PR found\n")))
			},
			wantSkip: "true",
		},
		{
			name: "a real diff is not skipped",
			setup: func(t *testing.T, ws *artifact.Workspace) {
				t.Helper()
				assert.NilError(t, ws.Write(artifact.FileDiff, []byte("+ added line")))
			},
			wantSkip: "false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			ws := &artifact.Workspace{Dir: dir}
			tt.setup(t, ws)

			requestOut := filepath.Join(t.TempDir(), "request")
			skipOut := filepath.Join(t.TempDir(), "skip")

			err := run([]string{
				"--workspace", dir,
				"--request-out", requestOut,
				"--skip-out", skipOut,
			})
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)

			skip, err := os.ReadFile(skipOut) //nolint:gosec // test-local temp file path
			assert.NilError(t, err)
			assert.Equal(t, string(skip), tt.wantSkip)

			if tt.wantSkip == "true" {
				assert.Assert(t, ws.Exists(artifact.FileFailed))
				assert.Assert(t, ws.Exists(artifact.FileReview))
			} else {
				request, err := os.ReadFile(requestOut) //nolint:gosec // test-local temp file path
				assert.NilError(t, err)
				assert.Assert(t, is.Contains(string(request), "added line"))
			}
		})
	}
}

func TestRunRequiresOutputFlags(t *testing.T) {
	err := run([]string{"--workspace", t.TempDir()})
	assert.ErrorContains(t, err, "required")
}
