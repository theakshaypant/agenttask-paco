package review

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/theakshaypant/agenttask-paco/pkg/artifact"
)

func TestWriteFailure(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{name: "simple reason", reason: "No reviewable changes found in this diff."},
		{name: "empty reason still writes valid artifacts", reason: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &artifact.Workspace{Dir: t.TempDir()}

			assert.NilError(t, WriteFailure(ws, tt.reason))

			assert.Assert(t, ws.Exists(artifact.FileFailed))

			data, err := ws.Read(artifact.FileReview)
			assert.NilError(t, err)

			var got Review
			assert.NilError(t, json.Unmarshal(data, &got))
			assert.Equal(t, got.Summary, tt.reason)
			assert.Equal(t, len(got.Comments), 0)
		})
	}
}
