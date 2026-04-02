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

```bash
kubectl get deployments
kubectl get services
kubectl get pods
kubectl get configmaps
kubectl get secrets
kubectl get pvc
kubectl get networkpolicies
```

Cross-check:
- Each LanguageAgent has a matching Deployment, Service, ConfigMap (`{name}-agent`), and NetworkPolicy
- If `spec.openclaw.token` or `spec.opencode.password` was set, a `{name}-runtime` Secret should exist
- If `spec.workspace` is set, a PVC should exist
- Pod is Running (not just Pending or CrashLoopBackOff)

### Step 5 — Check the LiteLLM gateway

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

### Step 6 — Spot-check agent config injection

Pick the first Running agent pod and verify operator-injected env vars:

```bash
kubectl exec <pod> -- env | grep -E 'MODEL_ENDPOINTS|LLM_MODEL|MCP_SERVERS|AGENT_NAME|AGENT_NAMESPACE|AGENT_CLUSTER'
```

Cross-check against the LanguageAgent spec:
- `MODEL_ENDPOINTS` should point to `http://gateway.<namespace>.svc.cluster.local:8000`
- `LLM_MODEL` should list comma-separated model names matching `spec.models`
- `AGENT_NAME` should match the LanguageAgent name
- `MCP_SERVERS` should be present only if tools are referenced

Also check the mounted config file:
```bash
kubectl exec <pod> -- cat /etc/agent/config.yaml
```

Verify models, tools, and persona sections match what the CRs declare.

### Step 7 — Check NetworkPolicies

```bash
kubectl get networkpolicies -o yaml
```

Verify:
- The cluster-level `{cluster-name}-agents` NetworkPolicy selects pods with `langop.io/kind=LanguageAgent` (not the gateway)
- Each agent has its own NetworkPolicy allowing ingress on its port
- The gateway pod is NOT selected by the agents NetworkPolicy (it needs unrestricted egress to reach model APIs)

### Step 8 — Check for warnings and events

```bash
kubectl get events --sort-by='.lastTimestamp' | tail -30
kubectl describe languageagents
```

Flag any:
- `Warning` events (ImagePullBackOff, OOMKilled, Evicted, FailedScheduling)
- Failed reconciliation conditions on any CR
- Pods not matching their Deployment's desired replica count

### Step 9 — Report findings

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
