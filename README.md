# Language Operator

**Stop writing code to automate your work. Just describe what you want done.**

Language Operator is a Kubernetes operator that turns natural language into autonomous agents. Describe your task in plain English, and the system synthesizes the code, deploys the agent, and executes it—autonomously, on schedule, forever.

```bash
aictl agent create "review my spreadsheet at 4pm daily and email me any errors"
```

That's it. No YAML. No scripting. No infrastructure. Just tell it what you want.

---

## Why This Exists

Knowledge workers waste hours on repetitive tasks:

* Accountants review the same spreadsheets every day
* DevOps engineers check the same dashboards every hour
* Lawyers scan inboxes for urgent client emails
* Executives need meeting prep that never changes format

**These tasks are repetitive, rule-based, and soul-crushing.**

Language Operator frees you from this. Describe the task once, and an autonomous agent handles it forever.

---

## Natural Language → Code → Execution

Traditional automation requires you to be a programmer. Language Operator doesn't.

**You write this:**
```
"Every morning at 9am, check my inbox and email me a list of urgent messages"
```

**The operator synthesizes this:**
```ruby
agent "inbox-triage" do
  schedule "0 9 * * *"

  workflow do
    step :fetch_emails, tool: "gmail"
    step :categorize, analyze: "urgency and importance"
    step :notify, tool: "email", condition: "urgent_found"
  end
end
```

**The agent executes it:**
```
09:00:01 | Starting execution
09:00:02 | Fetching unread emails (23 found)
09:00:05 | Analyzing urgency... 3 urgent
09:00:06 | Sending notification
09:00:07 | Complete
```

No code. No deploy pipeline. No debugging. **It just works.**

---

## Real Examples

### Accountant: Daily Spreadsheet Review
```bash
aictl agent create "review my recent changes in https://docs.google.com/spreadsheets/d/xyz \
  at 4pm every day and let me know if I've made any mistakes before I sign off"
```

**Result:** Agent runs daily at 4pm, analyzes spreadsheet changes, emails you if it finds errors.

### DevOps: Health Monitoring
```bash
aictl agent create "check https://api.example.com/health every 5 minutes \
  and page me if status isn't 200"
```

**Result:** Agent monitors your API, sends PagerDuty alert on failure.

### Executive: Meeting Prep
```bash
aictl agent create "email me a summary of tomorrow's meetings every evening at 6pm"
```

**Result:** Agent fetches calendar, generates prep summary, emails you daily.

### Lawyer: Client Intake
```bash
aictl agent create "when someone emails info@law.com, create a ticket in our CRM \
  and send an auto-reply acknowledging receipt"
```

**Result:** Agent monitors inbox, auto-responds to clients, logs everything.

---

## How It Works

### Three Layers of Intelligence

```
┌──────────────────────────────────────────────┐
│  Natural Language Interface                 │
│  "review my spreadsheet at 4pm daily..."    │
└──────────────────────────────────────────────┘
                    ↓
┌──────────────────────────────────────────────┐
│  Synthesis Engine (LLM-Powered)             │
│  Instructions → Ruby DSL Code                │
└──────────────────────────────────────────────┘
                    ↓
┌──────────────────────────────────────────────┐
│  Autonomous Execution                        │
│  Schedule → Execute → Report                 │
└──────────────────────────────────────────────┘
```

**Layer 1: You speak, it understands**
The CLI parses your natural language and creates a Kubernetes resource with your instructions.

**Layer 2: The operator synthesizes behavior**
An LLM reads your instructions, generates executable Ruby code, validates it, and deploys it.

**Layer 3: The agent executes autonomously**
Your agent runs on schedule, uses tools, makes decisions, and reports results.

### The Innovation: Behavior Synthesis

Traditional operators reconcile **configuration**.
Language Operator reconciles **behavior**.

When you change instructions, the operator:
1. Calls an LLM with your new description
2. Generates new executable code
3. Validates and deploys it
4. The agent immediately adopts the new behavior

**Your agent's behavior is synthesized on-demand from natural language.**

---

## Core Concepts

### Agents
Autonomous programs that execute tasks based on natural language instructions.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  instructions: |
    Check my inbox every hour and categorize emails by urgency
```

That's all you write. The operator handles the rest.

### Tools (MCP Servers)
Capabilities your agents can use: email, web search, spreadsheets, APIs, databases.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: gmail
spec:
  image: git.theryans.io/language-operator/gmail-tool:latest
  deploymentMode: sidecar
```

