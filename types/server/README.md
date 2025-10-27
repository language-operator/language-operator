# based/server

An extendable tool server based on [the official MCP Ruby SDK](https://github.com/modelcontextprotocol/ruby-sdk).

## Quick Start

Run the server with example tools:

```bash
docker run -p 8080:80 based/server:latest
```

Test the server:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/tools
```

## Creating Your Own MCP Server

### 1. Create Tool Definitions

Create Ruby files using the MCP DSL to define your tools:

```ruby
# tools/hello.rb
tool "greet" do
  description "Greets a person by name"

  parameter "name" do
    type :string
    required true
    description "The name of the person to greet"
  end

  parameter "greeting" do
    type :string
    required false
    description "Custom greeting (default: Hello)"
    default "Hello"
  end

  execute do |params|
    greeting = params["greeting"] || "Hello"
    "#{greeting}, #{params['name']}!"
  end
end
```

### 2. Mount Your Tools Directory

Run the container with your tools directory mounted at `/mcp/tools`:

```bash
docker run -p 8080:80 -v $(pwd)/tools:/mcp/tools based/server:latest
```

### 3. Use Your Tools

Call your tools via the MCP protocol:

```bash
# List available tools
curl -X POST http://localhost:8080/tools/list

# Call a tool
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"greet","arguments":{"name":"World"}}'
```

## DSL Reference

### Defining a Tool

```ruby
tool "tool_name" do
  description "What this tool does"

  parameter "param_name" do
    type :string              # :string, :number, :boolean, :array, :object
    required true             # or false
    description "Parameter description"
    enum ["option1", "option2"]  # optional: restrict to specific values
    default "default_value"   # optional: default value
  end

  execute do |params|
    # Your tool logic here
    # Access parameters via params["param_name"]
    # Return a string result
    "Result: #{params['param_name']}"
  end
end
```

### Parameter Types

- `:string` - Text values
- `:number` - Numeric values (integers or floats)
- `:boolean` - true/false
- `:array` - Lists of values
- `:object` - Complex objects

### Multiple Tools Per File

You can define multiple tools in a single file:

```ruby
tool "add" do
  # ... tool definition
end

tool "subtract" do
  # ... tool definition
end
```

## Example Tools

See [examples/calculator.rb](examples/calculator.rb) for complete examples including:
- Calculator with arithmetic operations
- Echo tool for simple string operations

## Configuration

| Environment Variable | Default | Description |
| -- | -- | -- |
| PORT | 80 | Port to run HTTP server on |
| RACK_ENV | production | Rack environment |

## API Endpoints

### MCP Protocol Endpoints

- `POST /initialize` - Initialize MCP session
- `POST /tools/list` - List available tools
- `POST /tools/call` - Execute a tool

### Debug Endpoints

- `GET /health` - Health check
- `GET /tools` - List loaded tools (simple format)
- `POST /reload` - Reload tools from `/mcp` directory