# web

An MCP server that provides web search and HTTP utilities. Built on top of [based/svc/mcp](../mcp), this server allows AI assistants and other tools to search the web and fetch content from URLs.

## Quick Start

Run the server:

```bash
docker run -p 8080:80 based/svc/web:latest
```

Search the web:

```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"web_search","arguments":{"query":"alpine linux"}}'
```

## Available Tools

### `web_search`
Search the web using DuckDuckGo.

**Parameters:**
- `query` (string, required) - The search query
- `max_results` (number, optional) - Maximum number of results (default: 5)

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"web_search","arguments":{"query":"docker containers","max_results":3}}'
```

### `web_fetch`
Fetch and extract content from a URL.

**Parameters:**
- `url` (string, required) - The URL to fetch
- `html` (boolean, optional) - Return raw HTML instead of text (default: false)

**Example:**
```bash
# Get text content
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"web_fetch","arguments":{"url":"https://example.com"}}'

# Get raw HTML
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"web_fetch","arguments":{"url":"https://example.com","html":true}}'
```

### `web_headers`
Fetch HTTP headers from a URL.

**Parameters:**
- `url` (string, required) - The URL to check

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"web_headers","arguments":{"url":"https://example.com"}}'
```

### `web_status`
Check the HTTP status code of a URL.

**Parameters:**
- `url` (string, required) - The URL to check

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"web_status","arguments":{"url":"https://example.com"}}'
```

## Features

- **No API Keys Required**: Uses DuckDuckGo's HTML interface for searching
- **Simple HTTP Utilities**: Check status, headers, and fetch content
- **Text Extraction**: Automatically strips HTML for clean text output
- **Privacy-Focused**: Uses DuckDuckGo which doesn't track searches

## Configuration

Inherits configuration from [based/svc/mcp](../mcp):

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

Run unit tests:

```bash
make spec
```

Test all endpoints (integration):

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

### Testing

The project includes comprehensive test coverage using RSpec:

- **Unit tests** for all 4 web tools (`web_search`, `web_fetch`, `web_headers`, `web_status`)
- **Integration tests** for tool loading and registry
- **Mocked HTTP requests** using WebMock to avoid external dependencies
- **Real fixtures** captured from actual DuckDuckGo responses

Test coverage includes:
- Parameter validation
- URL validation
- Error handling
- Edge cases (empty results, malformed HTML, network errors)
- All HTTP status codes
- HTML stripping and text extraction
- Content truncation

Run tests with `make spec` to execute the full test suite in Docker.

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

- **AI Web Search**: Allow AI assistants to search the web for current information
- **Content Monitoring**: Check website status and headers
- **Data Extraction**: Fetch and parse web content programmatically
- **SEO Tools**: Check HTTP status codes and headers
- **Research Tools**: Search and retrieve information from multiple sources

## Architecture

This image extends `based/svc/mcp:latest` and uses the MCP DSL to define web-related tools. The tools are defined in [tools/web.rb](tools/web.rb) and use `curl` for HTTP requests.

## Limitations

- Search results are parsed from HTML and may change if DuckDuckGo updates their interface
- Text extraction from `web_fetch` is basic and may not work well with complex layouts
- No JavaScript execution (static HTML only)
- Content is limited to first 2000 characters in text mode
