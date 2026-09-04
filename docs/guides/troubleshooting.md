# Troubleshooting

Common failure modes, and how to tell them apart.

## The operator will not start

```
Argo Workflows is not installed in this cluster. LanguageAgents run as Argo Workflows.
```

Agents run as Argo Workflows, so the operator checks for the `argoproj.io` CRDs before it
starts and exits if they are missing. This is deliberate — the alternative is every agent
reconcile failing with `no matches for kind "WorkflowTemplate"`, which says nothing about the
cause.

```bash
kubectl get crds | grep argoproj.io
```

Expect `workflows`, `workflowtemplates`, `cronworkflows`, and `workflowtaskresults`.

If they are missing and you installed with the bundled subchart, the pre-install hook that
applies them failed:

```bash
kubectl logs -n language-operator job/language-operator-argo-workflows-crd-install
```

If you run your own Argo, see [Bring your own Argo](../getting-started/installation.md#bring-your-own-argo).

## An agent never runs

Check what the operator thinks it created:

```bash
kubectl get lagent <agent> -n <namespace> -o wide
kubectl get workflowtemplate,workflow,cronworkflow -n <namespace>
```

| Symptom | Cause |
|---|---|
| No `WorkflowTemplate` | The operator could not build the pod spec. Check `status.conditions` for a `WorkflowError` reason. |
| `WorkflowTemplate` but no `Workflow`, `mode: service` | The agent is suspended (`spec.execution.suspend: true`), or the operator is not running. |
| `CronWorkflow` exists but never fires | The Argo workflow controller is not watching this namespace, or the CronWorkflow is suspended. |
| Phase stuck at `Pending` | The pod cannot be scheduled or its image cannot be pulled. `kubectl describe pod -n <namespace> -l langop.io/kind=LanguageAgent`. |

A namespace-scoped Argo installation is the quietest failure here: the objects are created
correctly and simply never run. Confirm the controller watches your namespace:

```bash
kubectl get configmap -n language-operator \
  language-operator-argo-workflows-workflow-controller-configmap -o yaml | grep -A3 namespaces
```

## Runs fail at completion

If the agent's own work succeeds but the run ends `Failed`, and the pod logs show an RBAC
error mentioning `workflowtaskresults`, the agent's ServiceAccount is missing a permission
the Argo executor needs to report each node's outcome.

The operator grants this automatically to the ServiceAccount it manages. If you set
`spec.deployment.serviceAccountName` to one of your own, add:

```yaml
- apiGroups: ["argoproj.io"]
  resources: ["workflowtaskresults"]
  verbs: ["create", "patch"]
```

## Admission rejections

These are deliberate. The operator rejects them rather than silently ignoring them, because
silently ignoring a setting is worse than failing to apply it.

| Message | Why | Fix |
|---|---|---|
| `spec.deployment.replicas: not supported` | An Argo Workflow has no replica count | Remove the field. To run more agents, create more agents. |
| `spec.deployment.autoscaling: not supported` | No scale subresource to target | Remove the field. |
| `spec.ports: not supported when spec.execution.mode is "task"` | A task agent's pods exist only during a run, so a Service would point at nothing | Remove `spec.ports`, or switch to `mode: service`. |
| `spec.execution.schedule: only valid when mode is "task"` | A service agent runs continuously; a schedule would do nothing | Set `mode: task`, or remove the schedule. |
| `spec.execution.schedule: ... must have 5 fields` | Invalid cron expression | Use a 5-field expression or an `@` macro. See [Execution Modes](execution-modes.md#schedules). |

## An agent restarts when I edit it

Expected. An Argo `Workflow` spec cannot be updated once it is running, so the operator
**replaces** a service-mode agent's Workflow whenever its config hash or generation changes.
The pod restarts, exactly as it would have under a Deployment rollout.

This matters for interactive agents: editing a Claude Code agent's spec ends the terminal
session. Anything you need to keep should live on the workspace PVC, which survives.

It also matters for [self-configuration](../api/languageagentselfconfig.md) — an agent that
patches its own spec restarts itself.

## Logs

Agents run as Argo pods, so the container is named `main`:

```bash
argo logs @latest -n <namespace> -f
kubectl logs -n <namespace> -l langop.io/kind=LanguageAgent -c main -f
```

For a finished task run, name it directly — the pod may already be gone, but Argo retains
the run until `spec.execution.ttlSecondsAfterFinished` elapses:

```bash
argo list -n <namespace>
argo logs <run-name> -n <namespace>
```

## Everything is stuck and I want to start over

`make wipe` removes every trace of a development install — CRs (stripping finalizers if the
operator is already gone), both Helm releases, the `langop.io` and `argoproj.io` CRDs, the
namespace, orphaned cluster-scoped resources, and the dev images imported into k3s.
