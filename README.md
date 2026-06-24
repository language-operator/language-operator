# Language Operator

**Deploy natural-language workloads to Kubernetes.**

Language Operator extends Kubernetes with purpose-built CRDs for AI agents. The workload is natural language -- you describe what you want done, declare the models and tools the agent can use, and apply it like any other manifest. The operator runs it on a runtime you already know: [Claude Code](https://github.com/anthropics/claude-code), [OpenClaw](https://github.com/openclaw/openclaw), [OpenCode](https://github.com/sst/opencode), or [DeepAgents](https://github.com/langchain-ai/deepagents).

Each agent is an ordinary Deployment scheduled by the control plane, observable with the tools you already run, and managed through the same GitOps workflows as the rest of your cluster. No new framework to adopt and no code generation: you write the intent, and the operator wires up models, tools, config, networking, and storage.

## Resources

Language Operator has purpose-built CRDs for agentic workloads:

| Resource | Status | Purpose |
|----------|--------|---------|
| [`LanguageCluster`](https://langop.io/docs/api/reference/#languagecluster) | Alpha | Configures a managed namespace for agents |
| [`LanguageAgent`](https://langop.io/docs/api/reference/#languageagent) | Alpha | Deploy a goal-directed agent |
| [`LanguageAgentRuntime`](https://langop.io/docs/api/reference/#languageagentruntime) | Alpha | Presets for popular runtimes (Claude Code, OpenClaw, OpenCode, DeepAgents) |
| [`LanguageModel`](https://langop.io/docs/api/reference/#languagemodel) | Alpha | Model configuration (proxied through LiteLLM) |
| [`LanguageTool`](https://langop.io/docs/api/reference/#languagetool) | Alpha | MCP-compatible tool server |
| [`LanguagePersona`](https://langop.io/docs/api/reference/#languagepersona) | Development | Reusable behaviors and expertise |

## Examples

Declare what you want in a manifest; the operator reconciles it into a Deployment, Service, and NetworkPolicy, injects the model and tool endpoints, mounts the instruction and persona config, and keeps it that way as your cluster changes.

### Data Analysis

A fully-autonomous [deepagents](https://langop.io/docs/runtimes/deepagents/) task for data analysis:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  runtime: deepagents
  instructions: |
    Analyze the orders database and email a summary of monthly revenue
    trends, the top ten customers, and any anomalies to analytics@example.com.

    Work read-only and keep it cheap: inspect the schema and indexes first,
    check each query with EXPLAIN, and avoid full table scans on orders.
  models:
    - name: claude-sonnet
  tools:
    - name: orders-postgres-db
    - name: email
```

### Live Coding

A [Claude Code](https://langop.io/docs/runtimes/claude-code/) agent with the repository cloned into its workspace and [Context7](https://langop.io/docs/guides/mcp-tools/) for up-to-date library docs:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: workstation
spec:
  runtime: claude-code
  repository:
    url: https://github.com/your-org/your-service
    secretRef:
      name: github-credentials
  tools:
    - name: context7
```

### Autonomous Development

Same as above, but with default instructions:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: maintainer
spec:
  runtime: claude-code
  repository:
    url: https://github.com/your-org/your-service
    secretRef:
      name: github-credentials
  instructions: |
    Triage open issues labeled "good first issue", implement the fix on a
    branch, run the tests, and open a pull request.
  tools:
    - name: context7
```

### Development Teams

[examples/development-team](https://github.com/language-operator/language-operator/tree/main/examples/development-team) wires several of these together into a self-managing engineering team: a supervisor triages open issues into priority queues, and worker agents implement changes, run tests, and open pull requests — all in one namespace.

## Getting Started

Follow the [installation guide](https://langop.io/docs/getting-started/installation/) and [quick start](https://langop.io/docs/getting-started/quickstart/) to get started.

## Development

See the [development setup guide](docs/development/setup.md) for full instructions.

## Status

**Alpha** — APIs may change between releases.

## License

[MIT](LICENSE)
