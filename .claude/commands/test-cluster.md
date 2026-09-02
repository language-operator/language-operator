---
description: Inspect the current Kubernetes cluster namespace and the language-operator deployment — check CRs, generated resources, and spot-check for expected behavior
---

## Directions

You are a Kubernetes operator engineer doing a live cluster health check. Use `kubectl` to inspect the current state and report findings clearly. Do not modify anything — read only.

### Step 1 — Identify the context

```bash
kubectl config current-context
kubectl config view --minify --output jsonpath='{.contexts[0].context.namespace}'
```

Determine:
- Current namespace (should be a LanguageCluster-managed namespace, e.g. `language-operator-*`)
- If the namespace looks wrong, note it and continue anyway

### Step 2 — Inspect the operator itself

```bash
kubectl get deployment -n language-operator -o wide
kubectl get pods -n language-operator
kubectl logs -n language-operator -l app.kubernetes.io/name=language-operator --tail=50
```

Check:
- Operator pod is Running and Ready
- No crash loops or error logs
- Leader election is working (look for "acquired" in logs)

### Step 3 — Inspect the current namespace (LanguageCluster)

```bash
kubectl get languagecluster
kubectl get languagemodels
kubectl get languageagentruntimes
kubectl get languageagents
kubectl get languagetools
kubectl get languagepersonas
```

For each CR found, note: name, age, and any status conditions.

### Step 4 — Check generated resources

For each LanguageAgent found, verify the operator created the expected child resources:

Agents run as Argo Workflows, not Deployments.

```bash
kubectl get workflowtemplates,workflows,cronworkflows
kubectl get services
kubectl get pods
kubectl get configmaps
kubectl get secrets
kubectl get pvc
kubectl get networkpolicies
kubectl get lagent -o wide
```

Cross-check, per LanguageAgent:
- A `WorkflowTemplate` named after the agent always exists — it holds the pod spec
- `spec.execution.mode: service` (the default) → also a `Workflow` named after the agent, a
  `Service`, and (with a cluster domain) an `Ingress`
- `spec.execution.mode: task` → a `CronWorkflow` **only if** `spec.execution.schedule` is set,
  and **no** Service or Ingress. A task agent with a Service is a bug
- A ConfigMap `{name}-agent` and a NetworkPolicy in both modes
- If credentials are declared without a `valueFrom`, a `{name}-runtime` Secret should exist
- If `spec.workspace` is enabled, a PVC `{name}-workspace` should exist
- The per-agent Role `language-agent-{name}` grants `create`/`patch` on
  `argoproj.io/workflowtaskresults` — without it every run fails at completion
- No agent should have a Deployment, HorizontalPodAutoscaler, or PodDisruptionBudget

Check the reported status:

```bash
kubectl get lagent -o custom-columns=\
'NAME:.metadata.name,MODE:.spec.execution.mode,PHASE:.status.phase,TEMPLATE:.status.workflowTemplateName,ACTIVE:.status.activeWorkflowName,LASTRUN:.status.lastRunPhase'
```

- Service agents should report `PHASE=Running` with `ACTIVE` set
- Task agents report the most recent run in `LASTRUN`; `ACTIVE` is empty by design
- `PHASE=Suspended` means `spec.execution.suspend` is set

### Step 5 — Check Argo Workflows

The operator refuses to start without these, so a missing piece here explains an operator
CrashLoopBackOff.

```bash
kubectl get crds | grep argoproj.io
kubectl get pods -n language-operator -l app.kubernetes.io/name=argo-workflows-workflow-controller
kubectl logs -n language-operator -l app.kubernetes.io/name=argo-workflows-workflow-controller --tail=30
```

Verify:
- Four CRDs are served: `workflows`, `workflowtemplates`, `cronworkflows`, `workflowtaskresults`
- The workflow controller pod is Running
- Its configured `namespaces` is empty (watch-all). A namespace-scoped controller creates the
  objects correctly and then silently never runs them — the quietest failure mode here

### Step 6 — Check the LiteLLM gateway

```bash
kubectl get deployment gateway
kubectl get service gateway
kubectl get pods -l app.kubernetes.io/component=gateway
kubectl logs -l app.kubernetes.io/component=gateway --tail=30
```

Verify:
- Gateway Deployment exists and pod is Running
- Gateway is serving: `kubectl exec -it <agent-pod> -- wget -qO- http://gateway.<namespace>.svc.cluster.local:8000/v1/models`
- Models listed in the response match the LanguageModel CRs in the namespace

### Step 7 — Spot-check agent config injection

Pick the first Running agent pod and verify operator-injected env vars:

```bash
kubectl exec <pod> -- env | grep -E 'MODEL_ENDPOINT|LLM_MODEL|MCP_SERVERS|AGENT_NAME|AGENT_NAMESPACE|AGENT_CLUSTER'
```

Cross-check against the LanguageAgent spec:
- `MODEL_ENDPOINT` should point to `http://gateway.<namespace>.svc.cluster.local:8000`
- `LLM_MODEL` should list comma-separated model names matching `spec.models`
- `AGENT_NAME` should match the LanguageAgent name
- `MCP_SERVERS` should be present only if tools are referenced

Also check the mounted config file:
```bash
kubectl exec <pod> -- cat /etc/agent/config.yaml
```

Verify models, tools, and persona sections match what the CRs declare.

### Step 8 — Check NetworkPolicies

```bash
kubectl get networkpolicies -o yaml
```

Verify:
- The cluster-level `{cluster-name}-agents` NetworkPolicy selects pods with `langop.io/kind=LanguageAgent` (not the gateway)
- Each agent has its own NetworkPolicy allowing ingress on its ports (`spec.ports`)
- Selectors match the labels Argo stamps on workflow pods via `podMetadata` — if a Service has
  no endpoints, compare its selector against the actual pod labels
- The gateway pod is NOT selected by the agents NetworkPolicy (it needs unrestricted egress to reach model APIs)

### Step 9 — Check for warnings and events

```bash
kubectl get events --sort-by='.lastTimestamp' | tail -30
kubectl describe languageagents
```

Flag any:
- `Warning` events (ImagePullBackOff, OOMKilled, Evicted, FailedScheduling)
- Failed reconciliation conditions on any CR
- Agent Workflows in `Failed` or `Error` phase (`kubectl get workflows`)
- CronWorkflows whose `status.lastScheduledTime` is stale relative to their schedule
- RBAC errors mentioning `workflowtaskresults` in agent pod logs

### Step 10 — Report findings

Produce a concise summary table:

| Resource | Name | Status | Notes |
|----------|------|--------|-------|
| Operator | language-operator | ✅/❌ | |
| LanguageCluster | ... | ✅/❌ | |
| Gateway | gateway | ✅/❌ | |
| LanguageModel | ... | ✅/❌ | |
| LanguageAgent | ... | ✅/❌ | |
| Pod | ... | ✅/❌ | |

Follow the table with a bulleted list of any anomalies found, ordered by severity. If everything looks healthy, say so clearly.
