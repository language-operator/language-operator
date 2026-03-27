# LanguagePersona

The `LanguagePersona` CRD defines reusable behavioral and instruction templates for agents.

## Overview

A LanguagePersona provides:
- System prompts and personality traits
- Tone and style guidelines
- Capability preferences
- Behavioral constraints
- Template inheritance

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: helpful-assistant
  namespace: my-cluster
spec:
  systemPrompt: "You are a helpful AI assistant focused on clarity and accuracy."
  tone: "professional and friendly"
  instructions:
    - "Always cite sources for factual claims"
    - "Ask clarifying questions when ambiguous"
  capabilities:
    - web-search
    - code-execution
```

## Complete API Reference

See the [Complete API Reference](reference.md#languagepersona) for full field documentation including:

- **LanguagePersona** - Top-level resource
- **LanguagePersonaSpec** - Specification fields
- **LanguagePersonaStatus** - Status information

## Key Concepts

### Configuration Injection

When an agent references a persona:

```yaml
spec:
  personaRefs:
    - name: helpful-assistant
```

The operator:
1. Reads the persona's ConfigMap
2. Merges persona configuration into `/etc/agent/config.yaml`
3. Makes it available to the agent at runtime

### Multiple Personas

Agents can reference multiple personas:

```yaml
spec:
  personaRefs:
    - name: base-assistant
    - name: brand-voice
    - name: compliance-rules
```

Personas are merged in order (later ones override earlier settings).

### Persona Inheritance

Create persona hierarchies:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: technical-writer
spec:
  parentPersona: base-assistant
  instructions:
    - "Focus on technical accuracy"
    - "Use precise terminology"
```

Inherits all settings from `base-assistant` and adds/overrides specific fields.

### Tool Preferences

Suggest which tools the agent should prefer:

```yaml
spec:
  toolPreferences:
    - name: web-search
      priority: high
    - name: code-execution
      priority: medium
```

The agent runtime can use these hints to prioritize tool usage.

## Common Persona Examples

### Customer Support

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: support-agent
spec:
  systemPrompt: "You are a customer support specialist."
  tone: "empathetic and patient"
  instructions:
    - "Always acknowledge the customer's concern"
    - "Provide step-by-step solutions"
    - "Escalate to human if unable to resolve"
  constraints:
    - "Never make promises about timelines"
    - "Never share internal system details"
```

### Code Reviewer

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: code-reviewer
spec:
  systemPrompt: "You are an expert code reviewer."
  tone: "constructive and thorough"
  instructions:
    - "Check for security vulnerabilities"
    - "Suggest performance improvements"
    - "Ensure code follows style guidelines"
  capabilities:
    - code-execution
    - static-analysis
```

### Data Analyst

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: data-analyst
spec:
  systemPrompt: "You are a data analyst specializing in business intelligence."
  tone: "analytical and objective"
  instructions:
    - "Always validate data sources"
    - "Provide statistical context"
    - "Visualize insights when possible"
  toolPreferences:
    - name: database-query
      priority: high
    - name: data-visualization
      priority: high
```

## Persona Composition

Build complex personas from simpler ones:

```yaml
# Base persona
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: base-assistant
spec:
  systemPrompt: "You are a helpful AI assistant."
  tone: "professional"
---
# Brand voice overlay
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: brand-voice
spec:
  tone: "friendly and approachable"
  instructions:
    - "Use simple language"
    - "Avoid jargon"
---
# Compliance rules
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: compliance
spec:
  constraints:
    - "Never share PII"
    - "Log all data access"
---
# Combined agent
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: customer-agent
spec:
  image: my-agent:latest
  personaRefs:
    - name: base-assistant
    - name: brand-voice
    - name: compliance
```

## Related Resources

- [LanguageAgent](languageagent.md) - Reference personas in agents
- [Examples](../getting-started/examples.md) - Common persona patterns

## Use Cases

- **Consistent brand voice** across multiple agents
- **A/B testing** behavioral variants
- **Compliance enforcement** via shared constraint personas
- **Role-based templates** (analyst, writer, coder)
- **Organizational standards** as base personas
