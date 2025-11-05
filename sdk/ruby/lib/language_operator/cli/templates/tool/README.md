# {{name}}

An MCP tool created with Langop.

## Installation

```bash
bundle install
```

## Testing

Test your tool definitions:

```bash
langop test mcp/{{name}}.rb
```

## Running

Start the MCP server:

```bash
langop serve mcp/{{name}}.rb
```

The server will start on `http://0.0.0.0:80` by default.

## Customization

Edit `mcp/{{name}}.rb` to add your tool implementations. See the [Langop documentation](https://github.com/langop/language-operator/tree/main/sdk/ruby) for more information.
