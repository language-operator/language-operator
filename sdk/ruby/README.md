# Langop - Ruby SDK for MCP Tools and Language Agents

A comprehensive Ruby SDK for building Model Context Protocol (MCP) tools and autonomous language agents with a clean, expressive DSL.

## Features

- **Clean DSL** - Define MCP tools with intuitive Ruby syntax
- **Type-Safe Parameters** - Built-in parameter validation and schema generation
- **Helper Libraries** - HTTP client, shell execution, config management
- **Client Library** - Connect to MCP servers and interact with LLMs
- **Agent Framework** - Build autonomous agents with scheduling and workspace support
- **CLI Tool** - Generate, test, and run tools and agents
- **Production Ready** - Used in the Language Operator Kubernetes platform

## Installation

Add this line to your application's Gemfile:

```ruby
gem 'langop'
```

And then execute:

```bash
bundle install
```

Or install it yourself as:

```bash
gem install langop
```

## Quick Start

### Create a Tool

```bash
langop new tool calculator
cd calculator
bundle install
langop test mcp/calculator.rb
langop serve mcp/calculator.rb
```

### Create an Agent

```bash
langop new agent news-bot
cd news-bot
bundle install
langop run
```

## Documentation

See the full documentation at [github.com/langop/language-operator/tree/main/sdk/ruby](https://github.com/langop/language-operator/tree/main/sdk/ruby)

## CLI Commands

- `langop new TYPE NAME` - Generate a new tool or agent project
- `langop serve [FILE]` - Start an MCP server for tools
- `langop test [FILE]` - Test tool definitions
- `langop run` - Run an agent
- `langop version` - Show version
- `langop console` - Start interactive console

## License

MIT