Tools are MCP servers that agents call to interact with the world.

### Models
LLM configurations that power agent intelligence.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude
spec:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
```

Use any model: Claude, GPT-4, local LLMs, custom endpoints.

### Personas
Reusable personality templates that define how agents behave.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: financial-analyst
spec:
  systemPrompt: |
    You are a meticulous financial analyst who reviews data
    for errors, inconsistencies, and anomalies...
```

Encode professional expertise once, reuse across agents.

### Clusters
Network-isolated environments for organizing agents and tools.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: finance-team
  namespace: production-agents
spec: {}
```

---

## Powerful CLI

Inspired by other modern, familiar command line tools:

### Create an Agent
```bash
$ aictl agent create "scan my inbox every morning at 9am and categorize emails"

Creating agent...
✓ Agent 'inbox-scanner' created
✓ Synthesizing code... (took 3.2s)
✓ Agent deployed and ready

Schedule: Daily at 9:00 AM (0 9 * * *)
Next run: Tomorrow at 9:00 AM (in 14h 23m)
Tools:    gmail, email
Persona:  general-assistant

View logs: aictl agent logs inbox-scanner -f
```

### Watch It Work
```bash
$ aictl agent logs inbox-scanner -f

09:00:01 | Starting execution cycle 12
09:00:02 | Loading persona: general-assistant
09:00:03 | Connecting to tool: gmail
09:00:05 | Fetching unread emails (47 found)
09:00:08 | Categorizing by urgency...
09:00:11 | Applied 47 labels (12 urgent, 25 normal, 10 low)
09:00:12 | Sending summary email
09:00:13 | Execution complete (success, 12.4s)
```

### See Everything
```bash
$ aictl status

Language Operator Status
========================

Cluster:     Connected (k3s v1.28)
Operator:    Running (v0.2.0)

Agents:      8 running, 2 ready, 1 failed
Tools:       12 installed, 11 connected
Models:      3 configured, 2 in use
Personas:    9 available (5 built-in, 4 custom)

Recent Activity:
  2m ago   inbox-scanner        Success (12.4s)
  15m ago  error-monitor        Alert sent
  1h ago   spreadsheet-reviewer Success (8.1s)
```

---

## Workspace: Shared Memory

Agents and tools share a persistent workspace for coordination and state.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: news-summarizer
spec:
  instructions: |
    Fetch top Hacker News posts, summarize them, and email me
  workspace:
    enabled: true
    size: 10Gi
```

**The agent writes:**
```bash
echo "2025-11-06: Sent summary" >> /workspace/history.log
```

**The tool reads:**
```bash
cat /workspace/history.log
```

**Next run, the agent remembers:**
```bash
# Check what we did yesterday
last_run=$(tail -1 /workspace/history.log)
```

Workspaces enable agents to:
- Remember past actions
- Coordinate with tools
- Build long-term knowledge
- Debug execution history

---

## Network Isolation by Default

Every cluster is network-isolated. Resources can talk to each other but not the internet—unless you explicitly allow it.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
spec:
  image: git.theryans.io/language-operator/web-tool:latest
  egress:
  - description: Allow DuckDuckGo search
    to:
      dns:
      - "*.duckduckgo.com"
    ports:
    - port: 443
      protocol: TCP
