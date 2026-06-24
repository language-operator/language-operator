# Language Operator

**Deploy natural-language workloads to Kubernetes.**

Language Operator extends Kubernetes with purpose-built CRDs for AI agents. The workload is natural language — you describe what you want done, declare the models and tools the agent can use, and apply it like any other manifest. The operator runs it on a runtime you already know: [Claude Code](https://github.com/anthropics/claude-code), [OpenClaw](https://github.com/openclaw/openclaw), [OpenCode](https://github.com/sst/opencode), or [DeepAgents](https://github.com/langchain-ai/deepagents).

Each agent is an ordinary Deployment — scheduled by the control plane, observable with the tools you already run, and managed through the same GitOps workflows as the rest of your cluster. No new framework to adopt and no code generation: you write the intent, and the operator wires up models, tools, config, networking, and storage.

## Examples

Declare what you want in a manifest; the operator reconciles it into a Deployment, Service, and NetworkPolicy, injects the model and tool endpoints, mounts the instruction and persona config, and keeps it that way as your cluster changes.

### Data Analysis

A [deepagents](runtimes/deepagents.md) agent runs a task end to end — pull data, work the database, report out — then idles:

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

### Autonomous Development

A [Claude Code](runtimes/claude-code.md) agent with the repository cloned into its workspace and [Context7](guides/mcp-tools.md) for up-to-date library docs, given a standing task to work on its own:

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

### Live Coding

The same setup with no instructions — an interactive Claude Code terminal in the browser, repo and docs ready, for you to pair with:

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

### Development Teams

[examples/development-team](https://github.com/language-operator/language-operator/tree/main/examples/development-team) wires several of these together into a self-managing engineering team: a supervisor triages open issues into priority queues, and worker agents implement changes, run tests, and open pull requests — all in one namespace.

## Project Status

**Alpha** — APIs may change between releases.

## License

[MIT](https://github.com/language-operator/language-operator/blob/main/LICENSE)
