# language-operator

Custom resource definitions for language-based automations.


## Quick Start


```

## Services

The docker-compose setup runs the following services:

| Service | Port | Description |
| --- | --- | --- |
| server | 8080 | MCP Tools Server (base, no tools loaded) |
| doc | 8081 | Documentation/Man Pages Tool |
| email | 8082 | Email/SMTP Tool |
| sms | 8083 | SMS Tool (Twilio/Vonage) |
| web | 8084 | Web Search & HTTP Tool |

All services use the MCP (Model Context Protocol) JSON-RPC 2.0 format.

### Example Request

```bash
curl -X POST http://localhost:8084/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "web_status",
      "arguments": {"url": "https://example.com"}
    }
  }'
```

## Configuration

Copy the example environment file and configure your credentials:

```bash
cp .env.example .env
```

Edit `.env` to add your SMTP and SMS provider credentials.

## Volume Mounts

Each tool service mounts its tool definitions and documentation:

- Tool definitions: `./tools/{name}/tools:/mcp:ro` (read-only)
- Documentation: `./tools/{name}/doc:/app/doc` (read-write)

This allows you to:
- Edit tool definitions and reload without rebuilding
- Generate documentation on your host machine
- View documentation via HTTP at `/doc/` (e.g., `http://localhost:8084/doc/`)

Note: The base server code (`server.rb`, `lib/`) is baked into the image. Only tool definitions in the `/mcp` directory are mounted. Documentation files in `/app/doc` are served as static files at the web root.

## Development Workflow

1. Build all images initially:
   ```bash
   make build
   ```

2. Start all services:
   ```bash
   make up
   ```

3. Make changes to code in `tools/` or `types/`

4. Rebuild specific image:
   ```bash
   docker build -t based/tools/web:latest tools/web/
   ```

5. Restart the service:
   ```bash
   docker-compose restart web
   ```

## Available Make Targets

Run `make help` to see all available targets:

- `make build` - Build all Docker images
- `make up` - Start all services
- `make down` - Stop all services
- `make logs` - View logs from all services
- `make ps` - List running containers
- `make restart` - Restart all services
- `make clean-volumes` - Stop and remove all volumes

## Project Structure

```
based/
├── types/                  # Base image types
│   ├── base/              # Alpine base with Ruby
│   ├── devel/             # Development tools
│   ├── server/            # MCP server framework
│   └── task/              # Task runner
├── tools/                  # MCP tool services
│   ├── doc/               # Man page tool
│   ├── email/             # Email/SMTP tool
│   ├── sms/               # SMS tool
│   └── web/               # Web search tool
├── scripts/
│   └── build              # Build all images script
├── docker-compose.yml      # Service orchestration
├── .env.example           # Environment template
└── Makefile               # Build and run commands
```