```

**Zero-trust by default. Explicit allowlist for everything.**

> **Note:** Network isolation uses Kubernetes NetworkPolicy. Enforcement requires a compatible CNI plugin (Cilium, Calico, Weave Net, or Antrea). The default k3s CNI (Flannel) does not enforce NetworkPolicy. The operator will create NetworkPolicy resources, but they may be silently ignored if your CNI doesn't support them. Check your agent status for `NetworkPolicyEnforced` condition.

---

## Getting Started

### 1. Install the Operator

```bash
helm install language-operator oci://git.theryans.io/helm/language-operator
```

### 2. Create Your First Agent

```bash
aictl agent create "send me a summary of my inbox every morning at 9am"
```

### 3. Watch It Work

```bash
aictl agent logs inbox-summarizer -f
```

**That's it. You're automating.**

---

## Why Kubernetes?

Because your automation should be:
- **Durable**: Survives restarts, crashes, cluster failures
- **Scalable**: Runs 1 agent or 10,000
- **Observable**: Full logging, metrics, traces
- **Declarative**: Version control your agents
- **Production-ready**: Built on battle-tested infrastructure

Kubernetes isn't overkill. It's **exactly what you need** when automation matters.

---

## Project Status

**Current Version:** 0.2.0 (Alpha / v1alpha1 CRDs)

### Current Features (v0.2 / v1alpha1)

Production-ready features available now:

- ✅ **Natural language agent synthesis** - Describe tasks in plain English, operator generates executable code
- ✅ **Autonomous agent execution** - Agents run on schedules, use tools, make decisions independently
- ✅ **MCP tool integration** - Standard tool protocol for extensibility
- ✅ **CLI (`aictl`)** - Command-line interface for agent management (via language-operator gem)
- ✅ **AST-based security validation** - CVE-001 mitigated via Ruby AST parsing (see [ADR 001](docs/adr/001-ast-based-ruby-validation.md))
- ✅ **NetworkPolicy enforcement with CNI detection** - CVE-002 mitigated via Cilium/Calico/Weave/Antrea validation
- ✅ **Container registry whitelist** - CVE-003 mitigated via configurable allowed registries
- ✅ **Resource limits and security contexts** - Non-root execution, read-only filesystems, resource quotas
- ✅ **Workspace sharing** - Persistent storage for agent state and coordination
- ✅ **Persona system** - Reusable personality templates for agents
- ✅ **Five CRDs**: LanguageAgent, LanguageTool, LanguageModel, LanguagePersona, LanguageCluster

### Planned Features (Roadmap)

Features under development:

- ⏳ **Multi-agent workflows** - Coordinate multiple agents on complex tasks
- ⏳ **Advanced scheduling** - Conditional triggers, event-driven execution
- ⏳ **Dashboard** - Web UI for viewing cluster state

---

## Security

Language Operator implements defense-in-depth security with multiple layers of protection:

- **AST-based code validation** - Blocks dangerous Ruby code patterns at synthesis and runtime
- **NetworkPolicy enforcement** - Zero-trust networking with CNI compatibility detection
- **Registry whitelist** - Only approved container registries allowed
- **Non-root execution** - All containers run as unprivileged user
- **Read-only filesystems** - Root filesystem is read-only by default
- **Resource limits** - CPU and memory quotas enforced

**For detailed security documentation, see:**
- [Security Overview](docs/security/README.md) - Security model and features
- [CVE Mitigations](docs/security/cve-mitigations.md) - Attack scenarios and defenses
- [CNI Requirements](docs/security/cni-requirements.md) - NetworkPolicy enforcement setup
- [Registry Whitelist](docs/security/registry-whitelist.md) - Container registry configuration

---

## Philosophy

**1. Natural language is the interface**
No one should need to learn YAML or DSLs. Describe what you want. Done.

**2. Synthesis beats templates**
Don't force users into predefined patterns. Generate the right code for their specific need.

**3. Autonomous by default**
Set it and forget it. Agents should run forever without babysitting.

**4. Kubernetes-native**
Don't fight the platform. Embrace CRDs, controllers, and declarative config.

**5. Beautiful UX matters**
If it's not delightful to use, we failed.

---

## Inspiration

**Cilium** - showed us how beautiful Kubernetes UX can be
**Temporal** - proved workflow-as-code works at scale
**MCP** - gave us a standard for tool integration
**Kubernetes Operators** - taught us how to reconcile desired state

We combined the best ideas and made something new.

---

## Contributing

We're building this in the open. Join us.

**Build tools:**
MCP servers that connect to your systems. See [language-tools](https://github.com/language-operator/language-tools).

**Build personas:**
Professional templates that encode expertise. See [examples/personas](examples/personas).

**Improve synthesis:**
Better prompts, better code generation, better validation.

**Join the community:**
- GitHub: [language-operator/language-operator](https://github.com/language-operator/language-operator)
- Issues: [Report bugs, request features](https://github.com/language-operator/language-operator/issues)
- Discussions: [Ask questions, share agents](https://github.com/language-operator/language-operator/discussions)

---

## License

MIT License - see [LICENSE](LICENSE)

---

## The Future We're Building

Imagine a world where:
- Accountants review 1,000 spreadsheets with zero manual work
- DevOps teams respond to incidents before humans notice
- Lawyers never miss an urgent client email
- Every professional has a team of tireless assistants

**That's the future. And it starts with describing what you want done.**

```bash
aictl agent create "..."
```

**Welcome to automation without code.**
