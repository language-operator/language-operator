# OpenTelemetry Semantic Conventions

This document defines the custom semantic conventions used in language-operator traces. These conventions follow OpenTelemetry best practices and provide consistent attribute naming across all instrumented operations.

## Overview

The language-operator uses OpenTelemetry distributed tracing to provide deep visibility into:
- Agent lifecycle operations (creation, synthesis, deployment)
- Code synthesis and validation
- Self-healing operations (failure detection and recovery)
- Controller reconciliation loops

All traces use the service name `language-operator` with versioning information included as resource attributes.

## Span Names

### Agent Controller Spans

#### `agent.reconcile`
**Description**: Root span for the LanguageAgent controller reconciliation loop

**When Created**: Every time the controller processes a LanguageAgent resource

**Parent**: None (root span for agent operations)

**Attributes**:
- `agent.name` (string): Name of the LanguageAgent resource
- `agent.namespace` (string): Kubernetes namespace of the LanguageAgent
- `agent.mode` (string): Execution mode (`deployment`, `cronjob`, etc.)
- `agent.generation` (int64): Resource generation number

---

#### `agent.synthesize`
**Description**: Code synthesis operation for generating agent DSL code

**When Created**: When the operator needs to generate or regenerate agent code

**Parent**: `agent.reconcile`

**Attributes**:
- `synthesis.agent_name` (string): Name of the agent being synthesized
- `synthesis.namespace` (string): Namespace of the agent
- `synthesis.tools_count` (int): Number of tools available to the agent
- `synthesis.models_count` (int): Number of models available to the agent
- `synthesis.is_retry` (bool): Whether this is a retry attempt
- `synthesis.attempt` (int): Synthesis attempt number (1-based)
- `synthesis.code_length` (int): Length of synthesized code in characters
- `synthesis.duration_seconds` (float64): Duration of synthesis operation

---

#### `agent.self_healing.detect`
**Description**: Pod failure detection for self-healing

**When Created**: When the controller checks for failed agent pods

**Parent**: `agent.reconcile`

**Attributes**:
- `agent.name` (string): Name of the LanguageAgent
- `agent.namespace` (string): Namespace of the LanguageAgent
- `agent.pod_failures` (int): Number of failed pods detected
- `agent.error_patterns` (string[]): List of error messages extracted from failed pods

---

#### `agent.self_healing.synthesize`
**Description**: Self-healing synthesis with error context

**When Created**: When the operator attempts to fix agent code based on runtime failures

**Parent**: `agent.reconcile`

**Attributes**:
- `agent.name` (string): Name of the LanguageAgent
- `agent.namespace` (string): Namespace of the LanguageAgent
- `self_healing.attempt_number` (int): Self-healing attempt number
- `self_healing.error_context` (string): First error message from failure context
- `self_healing.runtime_errors_count` (int): Number of runtime errors in context
- `self_healing.validation_errors_count` (int): Number of validation errors in context
- `synthesis.duration_seconds` (float64): Duration of synthesis operation
- `synthesis.code_length` (int): Length of synthesized code

---

### Synthesis Package Spans

#### `synthesis.agent.generate`
**Description**: Core LLM-based code generation

**When Created**: When synthesizing agent DSL code using the LLM

**Parent**: `agent.synthesize` or `agent.self_healing.synthesize`

**Attributes**:
- `synthesis.agent_name` (string): Name of the agent
- `synthesis.namespace` (string): Namespace of the agent
- `synthesis.tools_count` (int): Number of available tools
- `synthesis.models_count` (int): Number of available models
- `synthesis.is_retry` (bool): Whether this is a retry
- `synthesis.attempt_number` (int): Attempt number for this synthesis
- `synthesis.input_tokens` (int64): Number of input tokens sent to LLM
- `synthesis.output_tokens` (int64): Number of output tokens from LLM
- `synthesis.cost_usd` (float64): Total cost in USD
- `synthesis.model` (string): Model name used for synthesis
- `synthesis.code_length` (int): Length of generated code
- `synthesis.duration_seconds` (float64): Duration of generation

---

#### `synthesis.validate`
**Description**: Validation of synthesized DSL code

**When Created**: After code generation, to validate syntax and security

**Parent**: `synthesis.agent.generate`

**Attributes**:
- `validation.language` (string): Language being validated (always "ruby")
- `validation.code_length` (int): Length of code being validated
- `validation.error_type` (string): Type of validation error (if failed)
  - Values: `empty_code`, `missing_agent`, `missing_require`, `security_violation`
- `validation.result` (string): Validation result (always "success" for successful spans)

---

## Attribute Reference

### Agent Attributes (`agent.*`)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `agent.name` | string | Name of the LanguageAgent resource | `"email-responder"` |
| `agent.namespace` | string | Kubernetes namespace | `"default"` |
| `agent.mode` | string | Execution mode | `"deployment"`, `"cronjob"` |
| `agent.generation` | int64 | Kubernetes resource generation | `3` |
| `agent.pod_failures` | int | Number of failed pods detected | `2` |
| `agent.error_patterns` | string[] | Error messages from failed pods | `["undefined method 'call'", "NoMethodError"]` |

