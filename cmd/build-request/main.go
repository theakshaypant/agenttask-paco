// Command build-request is the second step of the agenttask-paco pipeline.
//
// It reads the workspace artifacts written by paco-cli's own "diff" step
// (.pr.diff, .existing-feedback.txt, .tekton/ai/REVIEW.md,
// .toolchain-versions) and renders the single free-text prompt string that
// gets passed as the "request" param to the "review" AgentTask CustomTask
// step. The rendered prompt and a "skip" flag are written to files
// (typically Tekton Task result paths) so they can flow into later
// PipelineTasks.
//
// When paco-cli's "diff" step already decided to skip the run (no
// reviewable diff), this command writes the final .paco-review.json/
// .paco-failed artifacts itself - matching paco-cli's own review step
// behavior - and reports skip=true so the pipeline can bypass both the
// "review" AgentTask step and "translate-result".
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/theakshaypant/agenttask-paco/pkg/artifact"
	"github.com/theakshaypant/agenttask-paco/pkg/promptbuild"
	"github.com/theakshaypant/agenttask-paco/pkg/review"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "build-request:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("build-request", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "Workspace directory containing paco-cli diff artifacts")
	triggerComment := fs.String("trigger-comment", os.Getenv("TRIGGER_COMMENT"),
		"Trigger comment text (determines review vs summary mode)")
	requestOut := fs.String("request-out", "", "File path to write the rendered request string to (required)")
	skipOut := fs.String("skip-out", "", "File path to write \"true\"/\"false\" to, gating the review step (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *requestOut == "" || *skipOut == "" {
		return errors.New("--request-out and --skip-out are required")
	}

	ws := &artifact.Workspace{Dir: *workspace}

	skip, reason := shouldSkip(ws)
	if skip {
		if err := review.WriteFailure(ws, reason); err != nil {
			return err
		}
		return writeOutputs(*requestOut, "", *skipOut, true)
	}

	diff, err := ws.Read(artifact.FileDiff)
	if err != nil {
		return fmt.Errorf("reading %s: %w", artifact.FileDiff, err)
	}

	feedback := ws.ReadOptional(artifact.FileExistingFeedback)
	reviewRules := ws.ReadOptional(artifact.FileReviewRules)
	toolchains := promptbuild.ParseToolchainVersions(ws.ReadOptional(artifact.FileToolchains))

	mode := promptbuild.DetectMode(*triggerComment)
	if err := ws.Write(artifact.FileMode, []byte(mode+"\n")); err != nil {
		return fmt.Errorf("writing mode artifact: %w", err)
	}

	prompt := promptbuild.Bound(promptbuild.BuildPrompt(mode, string(diff), feedback, reviewRules, toolchains))
	return writeOutputs(*requestOut, prompt, *skipOut, false)
}

// shouldSkip reports whether paco-cli's diff step decided to skip the run,
// or there is simply no reviewable diff to send to the review engine.
func shouldSkip(ws *artifact.Workspace) (bool, string) {
	if ws.Exists(artifact.FileError) {
		return true, strings.TrimSpace(ws.ReadOptional(artifact.FileError))
	}
	diff, err := ws.Read(artifact.FileDiff)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "No reviewable changes found in this diff."
		}
		// A non-not-exist error (e.g. permission denied because this
		// step's container runs as a different UID than the one that
		// wrote the shared workspace) means the diff exists but
		// couldn't be read - that's an infra bug, not "nothing to
		// review", so it must not be silently swallowed as a skip.
		return true, fmt.Sprintf("reading %s: %v", artifact.FileDiff, err)
	}
	if len(diff) == 0 {
		return true, "No reviewable changes found in this diff."
	}
	return false, ""
}

func writeOutputs(requestOut, request, skipOut string, skip bool) error {
	skipStr := "false"
	if skip {
		skipStr = "true"
	}
	if err := os.WriteFile(requestOut, []byte(request), 0o600); err != nil { //nolint:gosec // requestOut is an operator-controlled Task result path, not end-user input
		return fmt.Errorf("writing %s: %w", requestOut, err)
	}
	if err := os.WriteFile(skipOut, []byte(skipStr), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", skipOut, err)
	}
	return nil
}
