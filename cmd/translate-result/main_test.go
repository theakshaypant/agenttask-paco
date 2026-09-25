package main

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/theakshaypant/agenttask-paco/pkg/artifact"
)

func TestRunSkipIsNoop(t *testing.T) {
	dir := t.TempDir()
	err := run(context.Background(), []string{"--workspace", dir, "--skip"})
	assert.NilError(t, err)

	ws := &artifact.Workspace{Dir: dir}
	assert.Assert(t, !ws.Exists(artifact.FileReview), "translate-result should not write anything when --skip is set")
}

func TestRunRequiresFlagsUnlessSkipped(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no flags at all", args: []string{"--workspace", t.TempDir()}},
		{name: "missing analysis-result-name", args: []string{"--workspace", t.TempDir(), "--namespace", "ns", "--outcome", "no-action-required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), tt.args)
			assert.ErrorContains(t, err, "required")
		})
	}
}
