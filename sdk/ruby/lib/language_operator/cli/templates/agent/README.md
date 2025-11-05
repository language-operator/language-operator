# {{name}}

A language agent created with Langop.

## Installation

```bash
bundle install
```

## Configuration

Edit `config/config.yaml` to configure:
- LLM provider and model
- MCP server connections
- Agent instructions
- Execution mode and schedules

## Running

Start the agent:

```bash
langop run
```

Or with a specific config file:

```bash
langop run --config config/config.yaml
```

## Modes

The agent supports different execution modes via the `AGENT_MODE` environment variable:

- `autonomous` - Continuous execution with rate limiting
- `interactive` - Interactive mode (coming soon)
- `scheduled` - Cron-based scheduled execution
- `event-driven` - Event-driven execution

```bash
AGENT_MODE=scheduled langop run
```

## Customization

See the [Langop documentation](https://github.com/langop/language-operator/tree/main/sdk/ruby) for more information on customizing your agent.
