# Language Operator

AI agents as first-class Kubernetes workloads.

Language Operator extends Kubernetes with purpose-built CRDs for deploying and operating AI agents like [OpenClaw](https://github.com/openclaw/openclaw) and [OpenCode](https://github.com/sst/opencode). Agents run as standard Deployments — managed by the control plane, observable with existing tooling, and configured through the same GitOps workflows as the rest of your infrastructure.

There is no proprietary runtime, no agent framework, and no code generation. Bring a container image; the operator handles the rest.

## How It Works

Declare what you want — the operator reconciles it:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  runtime: openclaw
  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
  models:
    - name: claude-sonnet
  tools:
    - name: python-executor
```

The operator creates the Deployment, Service, and NetworkPolicy, injects model endpoints and tool URLs, mounts persona and instruction config, and keeps everything reconciled as your cluster changes.

## Resources

| CRD | Purpose |
|-----|---------|
| `LanguageCluster` | Managed namespace; owns the shared model gateway |
| `LanguageAgent` | An agent workload — image, instructions, models, tools |
| `LanguageAgentRuntime` | Reusable defaults for a class of agents |
| `LanguageModel` | An LLM endpoint, proxied through LiteLLM |
| `LanguageTool` | An MCP-compatible tool server |
| `LanguagePersona` | Reusable behavioral configuration — tone, expertise, constraints |

## Project Status

**Pre-release** — not ready for production use.

## License

[MIT](https://github.com/language-operator/language-operator/blob/main/LICENSE)
