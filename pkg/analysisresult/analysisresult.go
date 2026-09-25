// Package analysisresult fetches the agentic.openshift.io/v1alpha1
// AnalysisResult object created by the real Lightspeed Agentic Operator and
// maps it into paco-cli's own review.Review schema, including per-line
// inline comments extracted from a "paco-review" fenced block the model
// was asked (via pkg/promptbuild) to embed in its free-text response - see
// promptbuild.ExtractEmbedded.
//
// A dynamic/unstructured client is used deliberately instead of importing
// github.com/openshift/lightspeed-agentic-operator/api (an unreleased
// pseudo-version pinned by agenttask-adapter-lightspeed), to avoid coupling
// agenttask-paco's build to that experimental module's exact Go API.
package analysisresult

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/theakshaypant/agenttask-paco/pkg/promptbuild"
	"github.com/theakshaypant/agenttask-paco/pkg/review"
)

// GroupVersionResource is the GVR of the AnalysisResult CRD, as defined by
// github.com/openshift/lightspeed-agentic-operator/api/v1alpha1.
var GroupVersionResource = schema.GroupVersionResource{
	Group:    "agentic.openshift.io",
	Version:  "v1alpha1",
	Resource: "analysisresults",
}

const (
	// OutcomeActionRequired and OutcomeNoActionRequired mirror the two
	// values agenttask-adapter-lightspeed's "outcome" CustomRun result can
	// take (see agenttask-adapter-lightspeed/pkg/adapter/adapter.go).
	OutcomeActionRequired   = "action-required"
	OutcomeNoActionRequired = "no-action-required"
)

// defaultRating is used when Lightspeed's AnalysisResult gives us no
// numeric risk/difficulty signal to map onto paco-cli's 1-5 review_score.
const defaultRating = 3

// Fetch retrieves the named AnalysisResult from the given namespace.
func Fetch(ctx context.Context, client dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	obj, err := client.Resource(GroupVersionResource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting AnalysisResult %s/%s: %w", namespace, name, err)
	}
	return obj, nil
}

// ToReview maps an AnalysisResult (fetched via Fetch) plus the adapter's
// "outcome" CustomRun result into paco-cli's review.Review schema.
//
// Lightspeed's AnalysisResult itself has no native per-line/file anchoring
// field, but pkg/promptbuild's request asks the model to embed one as a
// "paco-review" fenced JSON block inside its free-text response (which
// AnalysisResult carries verbatim in "diagnosis.summary"/an option's
// "description"). ToReview extracts that block, on a best-effort basis, via
// promptbuild.ExtractEmbedded and hands the resulting Comments straight
// through to paco-cli's unmodified "post" step, which posts genuine inline
// PR review comments exactly as it does for its own opencode-based
// reviews. When no such block is present (the model didn't follow the
// instructions, or an older prompt was used), Comments falls back to empty
// and the analysis is still fully surfaced through Summary.
func ToReview(outcome string, result *unstructured.Unstructured) (*review.Review, error) {
	switch outcome {
	case OutcomeNoActionRequired:
		return noActionRequiredReview(result)
	case OutcomeActionRequired:
		return actionRequiredReview(result)
	default:
		return nil, fmt.Errorf("unrecognized outcome %q from agenttask-adapter-lightspeed", outcome)
	}
}

func noActionRequiredReview(result *unstructured.Unstructured) (*review.Review, error) {
	summary, _, err := unstructured.NestedString(result.Object, "status", "diagnosis", "summary")
	if err != nil {
		return nil, fmt.Errorf("reading status.diagnosis.summary: %w", err)
	}
	rootCause, _, _ := unstructured.NestedString(result.Object, "status", "diagnosis", "rootCause")

	prose, embedded, ok := promptbuild.ExtractEmbedded(summary)
	if prose == "" {
		prose = "No issues found."
	}
	rev := &review.Review{
		Summary:     prose,
		ReviewScore: review.ReviewScore{Rating: 1, Reason: rootCause},
		Comments:    []review.Comment{},
	}
	if ok {
		rev.SecuritySensitive = embedded.SecuritySensitive
		rev.Comments = embedded.Comments
		if embedded.ReviewScore.Reason != "" {
			rev.ReviewScore.Reason = embedded.ReviewScore.Reason
		}
	}
	return rev, nil
}

func actionRequiredReview(result *unstructured.Unstructured) (*review.Review, error) {
	options, _, err := unstructured.NestedSlice(result.Object, "status", "options")
	if err != nil {
		return nil, fmt.Errorf("reading status.options: %w", err)
	}
	if len(options) == 0 {
		return &review.Review{
			Summary:     "Paco (via Lightspeed) flagged this change as requiring attention, but returned no remediation options.",
			ReviewScore: review.ReviewScore{Rating: defaultRating},
			Comments:    []review.Comment{},
		}, nil
	}

	summary := "Paco (via Lightspeed) found the following:\n\n"
	reason := ""
	var comments []review.Comment
	var securitySensitive bool
	for i, o := range options {
		opt, ok := o.(map[string]interface{})
		if !ok {
			continue
		}
		title, _, _ := unstructured.NestedString(opt, "title")
		optSummary, _, _ := unstructured.NestedString(opt, "summary")
		if title == "" {
			title = fmt.Sprintf("Option %d", i+1)
		}
		summary += fmt.Sprintf("- **%s**", title)
		if optSummary != "" {
			summary += ": " + optSummary
		}
		summary += "\n"

		// Each option's own "diagnosis.summary" carries the model's
		// free-text analysis for that option, which is where
		// pkg/promptbuild's requested "paco-review" fenced block (if any)
		// lands - see promptbuild.ExtractEmbedded.
		diagnosisSummary, _, _ := unstructured.NestedString(opt, "diagnosis", "summary")
		_, embedded, ok := promptbuild.ExtractEmbedded(diagnosisSummary)
		if ok {
			comments = append(comments, embedded.Comments...)
			securitySensitive = securitySensitive || embedded.SecuritySensitive
			if reason == "" && embedded.ReviewScore.Reason != "" {
				reason = embedded.ReviewScore.Reason
			}
		}
		if reason == "" {
			rootCause, _, _ := unstructured.NestedString(opt, "diagnosis", "rootCause")
			reason = rootCause
		}
	}
	if comments == nil {
		comments = []review.Comment{}
	}

	return &review.Review{
		Summary:           summary,
		ReviewScore:       review.ReviewScore{Rating: defaultRating + 1, Reason: reason},
		SecuritySensitive: securitySensitive,
		Comments:          comments,
	}, nil
}
