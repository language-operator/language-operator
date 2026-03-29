# LanguagePersona

The `LanguagePersona` CRD defines reusable behavioral templates for agents.

## Overview

A LanguagePersona configures:
- Tone and communication style
- Personality and character traits
- Domain expertise

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: helpful-assistant
  namespace: my-cluster
spec:
  tone: "professional and friendly"
  personality: "curious, patient, always explains reasoning step by step"
  expertise: "general-purpose assistant with strong research and writing skills"
```

## Complete API Reference

See the [Complete API Reference](reference.md#languagepersona) for full field documentation including:

- **LanguagePersona** - Top-level resource
- **LanguagePersonaSpec** - Specification fields
- **LanguagePersonaStatus** - Status information

## Spec Fields

| Field | Type | Description |
|-------|------|-------------|
| `tone` | string | Communication style, e.g. `"professional"`, `"concise and direct"` |
| `personality` | string | Character and behavioural traits, e.g. `"methodical, explains reasoning"` |
| `expertise` | string | Domain knowledge and skills, e.g. `"senior Go engineer specialising in distributed systems"` |

All fields are optional strings.

## Key Concepts

### Configuration Injection

When an agent references a persona:

```yaml
spec:
  persona: helpful-assistant
```

The operator reads the persona resource and merges its fields into `/etc/agent/config.yaml` under the `personas:` key:

```yaml
personas:
  - name: helpful-assistant
    tone: "professional and friendly"
    personality: "curious, patient, always explains reasoning step by step"
    expertise: "general-purpose assistant with strong research and writing skills"
```

### Referencing a Persona

Agents reference a single persona by name:

```yaml
spec:
  persona: base-assistant
```

## Common Persona Examples

### Customer Support

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: support-agent
spec:
  tone: "empathetic and patient"
  personality: "calm, solution-focused, always acknowledges the customer's concern before responding"
  expertise: "customer support specialist with knowledge of common product issues and escalation paths"
```

### Code Reviewer

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: code-reviewer
spec:
  tone: "constructive and thorough"
  personality: "detail-oriented, focuses on correctness and readability, suggests improvements rather than criticising"
  expertise: "senior software engineer experienced in security, performance, and style consistency"
```

### Data Analyst

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: data-analyst
spec:
  tone: "analytical and objective"
  personality: "precise, always validates data before drawing conclusions, surfaces uncertainty explicitly"
  expertise: "data analyst specialising in business intelligence and statistical reasoning"
```

## Related Resources

- [LanguageAgent](languageagent.md) - Reference personas in agents
- [Examples](../getting-started/examples.md) - Common persona patterns

## Use Cases

- **Consistent brand voice** across multiple agents
- **A/B testing** behavioral variants
- **Role-based templates** (analyst, writer, coder)
- **Organisational standards** as reusable base personas