### Synthesis Attributes (`synthesis.*`)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `synthesis.agent_name` | string | Agent name for synthesis | `"email-responder"` |
| `synthesis.namespace` | string | Agent namespace | `"default"` |
| `synthesis.tools_count` | int | Number of available tools | `3` |
| `synthesis.models_count` | int | Number of available models | `2` |
| `synthesis.is_retry` | bool | Whether this is a retry | `true` |
| `synthesis.attempt` | int | Synthesis attempt number | `2` |
| `synthesis.attempt_number` | int | Attempt number (alternate name) | `2` |
| `synthesis.input_tokens` | int64 | LLM input tokens | `1024` |
| `synthesis.output_tokens` | int64 | LLM output tokens | `512` |
| `synthesis.cost_usd` | float64 | Total cost in USD | `0.0034` |
| `synthesis.model` | string | Model used for synthesis | `"mistralai/magistral-small-2509"` |
| `synthesis.code_length` | int | Generated code length | `2048` |
| `synthesis.duration_seconds` | float64 | Duration in seconds | `2.453` |

### Self-Healing Attributes (`self_healing.*`)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `self_healing.attempt_number` | int | Self-healing attempt number | `1` |
| `self_healing.error_context` | string | First error message | `"undefined method 'execute_tool'"` |
| `self_healing.runtime_errors_count` | int | Number of runtime errors | `3` |
| `self_healing.validation_errors_count` | int | Number of validation errors | `0` |

### Validation Attributes (`validation.*`)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `validation.language` | string | Language being validated | `"ruby"` |
| `validation.code_length` | int | Code length in characters | `2048` |
| `validation.error_type` | string | Type of validation error | `"security_violation"` |
| `validation.result` | string | Validation result | `"success"` |

## Span Status Codes

The operator uses OpenTelemetry status codes to indicate operation outcomes:

- **`Ok`**: Operation completed successfully
- **`Error`**: Operation failed (check span events and attributes for details)

Common error status descriptions:
- `"Failed to get LanguageAgent"`: Resource retrieval failed
- `"Failed to add finalizer"`: Finalizer update failed
- `"Image registry validation failed"`: Image not from allowed registry
- `"LLM call failed"`: Synthesis LLM request failed
- `"Validation failed"`: Code validation failed
- `"Failed to list pods"`: Pod listing failed for failure detection

## Span Events

Spans may include events to mark significant moments:

| Event Name | Span | Description |
|------------|------|-------------|
| `"Deleting agent"` | `agent.reconcile` | Agent deletion initiated |

## Resource Attributes

All spans include these resource-level attributes:

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `service.name` | string | Service name | `"language-operator"` |
| `service.version` | string | Operator version | `"v0.1.0"` or `"dev"` |
| `k8s.namespace.name` | string | Kubernetes namespace (operator pod) | `"kube-system"` |
| `k8s.cluster.name` | string | Kubernetes cluster name (if configured) | `"production-cluster"` |

Resource attributes are set via:
- `service.name` and `service.version`: Automatically from build info
- `k8s.namespace.name`: From `POD_NAMESPACE` environment variable
- Custom attributes: Via `OTEL_RESOURCE_ATTRIBUTES` environment variable

## Trace Context Propagation

The operator automatically propagates trace context:
- Within the same reconciliation loop (parent-child relationships)
- Between controller operations and synthesis operations
- From synthesis to validation

This allows complete end-to-end tracing of agent lifecycle operations.

## Cost Tracking

Synthesis operations include cost tracking attributes when a cost tracker is configured:

- `synthesis.input_tokens`: Tokens sent to the LLM
- `synthesis.output_tokens`: Tokens received from the LLM
- `synthesis.cost_usd`: Calculated cost in USD
- `synthesis.model`: Model name for cost calculation

These attributes enable:
- Budget tracking per agent
- Cost analysis across namespaces
- Identification of expensive synthesis operations
- Optimization opportunities

## Best Practices

### Querying by Agent

To find all operations for a specific agent:
```
agent.name="email-responder"
```

### Finding Expensive Operations

To find synthesis operations that cost more than $0.10:
```
synthesis.cost_usd > 0.10
```

### Tracing Self-Healing

To trace complete self-healing cycles:
```
span.name="agent.self_healing.synthesize" OR span.name="agent.self_healing.detect"
```

### Finding Failures

To find failed operations:
```
status.code="ERROR"
```

### Analyzing Performance

To find slow synthesis operations (>10 seconds):
```
synthesis.duration_seconds > 10
```

## Implementation Notes

### Tracer Initialization

The operator creates tracers per package:
- `language-operator/agent-controller`: Agent controller operations
- `language-operator/synthesizer`: Synthesis and validation operations

### Conditional Instrumentation

OpenTelemetry is only initialized when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. Without this configuration, tracing has zero overhead.

### Error Recording

All errors are recorded on spans using `span.RecordError(err)` before setting error status codes.

### Attribute Naming

Attributes follow OpenTelemetry conventions:
- Namespaced by domain (`agent.*`, `synthesis.*`, `validation.*`)
- Use snake_case for multi-word attributes
- Use descriptive, unambiguous names
- Include units in names when applicable (`_seconds`, `_count`, `_usd`)
