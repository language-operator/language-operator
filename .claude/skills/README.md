# Skills

Domain-specific knowledge modules for the Language Operator project.

## Available Skills

- [kubernetes-operator-patterns](kubernetes-operator-patterns/SKILL.md) - Go controller patterns, CRDs, RBAC, and operator development

## Skill Activation

Skills automatically activate based on:
- **Keywords** in your prompts (e.g., "controller", "CRD", "reconcile")
- **File patterns** when editing relevant files (e.g., `src/controllers/*.go`)
- **Content patterns** when working with specific imports or code patterns

See [skill-rules.json](skill-rules.json) for complete trigger configuration.

## Usage

Skills can be invoked manually:
```
Use the kubernetes-operator-patterns skill to help me implement a new controller
```

Or they'll activate automatically when working on relevant code.