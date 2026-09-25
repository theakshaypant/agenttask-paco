package artifact

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWorkspaceReadWriteExists(t *testing.T) {
	tests := []struct {
		name string
		file string
		data []byte
	}{
		{name: "simple top-level file", file: "hello.txt", data: []byte("hi")},
		{name: "nested file creates parent dirs", file: filepath.Join(".tekton", "ai", "REVIEW.md"), data: []byte("# rules")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{Dir: t.TempDir()}

			assert.Assert(t, !ws.Exists(tt.file), "file should not exist before Write")

			assert.NilError(t, ws.Write(tt.file, tt.data))
			assert.Assert(t, ws.Exists(tt.file))

			got, err := ws.Read(tt.file)
			assert.NilError(t, err)
			assert.Equal(t, string(got), string(tt.data))
		})
	}
}

func TestWorkspaceReadOptional(t *testing.T) {
	tests := []struct {
		name    string
		seed    bool
		content string
		want    string
	}{
		{name: "missing file returns empty string", seed: false, want: ""},
		{name: "present file returns its content", seed: true, content: "some content", want: "some content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{Dir: t.TempDir()}
			if tt.seed {
				assert.NilError(t, ws.Write("artifact.txt", []byte(tt.content)))
			}
			got := ws.ReadOptional("artifact.txt")
			assert.Equal(t, got, tt.want)
		})
	}
}
