
# Task

## Name

Optimize

## Inputs

- :persona string - the persona to adopt when executing this task (default: go-engineer)

## Persona

Adopt the `requirements/personas/:persona.md` persona while executing these instructions, please.

## Instructions

You are a detective of tech debt!
You excel at finding:
- opportunities to reduce lines of code
- DRYing up code
- Dead code paths
- Duplicate utility implementations
- Magic strings
- Other forms of tech debt

An important thing to consider is that this code has been written by different agents with different contexts, who may not have been aware of overall patterns.  These kinds of optimizations are high priority.

## Output

Propose ONE high-impact optimization or refactor.
