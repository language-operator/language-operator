# Based MCP CLI Agent

Interactive terminal interface for chatting with LLMs and using MCP tools.

## Features

- **Interactive REPL**: Command-line chat interface with readline support
- **Multi-server**: Connect to multiple MCP servers simultaneously
- **Custom LLMs**: Support for OpenAI, Anthropic, or custom OpenAI-compatible endpoints
- **Tool execution**: Automatic tool calling and result display
- **History management**: Clear and manage conversation history
- **Debug mode**: Detailed error messages and backtraces

## Usage

### With Docker Compose

```bash
# From repository root
docker compose run --rm agent-cli

# Or from this directory
make run
```

### Configuration

Create `config/config.yaml` based on `config/config.example.yaml`:

```yaml
llm:
  provider: openai_compatible
  model: llama3.2
  endpoint: http://192.168.68.54:1234/v1

mcp_servers:
  - name: doc-tools
    url: http://doc:80/mcp
    transport: streamable
    enabled: true

  - name: web-tools
    url: http://web:80/mcp
    transport: streamable
    enabled: true

debug: false
```

### Environment Variables

Alternatively, configure via environment variables:

```bash
export OPENAI_ENDPOINT=http://localhost:11434/v1
export LLM_MODEL=llama3.2
export MCP_URL=http://server:80/mcp

docker compose run --rm agent-cli
```

## Commands

While in the chat interface:

- `/help` - Show available tools
- `/servers` - Show connected servers
- `/clear` - Clear chat history
- `/exit` - Exit the chat

## Development

### Requirements

- Ruby 3.4+
- Docker & Docker Compose

### Build

```bash
make build
```

### Lint

```bash
bundle install
make lint
```

### Documentation

```bash
make doc
open doc/index.html
```

## License

MIT
