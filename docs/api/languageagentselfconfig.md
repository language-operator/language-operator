# LanguageAgentSelfConfig

The `LanguageAgentSelfConfig` CRD (`lasc`) allows an agent pod to request runtime modifications to its own `LanguageAgent` spec. The operator validates each request against the parent agent's `spec.selfConfigure` allowlist before applying the patch.

## Overview

Self-config requests are short-lived, namespace-scoped resources. The controller processes them once and transitions them to a terminal phase (`Applied`, `Denied`, or `Failed`). Completed requests are automatically deleted one hour after reaching a terminal phase.

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentSelfConfig
metadata:
  name: add-web-search
  namespace: my-agents
spec:
  instanceRef: my-agent
  addTools:
    - web-search
```

The parent agent must have `spec.selfConfigure.enabled: true` and `tools` in `spec.selfConfigure.allowedActions` for this request to be applied.

## Complete API Reference

### `spec` fields (`LanguageAgentSelfConfigSpec`)

| Field | Type | Description |
|-------|------|-------------|
| `instanceRef` | string | Name of the `LanguageAgent` to modify. Must be in the same namespace. **Required.** |
| `addTools` | []string | `LanguageTool` names to append to `spec.tools` on the parent agent. Requires `tools` in `allowedActions`. |
| `removeTools` | []string | `LanguageTool` names to remove from `spec.tools` on the parent agent. Requires `tools` in `allowedActions`. |
| `addModels` | []string | `LanguageModel` names to append to `spec.models` on the parent agent. Requires `models` in `allowedActions`. |
| `removeModels` | []string | `LanguageModel` names to remove from `spec.models` on the parent agent. Requires `models` in `allowedActions`. |
| `addEnvVars` | []SelfConfigEnvVar | Plain-value environment variables to inject into `spec.deployment.env`. Existing variables with the same name are overwritten. SecretKeyRef and other value sources are not supported. Requires `envVars` in `allowedActions`. |
| `updateInstructions` | string | When non-empty, replaces `spec.instructions` on the parent agent. Requires `instructions` in `allowedActions`. |
| `addRoleRules` | []PolicyRule | RBAC policy rules to append to `spec.deployment.roleRules`. Duplicate rules are de-duplicated. Requires `roleRules` in `allowedActions`. |

### `spec.addEnvVars[]` fields (`SelfConfigEnvVar`)

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Environment variable name. **Required.** |
| `value` | string | Literal string value. **Required.** |

### `status` fields (`LanguageAgentSelfConfigStatus`)

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current processing state: `Pending`, `Applied`, `Failed`, or `Denied`. |
| `message` | string | Human-readable explanation of the current phase. |
| `completionTime` | time | Set when the request reaches a terminal phase. The CR is deleted 1 hour after this time. |

## Phase Lifecycle

```
Pending → Applied   (changes patched onto parent LanguageAgent)
        → Denied    (selfConfigure not enabled, or action not in allowedActions)
        → Failed    (controller error while processing the request)
```

## Enabling Self-Configuration

The parent agent must opt in via `spec.selfConfigure`:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  selfConfigure:
    enabled: true
    allowedActions:
      - tools      # addTools / removeTools
      - models     # addModels / removeModels
      - envVars    # addEnvVars
      - instructions  # updateInstructions
      - roleRules  # addRoleRules
```

When `enabled` is `true`, the operator grants the agent's ServiceAccount permission to create `LanguageAgentSelfConfig` resources in the namespace.

When `allowedActions` is empty (while `enabled: true`), all requests are immediately denied.

## Related Resources

- [LanguageAgent](languageagent.md) — parent resource; `spec.selfConfigure` controls the gate
- [CRD Overview](overview.md) — all Language Operator CRDs
