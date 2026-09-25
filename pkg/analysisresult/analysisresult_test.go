package analysisresult

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestToReview(t *testing.T) {
	tests := []struct {
		name           string
		outcome        string
		object         map[string]interface{}
		wantErr        string
		wantSummary    string
		wantRating     int
		wantComments   int
		wantNoComments bool
	}{
		{
			name:    "no action required uses diagnosis summary",
			outcome: OutcomeNoActionRequired,
			object: map[string]interface{}{
				"status": map[string]interface{}{
					"diagnosis": map[string]interface{}{
						"summary":   "The change looks safe.",
						"rootCause": "n/a",
					},
				},
			},
			wantSummary:    "The change looks safe.",
			wantRating:     1,
			wantNoComments: true,
		},
		{
			name:    "no action required with empty diagnosis falls back to default summary",
			outcome: OutcomeNoActionRequired,
			object: map[string]interface{}{
				"status": map[string]interface{}{},
			},
			wantSummary:    "No issues found.",
			wantRating:     1,
			wantNoComments: true,
		},
		{
			name:    "no action required extracts embedded paco-review comments",
			outcome: OutcomeNoActionRequired,
			object: map[string]interface{}{
				"status": map[string]interface{}{
					"diagnosis": map[string]interface{}{
						"summary": "The change looks mostly safe.\n\n" +
							"```paco-review\n" +
							`{"review_score":{"rating":2,"reason":"minor nit"},"security_sensitive":false,` +
							`"comments":[{"path":"main.go","line":10,"severity":"low","body":"unused var"}]}` +
							"\n```",
					},
				},
			},
			wantSummary:  "The change looks mostly safe.",
			wantRating:   1,
			wantComments: 1,
		},
		{
			name:    "action required lists remediation option titles and summaries",
			outcome: OutcomeActionRequired,
			object: map[string]interface{}{
				"status": map[string]interface{}{
					"options": []interface{}{
						map[string]interface{}{
							"title":   "Fix nil pointer",
							"summary": "handler.go:42 dereferences before nil check",
						},
					},
				},
			},
			wantSummary:    "Fix nil pointer",
			wantRating:     defaultRating + 1,
			wantNoComments: true,
		},
		{
			name:    "action required extracts embedded paco-review comments from option diagnosis",
			outcome: OutcomeActionRequired,
			object: map[string]interface{}{
				"status": map[string]interface{}{
					"options": []interface{}{
						map[string]interface{}{
							"title":   "Fix nil pointer",
							"summary": "handler.go:42 dereferences before nil check",
							"diagnosis": map[string]interface{}{
								"summary": "Detailed analysis of the nil dereference.\n\n" +
									"```paco-review\n" +
									`{"review_score":{"rating":4,"reason":"crash risk"},"security_sensitive":true,` +
									`"comments":[{"path":"handler.go","line":42,"severity":"high","body":"nil deref"}]}` +
									"\n```",
							},
						},
					},
				},
			},
			wantSummary:  "Fix nil pointer",
			wantRating:   defaultRating + 1,
			wantComments: 1,
		},
		{
			name:    "action required with no options still returns a review",
			outcome: OutcomeActionRequired,
			object: map[string]interface{}{
				"status": map[string]interface{}{},
			},
			wantSummary:    "no remediation options",
			wantRating:     defaultRating,
			wantNoComments: true,
		},
		{
			name:    "unrecognized outcome is an error",
			outcome: "something-else",
			object:  map[string]interface{}{"status": map[string]interface{}{}},
			wantErr: "unrecognized outcome",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{Object: tt.object}

			got, err := ToReview(tt.outcome, obj)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			if tt.wantNoComments {
				assert.Assert(t, len(got.Comments) == 0, "expected no per-line comments")
			} else {
				assert.Equal(t, len(got.Comments), tt.wantComments)
			}
			assert.Equal(t, got.ReviewScore.Rating, tt.wantRating)
			assert.Assert(t, strings.Contains(got.Summary, tt.wantSummary))
			assert.Assert(t, !strings.Contains(got.Summary, "paco-review"),
				"the fenced JSON block must not leak into the user-visible summary")
		})
	}
}
