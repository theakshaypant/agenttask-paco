# Failure investigation: direct invocation vs. AgentTask

A PipelineRun fails on GitHub and kicks off a second, analysis PipelineRun. An
agent in that run pulls the failed TaskRun's logs, events, and cluster state, works
out the root cause, and reports back (a comment, a status, a ticket). Two ways to
wire that agent in:

- **A — Direct invocation**: the pipeline runs the agent itself, as a step.
- **B — AgentTask**: the pipeline declares an `AgentTask` CustomRun, and an adapter
  controller runs it out-of-band (the engine lives behind the adapter, swappable).

## Trade-offs

| Dimension | A — Direct invocation | B — AgentTask |
|---|---|---|
| Who runs the agent | The pipeline | A controller, out-of-band |
| Engine/model/creds | In the Task | Behind `adapterRef` |
| Clean result for later steps | You write glue to parse the agent's prose | Declared on the `AgentTask`, filled by the adapter |
| Reuse across pipelines | Copy the Task per pipeline | One CR, any pipeline |
| Run as an object / RBAC | Logs only | A `CustomRun` under K8s RBAC |
| Isolation & security | Shares the pipeline's pod, SA, secrets | Adapter can sandbox it off the pipeline |
| Operational weight | Just the engine image | Controller + CRD + adapter |
| Maturity | Established | PoC (`v0.0.0-poc.1`, adapter untagged) |

Both are normal Tekton pipelines, so both can chain tasks and branch on results.
The difference isn't whether you can act on the verdict — it's where the clean
result comes from: glue you write in A, the contract in B.

## Which to pick

**A** is simpler when one known runtime is enough and you're fine maintaining it
per pipeline. For a read-only, low-risk agent, a tightly scoped Task is plenty.

**B** earns its extra plumbing when you want to build the agent once and reuse it,
swap the engine without touching pipelines, get clean results from the contract
instead of glue, and have every run tracked as an auditable object — at the cost of
a pre-release controller/adapter stack.

Security can decide it on its own: an agent that reads the cluster or runs commands
inherits the pipeline's pod, SA, and secrets under A, whereas a sandboxing adapter
under B can confine it to a minimal SA and egress — if the adapter actually does.

## Sources

- AgentTask pipeline — [`.tekton/pr-review-pipeline.yaml`](.tekton/pr-review-pipeline.yaml)
- AgentTask controller — [`github.com/openshift-pipelines/agenttask`][agenttask]
- Reference adapter — [`github.com/openshift-pipelines/agenttask-adapter-lightspeed`][adapter]

[agenttask]: https://github.com/openshift-pipelines/agenttask
[adapter]: https://github.com/openshift-pipelines/agenttask-adapter-lightspeed
