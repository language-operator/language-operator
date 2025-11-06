# frozen_string_literal: true

require "language_operator/tool_loader"

# Start the MCP server using LanguageOperator SDK
# This will load all tools from /mcp and start the server on PORT (default 80)
LanguageOperator::ToolLoader.start
