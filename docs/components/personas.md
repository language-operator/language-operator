# Personas

A `LanguagePersona` is a reusable behavioral template. It defines tone, personality, and expertise as free-text strings that the operator merges into an agent's `/etc/agent/config.yaml` at reconcile time.

## How It Works

A persona is a namespace-scoped resource with three string fields:

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

When an agent references a persona, the operator reads the persona resource and includes it under the `personas:` key in `/etc/agent/config.yaml`:

```yaml
personas:
  - name: data-analyst
    tone: "analytical and objective"
    personality: "precise, always validates data before drawing conclusions, surfaces uncertainty explicitly"
    expertise: "data analyst specialising in business intelligence and statistical reasoning"
```

The agent runtime is responsible for interpreting these fields. The operator does not apply them — it injects them.

## Referencing a Persona

An agent references a single persona by name:

```yaml
spec:
  persona: data-analyst
```

The persona must exist in the same namespace as the agent.

## Personas vs. Inline Instructions

`spec.instructions` and `spec.persona` serve different purposes:

| | `spec.instructions` | `spec.persona` |
|---|---|---|
| **Purpose** | What the agent does (task, goal) | How the agent behaves (style, character) |
| **Reuse** | Per-agent | Shared across agents |
| **Format** | Free text or markdown prompt | Structured tone/personality/expertise |

Use `instructions` for task-specific prompts. Use `persona` for organizational or role-level behavioral standards that apply across multiple agents.

```yaml
spec:
  persona: professional-tone       # shared behavior template
  instructions: |                  # task-specific prompt
    You are a quarterly report analyst. Summarize the attached CSV data
    into an executive brief with key metrics and recommended actions.
```

## Common Patterns

### Brand voice

Define a single persona for your organization's communication standard and reference it from every customer-facing agent:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: brand-voice
spec:
  tone: "warm, professional, never uses jargon"
  personality: "helpful and direct; acknowledges uncertainty rather than guessing"
  expertise: "general-purpose customer communications"
```

### Role-based templates

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

### Compliance constraints

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: compliance-baseline
spec:
  tone: "neutral and formal"
  personality: "never speculates; cites policy references when answering compliance questions; escalates ambiguous cases"
  expertise: "regulatory compliance in financial services"
```

## Related

- [LanguageAgent](../api/languageagent.md) — references personas via `spec.persona`
- [LanguagePersona API Reference](../api/languagepersona.md) — full field documentation
- [Agent Runtime Contract](agents.md) — `/etc/agent/config.yaml` format
