# doc

An MCP server that provides an interface to Unix manual pages (man pages). Built on top of [based/svc/mcp](../mcp), this server allows AI assistants and other tools to query and retrieve system documentation.

## Quick Start

Run the server:

```bash
docker run -p 8080:80 based/svc/doc:latest
```

Query a man page:

```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"man","arguments":{"page":"ls"}}'
```

## Available Tools

### `man`
Retrieve and display a manual page.

**Parameters:**
- `page` (string, required) - The name of the manual page (e.g., 'ls', 'grep', 'bash')
- `section` (number, optional) - The manual section (1-8)

**Example:**
```bash
# Get the ls man page
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"man","arguments":{"page":"ls"}}'

# Get printf from section 3 (C library functions)
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"man","arguments":{"page":"printf","section":3}}'
```

### `man_search`
Search for manual pages containing a keyword.

**Parameters:**
- `keyword` (string, required) - Search term

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"man_search","arguments":{"keyword":"network"}}'
```

### `whatis`
Display a brief description of a command.

**Parameters:**
- `command` (string, required) - Command name

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"whatis","arguments":{"command":"grep"}}'
```

### `man_sections`
List all manual sections and their descriptions.

**Parameters:** None

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"man_sections","arguments":{}}'
```

## Manual Sections

1. **User Commands** - Executable programs or shell commands
2. **System Calls** - Functions provided by the kernel
3. **Library Calls** - Functions within program libraries
4. **Special Files** - Usually devices found in /dev
5. **File Formats** - Configuration files and structures
6. **Games** - Games and entertainment programs
7. **Miscellaneous** - Macro packages, conventions, protocols
8. **System Administration** - Commands usually run by root

## Configuration

Inherits configuration from [based/mcp](../mcp):

| Environment Variable | Default | Description |
| -- | -- | -- |
| PORT | 80 | Port to run HTTP server on |
| RACK_ENV | production | Rack environment |

## Development

Build the image:

```bash
make build
```

Run the server:

```bash
make run
```

Test all endpoints:

```bash
make test
```

Run linter:

```bash
make lint
```

Auto-fix linting issues:

```bash
make lint-fix
```

## Documentation

Generate API documentation with YARD:

```bash
make doc
```

Serve documentation locally on http://localhost:8808:

```bash
make doc-serve
```

Clean generated documentation:

```bash
make doc-clean
```

## Use Cases

- **AI Assistants**: Allow AI models to look up command documentation
- **Documentation Tools**: Programmatic access to system documentation
- **Learning Platforms**: Interactive command reference
- **ChatOps**: Slack/Discord bots that can explain commands
- **IDE Extensions**: Context-aware command help

## Architecture

This image extends `based/mcp:latest` and uses the MCP DSL to define tools. The tools are defined in [tools/man.rb](tools/man.rb) and leverage standard Unix utilities (`man`, `apropos`, `whatis`).
