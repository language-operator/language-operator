# Deploying Custom Agents

Any container image that follows the Language Operator agent contract can run as a `LanguageAgent`. This guide shows how to build a compliant image, package it as a `LanguageAgentRuntime`, and deploy it.

## The Agent Contract

The operator injects configuration into every agent pod. Your image must be ready to consume it.

### What the Operator Provides

| What | Where | Notes |
|---|---|---|
| Agent config | `/etc/agent/config.yaml` | Instructions, persona, tools, models |
| Persistent storage | `spec.workspace.mountPath` (default `/workspace`) | Survives pod restarts and, in task mode, the boundary between runs |
| LLM gateway URL | `MODEL_ENDPOINT` env var | Single proxy URL for all models |
| Model names | `LLM_MODEL` env var | Comma-separated list |
| Tool URLs | `MCP_SERVERS` env var | Comma-separated MCP endpoint URLs |
| Identity | `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID` | Standard env vars |

See [Agent Runtime Container Specification](../components/agents.md) for the full contract including the `config.yaml` schema.

### What Your Image Must Do

1. **Read `/etc/agent/config.yaml`** on startup (if present) to load instructions, personas, and tool endpoints.
2. **Route LLM traffic through `MODEL_ENDPOINT`** — never call model APIs directly from inside the pod.
3. **Write persistent state to `/workspace`** — do not assume the local container filesystem survives restarts. In task mode every run is a brand-new pod, so anything not on the workspace volume is gone by the next run.
4. **Behave correctly for the execution mode it will be run in** — see below.

### Service mode vs task mode

Agents run as Argo Workflows, and `spec.execution.mode` decides what the operator expects
from your process. An image can support one mode or both; say which in your runtime's docs.

| | `service` (default) | `task` |
|---|---|---|
| Your process should | start, listen, and **keep running** | do the work and **exit** |
| Exit code | any exit is a crash; Argo restarts it | `0` → run `Succeeded`, non-zero → `Failed` |
| `spec.ports` | required to be reachable; the operator creates a Service and (with a cluster domain) an Ingress | **rejected** — a task agent is not addressable |
| Lifetime bound by | nothing; it runs until the agent is deleted or suspended | `spec.execution.activeDeadlineSeconds`, if set |
| Retries | unlimited, so the agent stays up | `spec.execution.retryLimit`, if set |

The most common mistake is a task-mode image that finishes its work and then idles. Argo has
no way to know the work is done, so the run never completes, the next scheduled fire is
skipped under the default `concurrencyPolicy: Forbid`, and the agent silently stops running.
**Exit when you are done.**

### ServiceAccount requirements

Agent pods run as the operator-managed ServiceAccount `language-agent-<agent-name>`, which is
granted `create` and `patch` on `argoproj.io/workflowtaskresults`. The Argo executor writes
one of those per node to report the outcome of a run.

If you point an agent at a ServiceAccount of your own with `spec.deployment.serviceAccountName`,
it **must** carry that permission or every run will fail at completion, no matter what your
image does:

```yaml
- apiGroups: ["argoproj.io"]
  resources: ["workflowtaskresults"]
  verbs: ["create", "patch"]
```

## Minimal Dockerfile

```dockerfile
FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY agent.py .

# The operator creates a Service on this port
EXPOSE 8080

CMD ["python", "agent.py"]
```

A minimal `agent.py` that reads operator-injected config:

```python
import os
import yaml
from http.server import HTTPServer, BaseHTTPRequestHandler

CONFIG_PATH = "/etc/agent/config.yaml"

def load_config():
    if os.path.exists(CONFIG_PATH):
        with open(CONFIG_PATH) as f:
            return yaml.safe_load(f)
    return {}

config = load_config()
model_endpoint = os.environ.get("MODEL_ENDPOINT", "")
models = os.environ.get("LLM_MODEL", "").split(",")

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")

HTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
```

## Creating a LanguageAgentRuntime

A `LanguageAgentRuntime` packages your image and its defaults so users can reference it by name. It is cluster-scoped — create it once, use it anywhere.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgentRuntime
metadata:
  name: my-agent
spec:
  image: ghcr.io/my-org/my-agent:latest
  ports:
    - name: http
      port: 8080
  workspace:
    size: 5Gi
    mountPath: /workspace
  deployment:
    resources:
      requests:
        memory: 256Mi
        cpu: 100m
      limits:
        memory: 1Gi
        cpu: 500m
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 30
EOF
```

Verify:

```bash
kubectl get languageagentruntimes
```

## Deploying the Agent

With the runtime in place, reference it from a `LanguageAgent`:

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  runtime: my-agent
  instructions: |
    You are a helpful assistant. Answer questions clearly and concisely.
  models:
    - name: claude-sonnet
EOF
```

Check status:

```bash
kubectl get languageagents
kubectl get pods -w
```

## Overriding Runtime Defaults

Agent fields always win over runtime defaults for scalar values. To override image or resources for a specific deployment:

```yaml
spec:
  runtime: my-agent
  image: ghcr.io/my-org/my-agent:v2.0.0   # overrides runtime image
  deployment:
    resources:
      limits:
        memory: 2Gi                         # overrides runtime resource limit
  models:
    - name: claude-sonnet
```

## Init Containers

For agents that need configuration pre-processing before startup (e.g. writing config to a non-standard path), add an init container in the runtime or agent spec:

```yaml
spec:
  runtime: my-agent
  deployment:
    initContainers:
      - name: seed-config
        image: ghcr.io/my-org/config-adapter:latest
        volumeMounts:
          - name: workspace
            mountPath: /workspace
          - name: agent-config
            mountPath: /etc/agent
            readOnly: true
```

The init container runs to completion before the main agent container starts. Both containers share the `workspace` volume.

