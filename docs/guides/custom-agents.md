# Deploying Custom Agents

Any container image that follows the Language Operator agent contract can run as a `LanguageAgent`. This guide shows how to build a compliant image, package it as a `LanguageAgentRuntime`, and deploy it.

## The Agent Contract

The operator injects configuration into every agent pod. Your image must be ready to consume it.

### What the Operator Provides

| What | Where | Notes |
|---|---|---|
| Agent config | `/etc/agent/config.yaml` | Instructions, persona, tools, models |
| Persistent storage | `spec.workspace.mountPath` (default `/workspace`) | Survives pod restarts |
| LLM gateway URL | `MODEL_ENDPOINTS` env var | Single proxy URL for all models |
| Model names | `LLM_MODEL` env var | Comma-separated list |
| Tool URLs | `MCP_SERVERS` env var | Comma-separated MCP endpoint URLs |
| Identity | `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID` | Standard env vars |

See [Agent Runtime Container Specification](../architecture/agents.md) for the full contract including the `config.yaml` schema.

### What Your Image Must Do

1. **Listen on a port** — the operator creates a ClusterIP Service on `spec.port` (default `8080`). Your agent must bind to this port.
2. **Read `/etc/agent/config.yaml`** on startup (if present) to load instructions, personas, and tool endpoints.
3. **Route LLM traffic through `MODEL_ENDPOINTS`** — never call model APIs directly from inside the pod.
4. **Write persistent state to `/workspace`** — do not assume the local container filesystem survives restarts.

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
model_endpoint = os.environ.get("MODEL_ENDPOINTS", "")
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

## Troubleshooting

**Pod running but agent can't reach LLM:**
Verify `MODEL_ENDPOINTS` is set and the gateway is running:
```bash
kubectl exec deployment/my-agent -- env | grep MODEL_ENDPOINTS
kubectl get pods | grep gateway
```

**Config file missing:**
The config is mounted from a ConfigMap. Check the ConfigMap exists and the pod has it mounted:
```bash
kubectl get configmap my-agent-config
kubectl exec deployment/my-agent -- cat /etc/agent/config.yaml
```

**Port not accessible:**
Verify your container is actually listening on `spec.port`:
```bash
kubectl exec deployment/my-agent -- ss -tlnp | grep 8080
```

## Next Steps

- [Agent Runtime Container Specification](../architecture/agents.md) — full contract with config.yaml schema
- [Examples](../getting-started/examples.md) — multi-model, persona, and tool patterns
- [LanguageAgentRuntime reference](../api/languageagentruntime.md)
- [LanguageTool reference](../api/languagetool.md) — adding MCP tool servers
