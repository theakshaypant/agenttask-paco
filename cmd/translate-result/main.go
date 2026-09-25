// Command translate-result is the fourth step of the agenttask-paco
// pipeline.
//
// It takes the "outcome" and "analysis-result-name" CustomRun results
// produced by the "review" AgentTask/agenttask-adapter-lightspeed step,
// fetches the real agentic.openshift.io/v1alpha1 AnalysisResult object the
// Lightspeed Agentic Operator created, and writes it into the workspace as
// ".paco-review.json" using paco-cli's own review.Review schema - so the
// real, unmodified paco-cli "post" step can post it to the pull request.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/theakshaypant/agenttask-paco/pkg/analysisresult"
	"github.com/theakshaypant/agenttask-paco/pkg/artifact"
	"github.com/theakshaypant/agenttask-paco/pkg/review"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "translate-result:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("translate-result", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "Workspace directory to write .paco-review.json into")
	namespace := fs.String("namespace", "", "Namespace of the AnalysisResult object (required)")
	outcome := fs.String("outcome", "", "Value of the review AgentTask CustomRun's \"outcome\" result")
	analysisResultName := fs.String("analysis-result-name", "",
		"Value of the review AgentTask CustomRun's \"analysis-result-name\" result")
	skip := fs.Bool("skip", false,
		"When true, this step is a no-op: build-request already wrote the terminal .paco-review.json/.paco-failed")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ws := &artifact.Workspace{Dir: *workspace}

	if *skip {
		return nil
	}
	if *namespace == "" || *outcome == "" || *analysisResultName == "" {
		return errors.New("--namespace, --outcome, and --analysis-result-name are required unless --skip is set")
	}

	client, err := newDynamicClient()
	if err != nil {
		return review.WriteFailure(ws, "Paco (via Lightspeed): could not build a Kubernetes client: "+err.Error())
	}

	obj, err := analysisresult.Fetch(ctx, client, *namespace, *analysisResultName)
	if err != nil {
		return review.WriteFailure(ws, "Paco (via Lightspeed): "+err.Error())
	}

	rev, err := analysisresult.ToReview(strings.TrimSpace(*outcome), obj)
	if err != nil {
		return review.WriteFailure(ws, "Paco (via Lightspeed): "+err.Error())
	}

	data, err := marshalReview(rev)
	if err != nil {
		return err
	}
	return ws.Write(artifact.FileReview, data)
}

// newDynamicClient builds a dynamic Kubernetes client using in-cluster
// config (the normal case, running as a Tekton Task step under a
// ServiceAccount) or the local kubeconfig (for local testing).
func newDynamicClient() (dynamic.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("building kube client config: %w", err)
		}
	}
	return dynamic.NewForConfig(cfg)
}

func marshalReview(rev *review.Review) ([]byte, error) {
	data, err := json.Marshal(rev)
	if err != nil {
		return nil, fmt.Errorf("marshaling review: %w", err)
	}
	return data, nil
}
