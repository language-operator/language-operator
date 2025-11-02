# Langop MCP Chat Client

A command-line chat interface that connects to **multiple** Langop MCP servers using [RubyLLM](https://rubyllm.com/) and [RubyLLM::MCP](https://www.rubyllm-mcp.com/).

## Features

- 🤖 Interactive chat with LLM (OpenAI, Anthropic, or **custom OpenAI-compatible endpoints**)
- 🏠 **Local LLM support** (Ollama, LM Studio, vLLM, etc.)
- 🔧 **Connect to multiple MCP servers** simultaneously
- 🛠️ Automatic tool discovery and aggregation from all servers
- 💬 Persistent conversation context
- 🎨 Simple command-line interface with readline support
- ⚙️ **YAML configuration file** for easy server management

## Requirements

- Ruby 3.4+
- One of:
  - Local LLM server with OpenAI-compatible API (Ollama, LM Studio, vLLM, etc.)
  - OpenAI API key
  - Anthropic API key
- Access to one or more Langop MCP servers

## Configuration

The client supports two configuration methods:

### Method 1: YAML Configuration File (Recommended)

Create `config/config.yaml` from the example:

```bash
cp config/config.example.yaml config/config.yaml
```

Edit the file to configure your LLM and MCP servers:

```yaml
# LLM Configuration
llm:
  provider: openai_compatible  # openai, anthropic, or openai_compatible
  model: llama3.2
  endpoint: http://host.docker.internal:11434/v1  # For local LLMs
  # api_key: sk-...  # Optional for local, required for cloud providers

# MCP Servers - Connect to multiple servers!
mcp_servers:
  - name: based-server
    url: http://server:80/mcp
    transport: streamable
    enabled: true
    description: "Main based MCP server"

  - name: doc-tools
    url: http://doc:80/mcp
    transport: streamable
    enabled: true
    description: "Documentation tools"

  - name: email-tools
    url: http://email:80/mcp
    transport: streamable
    enabled: true
    description: "Email tools"

  # Add more servers as needed...

debug: false
```

**Benefits of YAML config:**
- Connect to multiple MCP servers simultaneously
- Enable/disable servers without changing code
- Better organization and documentation
- Easy to version control

### Method 2: Environment Variables (Fallback)

If no `config.yaml` exists, the client falls back to environment variables:

```bash
# For local LLM (Ollama example)
OPENAI_ENDPOINT=http://host.docker.internal:11434/v1
LLM_MODEL=llama3.2

# For OpenAI
# OPENAI_API_KEY=sk-...
# LLM_MODEL=gpt-4

# For Anthropic
# ANTHROPIC_API_KEY=sk-ant-...
# LLM_MODEL=claude-3-5-sonnet-20241022

# MCP Server (single server only)
MCP_URL=http://server:80/mcp

# Debug mode
DEBUG=true
```

**Common local LLM endpoints:**
- **Ollama**: `http://host.docker.internal:11434/v1`
- **LM Studio**: `http://host.docker.internal:1234/v1`
- **vLLM**: `http://host.docker.internal:8000/v1`
- **Text Generation WebUI**: `http://host.docker.internal:5000/v1`

## Usage

### Using Kubernetes (Recommended)

Deploy the client as a LanguageClient resource:

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageClient
metadata:
  name: chat-client
spec:
  image: based/client:latest
  llmConfig:
    provider: openai_compatible
    model: llama3.2
    endpoint: http://llm-service:8080/v1
  mcpServers:
    - name: tools-server
      url: http://tools-server:80/mcp
EOF
```

### Using Docker Locally

```bash
# Build the image
docker build -t based/client:latest .

# Run with local LLM
docker run -it --rm \
  -e OPENAI_ENDPOINT=http://host.docker.internal:11434/v1 \
  -e LLM_MODEL=llama3.2 \
  --network based-network \
  based/client:latest

# Run with OpenAI
docker run -it --rm \
  -e OPENAI_API_KEY=sk-... \
  --network based-network \
  based/client:latest
```

### Local Development

```bash
# Install dependencies
bundle install

# Run with local LLM
OPENAI_ENDPOINT=http://localhost:11434/v1 \
LLM_MODEL=llama3.2 \
ruby chat.rb

# Run with OpenAI
OPENAI_API_KEY=sk-... ruby chat.rb
```

## Chat Commands

- `/help` - Show available tools from all connected servers
- `/servers` - Show all connected MCP servers
- `/clear` - Clear chat history
- `/exit` - Exit the chat (or Ctrl+D)

## Example Sessions

### With Multiple MCP Servers and Local LLM

```
🚀 Langop MCP Chat Client Starting...

📝 LLM: llama3.2 (custom endpoint: http://host.docker.internal:11434/v1)

🔌 Connecting to 5 MCP server(s)...

  • based-server: http://server:80/mcp
    └─ No tools available
  • doc-tools: http://doc:80/mcp
    └─ 4 tool(s): generate_docs, view_man_page, search_docs, list_docs
  • email-tools: http://email:80/mcp
    └─ 3 tool(s): send_email, send_bulk_email, validate_email
  • sms-tools: http://sms:80/mcp
    └─ 3 tool(s): send_sms, send_bulk_sms, get_sms_status
  • web-tools: http://web:80/mcp
    └─ 4 tool(s): fetch_url, scrape_page, http_request, extract_links

✅ Ready with 14 tool(s) from 5 server(s)

Type your messages below. Commands:
  /help     - Show available tools
  /servers  - Show connected servers
  /clear    - Clear chat history
  /exit     - Exit the chat

You: What tools do you have available?

You: /servers

Connected MCP servers:
  • doc-tools - 4 tool(s)
  • email-tools - 3 tool(s)
  • sms-tools - 3 tool(s)
  • web-tools - 4 tool(s)
```

### With Single Server and OpenAI

```
🚀 Langop MCP Chat Client Starting...

📝 LLM: gpt-4 (openai)

🔌 Connecting to 1 MCP server(s)...

  • based-server: http://server:80/mcp
    └─ 5 tool(s): read_file, write_file, list_directory, ...

✅ Ready with 5 tool(s) from 1 server(s)

Type your messages below. Commands:
  /help     - Show available tools
  /servers  - Show connected servers
  /clear    - Clear chat history
  /exit     - Exit the chat

You: List files in the current directory
```