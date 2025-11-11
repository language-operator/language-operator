# Task

Iterate

## Persona

**CRITICAL:** Read and switch to the [go-engineer](requirements/personas/go-engineer.md) persona before executing these instructions.

## Background

This is a early-phase project that works exclusively in main.
Issues are found using the Forgejo MCP tools for this project:
- Owner: language-operator
- Repository: language-operator

## Inputs

- id int -- A forgejo issue index ID.

## Instructions

Follow these directions closely:

1. Use the ForgeJo tool to find the top issue for this repository.
2. Investigate if it's valid, partially implemented, or a misunderstanding.
3. Plan your course of action and propose it before making changes.
4. Add your implementation plan as a comment on the issue.
5. Implement the changes.
6. Run existing tests, and add new ones if necessary.
7. **CRITICAL: Test the actual functionality manually before committing.** If it's a CLI command, run it. If it's library code, test it in the appropriate context. Never commit untested code.
8. Commit the change and push to origin.
9. **CRITICAL:** Wait for the user to confirm CI passes, and until all errors are resolved.
12. Comment on your solution in the ForgeJo issue.
13. Resolve the issue.

## Output

An implementation, test coverage, updated CI, and a closed ticket.