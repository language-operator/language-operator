# Execution Modes

A `LanguageAgent` runs as an [Argo Workflow](https://argo-workflows.readthedocs.io/). `spec.execution.mode` decides whether the agent is always on or invoked.

## The WorkflowTemplate

Whatever the mode, the operator renders one `WorkflowTemplate` named after the agent. It holds the agent's pod spec — image, env, config mount, workspace, tool sidecars — and is the unit you submit against for a one-off run:

```bash
argo submit --from workflowtemplate/my-agent -n my-cluster
argo list -n my-cluster
argo logs @latest -n my-cluster
```

Everything else the operator creates is derived from it.

## `mode: service` (default)

The always-on agent. The operator creates a long-lived `Workflow` that references the template and retries forever, so a crashed agent comes back the way a Deployment would have restarted it.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: assistant
spec:
  runtime: claude-code
  instructions: Help users explore the codebase.
  # execution.mode defaults to service
```

A service agent is **addressable**: it gets a ClusterIP Service on `spec.ports` (default `http`/`8080`), and an Ingress at `<agent>.<cluster domain>` when the cluster has a domain configured.

```bash
kubectl get lagent assistant
# NAME        MODE      PHASE     SCHEDULE   LAST RUN
# assistant   service   Running              Running
```

Because an Argo `Workflow` spec cannot be updated once running, editing the agent **replaces** the Workflow — the pod restarts. This is the same practical behaviour as a Deployment rollout.

## `mode: task`

The invoked agent. Each run starts, does its work, and exits. Set `spec.execution.schedule` to have a `CronWorkflow` fire it on a cron schedule:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: triage
spec:
  runtime: claude-code
  instructions: |
    Triage open GitHub issues, label them, and stop.
  execution:
    mode: task
    schedule: "*/15 * * * *"
    timezone: America/New_York
    concurrencyPolicy: Forbid
    activeDeadlineSeconds: 900
```

Omit `schedule` and the agent has only its template — invoke it by hand, or from whatever drives your automation.

Task agents are **not addressable**. Their pods exist only for the duration of a run, so a Service would point at nothing in between; `spec.ports` is rejected for them.

```bash
kubectl get lagent triage
# NAME     MODE   PHASE       SCHEDULE       LAST RUN
# triage   task   Succeeded   */15 * * * *   Succeeded
```

## Schedules

`spec.execution.schedule` accepts a standard 5-field cron expression (`minute hour day-of-month month day-of-week`), textual month and day aliases (`0 0 1 JAN MON`), the `@yearly`/`@monthly`/`@weekly`/`@daily`/`@hourly` macros, and `@every <duration>` (e.g. `@every 90m`). Invalid expressions are rejected at admission rather than silently never firing.

`concurrencyPolicy` decides what happens when a run is due while the previous one is still going:

| Value | Behaviour |
|-------|-----------|
| `Forbid` (default) | Skip the new run |
| `Allow` | Start it anyway, alongside the running one |
| `Replace` | Stop the running one and start the new one |

## Suspending

`spec.execution.suspend: true` stops the agent without deleting it: a service agent's Workflow is torn down, and a task agent's CronWorkflow stops firing. The `WorkflowTemplate` stays, so you can still trigger a run by hand while the schedule is paused.

## Status

| Field | Meaning |
|-------|---------|
| `status.phase` | `Pending`, `Running`, `Succeeded`, `Failed`, `Suspended`, or `Degraded` |
| `status.workflowTemplateName` | The template to submit against |
| `status.activeWorkflowName` | The long-lived Workflow (service mode only) |
| `status.lastRunName` / `lastRunPhase` | The most recent run and its Argo phase |
| `status.lastRunStartedAt` / `lastRunFinishedAt` | When it ran |
| `status.lastScheduledTime` | When the CronWorkflow last fired (task mode) |

`Degraded` means the agent is running but a non-critical subsystem failed — most often a NetworkPolicy that did not converge. Check `status.conditions` for the reason.

## No replicas, no autoscaling

An Argo Workflow has no replica count and no scale subresource, so `spec.deployment.replicas` and `spec.deployment.autoscaling` are rejected at admission rather than silently ignored. To run more agents, create more agents.

## ServiceAccount requirements

Agent pods run as `language-agent-<agent-name>`, which the operator grants `create` and `patch` on `argoproj.io/workflowtaskresults` — the Argo executor uses it to report each node's outcome. If you set `spec.deployment.serviceAccountName` to a ServiceAccount of your own, it must carry that permission or every run will fail at completion:

```yaml
- apiGroups: ["argoproj.io"]
  resources: ["workflowtaskresults"]
  verbs: ["create", "patch"]
```

## Prerequisites

Argo Workflows must be installed. The `language-operator` chart bundles it as a subchart and installs it by default; the operator refuses to start if the `argoproj.io` CRDs are missing. To manage Argo yourself:

```bash
helm install language-operator ... --set argo-workflows.enabled=false
```
