package promptbuild

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestDetectMode(t *testing.T) {
	tests := []struct {
		name           string
		triggerComment string
		want           string
	}{
		{name: "empty trigger comment defaults to review", triggerComment: "", want: ModeReview},
		{name: "slash paco review", triggerComment: "/paco review", want: ModeReview},
		{name: "slash paco summary", triggerComment: "/paco summary", want: ModeSummary},
		{name: "slash paco summary case insensitive", triggerComment: "/paco SUMMARY", want: ModeSummary},
		{name: "summary with trailing text on next line", triggerComment: "/paco summary\nplease", want: ModeSummary},
		{name: "unrelated comment defaults to review", triggerComment: "looks good to me", want: ModeReview},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMode(tt.triggerComment)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestParseToolchainVersions(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []ToolchainVersion
	}{
		{name: "empty data", data: "", want: nil},
		{
			name: "single entry",
			data: "Go\t1.24.0\tgo.mod\n",
			want: []ToolchainVersion{{Language: "Go", Version: "1.24.0", Source: "go.mod"}},
		},
		{
			name: "multiple entries and blank lines are skipped",
			data: "Go\t1.24.0\tgo.mod\n\nPython\t3.12\t.python-version\n",
			want: []ToolchainVersion{
				{Language: "Go", Version: "1.24.0", Source: "go.mod"},
				{Language: "Python", Version: "3.12", Source: ".python-version"},
			},
		},
		{name: "malformed line without three fields is skipped", data: "Go\t1.24.0\n", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolchainVersions(tt.data)
			assert.DeepEqual(t, got, tt.want)
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		diff        string
		feedback    string
		reviewRules string
		toolchains  []ToolchainVersion
		wantContain []string
		wantOmit    []string
	}{
		{
			name:        "review mode with just a diff",
			mode:        ModeReview,
			diff:        "+ added line",
			wantContain: []string{"+ added line", "\"comments\""},
			wantOmit:    []string{"EXISTING FEEDBACK", "TRUSTED REVIEW RULES", "summary-only request"},
		},
		{
			name:        "summary mode adds summary-only instructions",
			mode:        ModeSummary,
			diff:        "+ added line",
			wantContain: []string{"summary-only request"},
		},
		{
			name:        "feedback and review rules are included as data blocks",
			mode:        ModeReview,
			diff:        "+ added line",
			feedback:    "prior nit about naming",
			reviewRules: "always use table-driven tests",
			wantContain: []string{
				"BEGIN EXISTING FEEDBACK", "prior nit about naming",
				"BEGIN TRUSTED REVIEW RULES", "always use table-driven tests",
			},
		},
		{
			name:        "toolchain versions are listed",
			mode:        ModeReview,
			diff:        "+ added line",
			toolchains:  []ToolchainVersion{{Language: "Go", Version: "1.24.0", Source: "go.mod"}},
			wantContain: []string{"Go 1.24.0 (from go.mod)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPrompt(tt.mode, tt.diff, tt.feedback, tt.reviewRules, tt.toolchains)
			for _, want := range tt.wantContain {
				assert.Assert(t, is.Contains(got, want))
			}
			for _, omit := range tt.wantOmit {
				assert.Assert(t, !strings.Contains(got, omit), "expected prompt to not contain %q", omit)
			}
		})
	}
}

func TestExtractEmbedded(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantProse  string
		wantOK     bool
		wantRating int
		wantSec    bool
		wantLen    int
	}{
		{
			name:      "no fence present",
			text:      "just plain prose, no block here",
			wantProse: "just plain prose, no block here",
			wantOK:    false,
		},
		{
			name: "well-formed fence is parsed and stripped from prose",
			text: "Looks fine overall.\n\n```paco-review\n" +
				`{"review_score":{"rating":2,"reason":"low risk"},"security_sensitive":true,` +
				`"comments":[{"path":"a.go","line":3,"severity":"medium","body":"consider a check"}]}` +
				"\n```",
			wantProse:  "Looks fine overall.",
			wantOK:     true,
			wantRating: 2,
			wantSec:    true,
			wantLen:    1,
		},
		{
			name:      "malformed JSON inside the fence falls back to no match",
			text:      "prose\n```paco-review\nnot json\n```",
			wantProse: "prose\n```paco-review\nnot json\n```",
			wantOK:    false,
		},
		{
			name: "unescaped quotes around a backtick-quoted literal are repaired",
			text: "Looks risky.\n\n```paco-review\n" +
				"{\"review_score\":{\"rating\":3,\"reason\":\"panics on empty input\"}," +
				"\"security_sensitive\":false,\"comments\":[{\"path\":\"a.go\",\"line\":5," +
				"\"severity\":\"medium\",\"body\":\"should return `\"\"` for no words\"}]}" +
				"\n```",
			wantProse:  "Looks risky.",
			wantOK:     true,
			wantRating: 3,
			wantLen:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prose, embedded, ok := ExtractEmbedded(tt.text)
			assert.Equal(t, ok, tt.wantOK)
			assert.Equal(t, prose, tt.wantProse)
			if tt.wantOK {
				assert.Equal(t, embedded.ReviewScore.Rating, tt.wantRating)
				assert.Equal(t, embedded.SecuritySensitive, tt.wantSec)
				assert.Equal(t, len(embedded.Comments), tt.wantLen)
			} else {
				assert.Assert(t, embedded == nil)
			}
		})
	}
}

func TestBound(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		wantExact string
		checkLen  bool
	}{
		{name: "short prompt is unchanged", prompt: "hello", wantExact: "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bound(tt.prompt)
			assert.Equal(t, got, tt.wantExact)
		})
	}

	t.Run("long prompt is truncated to MaxRequestBytes and carries a marker", func(t *testing.T) {
		long := strings.Repeat("a", MaxRequestBytes*2)
		got := Bound(long)
		assert.Assert(t, len(got) <= MaxRequestBytes)
		assert.Assert(t, is.Contains(got, "truncated"))
	})
}

func TestBuildBoundedPrompt(t *testing.T) {
	t.Run("small diff survives even when reviewRules alone would blow the budget", func(t *testing.T) {
		hugeRules := strings.Repeat("rule ", 2000) // way over encodedResultBudget alone
		got := BuildBoundedPrompt(ModeReview, "diff --git a/f.go b/f.go\n+bug", "some feedback", hugeRules, nil)
		assert.Assert(t, is.Contains(got, "+bug"))
		assert.Assert(t, !strings.Contains(got, hugeRules))
	})

	t.Run("small diff survives even when feedback alone would blow the budget", func(t *testing.T) {
		hugeFeedback := strings.Repeat("noise ", 2000)
		got := BuildBoundedPrompt(ModeReview, "diff --git a/f.go b/f.go\n+bug", hugeFeedback, "", nil)
		assert.Assert(t, is.Contains(got, "+bug"))
		assert.Assert(t, !strings.Contains(got, hugeFeedback))
	})

	t.Run("huge diff still falls back to Bound's own truncation marker", func(t *testing.T) {
		hugeDiff := strings.Repeat("a", MaxRequestBytes*2)
		got := BuildBoundedPrompt(ModeReview, hugeDiff, "", "", nil)
		assert.Assert(t, is.Contains(got, "truncated"))
	})
}
