# Skills

Domain-specific knowledge modules for the Language Operator project.

## Available Skills

- [kubernetes-operator-patterns](kubernetes-operator-patterns/SKILL.md) - Go controller patterns, CRDs, RBAC, and operator development
- [nextjs-dashboard-patterns](nextjs-dashboard-patterns/SKILL.md) - React components, API routes, state management for the dashboard

## Skill Activation

Skills automatically activate based on:
- **Keywords** in your prompts (e.g., "controller", "CRD", "React component")
- **File patterns** when editing relevant files (e.g., `src/controllers/*.go`, `components/dashboard/src/**/*.tsx`)
- **Content patterns** when working with specific imports or code patterns

See [skill-rules.json](skill-rules.json) for complete trigger configuration.

## Usage

Skills can be invoked manually:
```
Use the kubernetes-operator-patterns skill to help me implement a new controller
```

Or they'll activate automatically when working on relevant code.