# Task

## Name

Test

## Inputs

- :persona string - the persona to adopt when executing this task (default: qa-engineer)

## Persona

Adopt the `requirements/personas/:persona.md` persona while executing these instructions, please.

## Instructions

- Use the `gh` command to select issues labeled "bug", so you do not file a duplicate issue.
- Ignore these folders:
  - requirements/
- Find a bug that a user of this gem is likely to encounter.
- Using the `gh` command, file a bug against this repository labeled "bug".

## Output

A GitHub issue ID.