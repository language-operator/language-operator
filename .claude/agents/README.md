# Agents

Specialized AI workflows for repetitive tasks in the Language Operator project.

## Available Agents

- [changelog-writer](changelog-writer.md) - Creates user-focused changelog entries from git diffs and PR descriptions

## Usage

Agents can be invoked with:
```
Use the changelog-writer agent to create a changelog entry for the recent UI improvements
```

## Agent Development

Agents are defined as Markdown files with YAML frontmatter containing:
- **name**: Agent identifier
- **description**: What the agent does and when to use it
- **model**: Model preference (inherit, sonnet, opus, haiku)
- **color**: Visual identifier in UI

Each agent contains step-by-step processes, quality standards, and output formatting guidelines.