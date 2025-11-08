# Test Instructions

This directory contains example natural language instructions for bulk testing the synthesis engine.

## Structure

Instructions are organized by category to test different domains and patterns:

- **financial/** - Accounting, auditing, financial analysis tasks
- **devops/** - Monitoring, incident response, deployment tasks
- **communication/** - Email, Slack, calendar, meeting tasks
- **legal/** - Contract review, compliance, deadline tracking
- **executive/** - Briefings, competitive intelligence, metrics
- **support/** - Ticket triage, SLA monitoring, escalation
- **research/** - News aggregation, paper summaries, trends
- **edge-cases/** - Complex multi-tool workflows, error recovery
- **temporal-patterns/** - Testing schedule detection (hourly, daily, weekly, etc.)
- **persona-driven/** - Instructions that should trigger specific personas
- **synthesis-challenges/** - Intentionally difficult cases (ambiguous, underspecified)

## File Format

Each `.txt` file contains:
1. YAML frontmatter with metadata
2. Natural language instructions

Example:
```
---
category: financial
persona: financial-analyst
execution_mode: scheduled
expected_schedule: "0 16 * * *"
expected_tools: ["google-sheets", "email"]
difficulty: easy
description: Basic daily spreadsheet error checking
---

Review my recent changes in https://docs.google.com/spreadsheets/d/xyz
around 4pm every day, and let me know if I've made any mistakes before
I sign off for the day
```

## Testing

Run the synthesis test harness:
```bash
# Validate metadata only (fast)
ruby test/synthesis_test.rb --validate-only

# Run synthesis on specific category
ruby test/synthesis_test.rb financial

# Run synthesis on all instructions (slow - calls LLM for each)
ruby test/synthesis_test.rb
```

This will:
- Process all instruction files
- Run synthesis for each (or validate only)
- Generate .rb files alongside .txt files
- Validate generated DSL structure
- Generate JSON results report

### Known Issues

**Synthesis Timeouts**: The synthesis binary has a default HTTP client timeout that may be too short for slower LLM endpoints. If you see "context deadline exceeded" errors:
- Use a faster LLM endpoint
- Process instructions in smaller batches
- File an issue to add a configurable timeout flag to the synthesize binary
