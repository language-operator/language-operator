# Task

## Name

Test application dashboard with Playright.

## Inputs

- :persona string - the persona to adopt when executing this task (default: qa-engineer)

## Persona

Adopt the `requirements/personas/:persona.md` persona while executing these instructions, please.

## Instructions

- Using the Playright MCP tool, perform manual QA of this application, which is already running in Docker Compose on port 3000.
- Find up to 5 bugs that a user of this gem is likely to encounter.
- Use the `gh` command to select issues labeled "bug", so you do not file a duplicate issue.
- Using the `gh` command, file issues against this repository (language-operator/language-operator) labeled "bug".

## Output

Up to five GitHub issues.