# Language Operator

A Kubernetes operator that synthesizes autonomous agents from natural language descriptions.

## What It Does

Language Operator converts natural language goals into executable agents that run in your Kubernetes cluster:

```bash
aictl agent create "monitor our API error rates and alert on spikes"
```

This creates a complete agent with:
- Code synthesized from your description
- Kubernetes deployment (pod, service, network policies)
- Observability integration (OpenTelemetry traces)
- Security isolation (AST validation, network policies)

## How It Works

**1. Natural Language → Code**

The operator calls an LLM to generate Ruby code from your instructions. You can use cloud models (GPT-4, Claude) or local quantized models (Llama, Mistral).

**2. Organic Functions**

Agents are composed of tasks with stable input/output contracts. Tasks can be:
- **Neural**: LLM decides implementation at runtime
- **Symbolic**: Explicit Ruby code

The caller doesn't know which type it's calling - the contract is the interface.

**3. Progressive Optimization**

After execution, the system analyzes OpenTelemetry traces to detect patterns. Deterministic neural tasks are automatically converted to symbolic code, reducing cost and latency while preserving the contract.

## Example

**Initial (fully neural):**
```ruby
task :check_api,
  instructions: "Check API health",
  outputs: { status: 'string' }

task :send_alert,
  instructions: "Send alert if unhealthy",
  inputs: { status: 'string' },
  outputs: { sent: 'boolean' }

main do
  result = execute_task(:check_api)
  execute_task(:send_alert, inputs: result)
end
```

**After learning (hybrid):**
```ruby
task :check_api,
  outputs: { status: 'string' }
do |inputs|
  execute_tool('http', 'get', url: 'https://api.example.com/health')
end

task :send_alert,  # Kept neural - decision logic varies
  instructions: "Send alert if unhealthy",
  inputs: { status: 'string' },
  outputs: { sent: 'boolean' }

main do  # Unchanged
  result = execute_task(:check_api)
  execute_task(:send_alert, inputs: result)
end
```

The `main` block never changes. Implementations evolve without breaking callers.

## Installation

```bash
# Add Helm repository
helm repo add language-operator https://charts.langop.io

# Install operator
helm install language-operator language-operator/language-operator

# Install aictl
gem install language-operator

# Set up your first cluster
aictl quickstart
```

## Requirements

- Kubernetes 1.26+
- NetworkPolicy-capable CNI (Cilium, Calico, Weave, Antrea)
- Optional: GPU nodes for local model inference

## Status

**Alpha** - Core functionality works but not yet recommended for production use.

What's implemented:
- ✅ Natural language synthesis
- ✅ Neural and symbolic task execution
- ✅ Progressive optimization via trace analysis
- ✅ Security validation (AST, NetworkPolicy, registry)
- ✅ OpenTelemetry integration
- ✅ ConfigMap versioning and rollback

## License

[FSL 1.1](LICENSE) - Converts to Apache 2.0 on 2028-01-01

**Use Limitation**: Cannot offer Language Operator as a commercial managed service until 2028. Internal use, consulting, and custom deployments are permitted.