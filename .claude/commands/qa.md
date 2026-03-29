---
description: QA the dashboard with Playwright — find up to 5 bugs and file GitHub issues
---

## Inputs

- $PERSONA (optional, default: qa-engineer) — persona to adopt; definitions in `requirements/personas/`

## Directions

Adopt the $PERSONA persona from `requirements/personas/$PERSONA.md`.

Using the Playwright MCP tool, perform manual QA of the dashboard running in Docker Compose on port 3000:
1. Check existing `gh` issues labelled "bug" to avoid duplicates
2. Find up to 5 bugs a real user is likely to encounter
3. File each as a GitHub issue against `language-operator/language-operator` with label "bug"

## Output

Up to five filed GitHub issues.
