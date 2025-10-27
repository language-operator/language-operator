# Based MCP Headless Agent

Autonomous agent that runs in a continuous loop to achieve specified goals.

## Features

- **Autonomous Execution**: Runs without human intervention
- **Goal-Oriented**: Works towards completing specified objectives
- **Tool Usage**: Automatically uses MCP tools to accomplish tasks
- **Progress Tracking**: Monitors completion and iteration limits
- **Auto-Termination**: Exits when goals complete or limits reached

## Usage

### With Docker Compose

```bash
# From repository root
docker compose up agent-headless

# Or with profile
docker compose --profile headless up
```

### Configuration

**MCP Configuration** (`config/config.yaml`):
Same format as CLI agent - configures LLM and MCP servers.

**Goals Configuration** (`config/goals.yaml`):

```yaml
objectives:
  - "Search for and summarize the latest Ruby news"
  - "Check documentation for man pages on grep"

success_criteria:
  - "All objectives have been addressed"

max_iterations: 50
timeout_minutes: 10
```

### Execution Flow

1. Agent connects to MCP servers and LLM
2. Iterates through each objective in order
3. Sends objective as prompt to LLM with tools
4. Records response and marks objective complete
5. Continues until all objectives done or limits reached
6. Displays summary and exits

### Termination Conditions

Agent stops when:
- ✅ All objectives are completed
- ⏱️ Timeout exceeded (default: 10 minutes)
- 🔄 Max iterations reached (default: 50)

## Goal Evaluation

The agent uses simple keyword matching to determine if an objective has been addressed. For production use, you'd want more sophisticated evaluation (e.g., semantic similarity, explicit confirmation).

## Development

### Requirements

- Ruby 3.4+
- Docker & Docker Compose

### Run Locally

```bash
cd agents/headless
bundle install
bin/based-headless
```

### Custom Goals

Edit `config/goals.yaml` to define your own objectives:

```yaml
objectives:
  - "Your custom task 1"
  - "Your custom task 2"

max_iterations: 100
timeout_minutes: 30
```

## Example Output

```
🤖 Based Headless Agent Starting...

📋 Objectives (2):
   1. Search for and summarize the latest Ruby news
   2. Check documentation for man pages on grep

🔌 Connected to 4 MCP server(s)
🛠️  Available tools: 14

🚀 Starting autonomous execution...

1. 🎯 Task: Search for and summarize the latest Ruby news
   ✅ Response: Based on my web search, here are the latest Ruby news...

2. 🎯 Task: Check documentation for man pages on grep
   ✅ Response: The grep manual page shows...

============================================================
🎉 Execution Complete!

Summary:
  Iterations: 2 / 50
  Objectives: 2 / 2 completed
  Elapsed time: 1.23 minutes

✅ All objectives completed successfully!
============================================================
```

## License

MIT
