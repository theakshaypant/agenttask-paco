// Package review defines the normalized review result schema written to the
// ".paco-review.json" workspace artifact.
//
// This schema intentionally matches paco-cli's own internal/review.Review
// type (github.com/pipelines-as-code/paco-cli) field-for-field, so the real,
// unmodified paco-cli "post" step can read whatever this package writes.
package review

import (
	"encoding/json"
	"fmt"

	"github.com/theakshaypant/agenttask-paco/pkg/artifact"
)

// Review is the normalized review result posted to a pull request by
// paco-cli's "post" step.
type Review struct {
	Summary           string      `json:"summary"`
	ReviewScore       ReviewScore `json:"review_score"`
	SecuritySensitive bool        `json:"security_sensitive"`
	Comments          []Comment   `json:"comments"`
}

// ReviewScore is a 1-5 difficulty/risk rating with a short reason.
type ReviewScore struct {
	Rating int    `json:"rating"`
	Reason string `json:"reason"`
}

// Comment is a single inline finding anchored to a file and line.
type Comment struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Body     string `json:"body"`
}

// ValidSeverities are the severities paco-cli's "post" step accepts; any
// other value is normalized to "medium" downstream.
var ValidSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
}

// MaxComments is the maximum number of inline comments paco-cli's "post"
// step will submit in a single review.
const MaxComments = 30

// WriteFailure writes a failed/skip review artifact matching paco-cli's own
// internal/review.Run "writeFail" behavior: a Review whose Summary is the
// human-readable reason and an empty Comments slice, plus the
// artifact.FileFailed marker so paco-cli's "post" step surfaces the message
// instead of an empty review.
func WriteFailure(ws *artifact.Workspace, reason string) error {
	r := Review{Summary: reason, Comments: []Comment{}}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshaling failure review: %w", err)
	}
	if err := ws.Write(artifact.FileReview, data); err != nil {
		return fmt.Errorf("writing %s: %w", artifact.FileReview, err)
	}
	return ws.Write(artifact.FileFailed, nil)
}
