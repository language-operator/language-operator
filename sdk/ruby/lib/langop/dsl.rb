# frozen_string_literal: true

require_relative 'version'
require_relative 'dsl/tool_definition'
require_relative 'dsl/parameter_definition'
require_relative 'dsl/registry'
require_relative 'dsl/adapter'
require_relative 'dsl/config'
require_relative 'dsl/helpers'
require_relative 'dsl/http'
require_relative 'dsl/shell'
require_relative 'dsl/context'
require_relative 'dsl/execution_context'

module Langop
  # DSL for defining MCP tools
  #
  # Provides a clean, Ruby-like DSL for defining tools that can be served
  # via the Model Context Protocol (MCP).
  #
  # @example Define a tool
  #   Langop::Dsl.define do
  #     tool "greet" do
  #       description "Greet a user by name"
  #
  #       parameter :name do
  #         type :string
  #         required true
  #         description "Name to greet"
  #       end
  #
  #       execute do |params|
  #         "Hello, #{params['name']}!"
  #       end
  #     end
  #   end
  #
  # @example Access tools
  #   registry = Langop::Dsl.registry
  #   tool = registry.get("greet")
  #   result = tool.call({"name" => "Alice"})
  module Dsl
    class << self
      # Global registry for tools
      #
      # @return [Registry] The global tool registry
      def registry
        @registry ||= Registry.new
      end

      # Define tools using the DSL
      #
      # @yield Block containing tool definitions
      # @return [Registry] The global registry with defined tools
      #
      # @example
      #   Langop::Dsl.define do
      #     tool "example" do
      #       # ...
      #     end
      #   end
      def define(&block)
        context = Context.new(registry)
        context.instance_eval(&block)
        registry
      end

      # Load tools from a file
      #
      # @param file_path [String] Path to the tool definition file
      # @return [Registry] The global registry with loaded tools
      #
      # @example
      #   Langop::Dsl.load_file("mcp/tools.rb")
      def load_file(file_path)
        code = File.read(file_path)
        context = Context.new(registry)
        context.instance_eval(code, file_path)
        registry
      end

      # Clear all defined tools
      #
      # @return [void]
      def clear!
        registry.clear
      end

      # Create an MCP server from the defined tools
      #
      # @param server_name [String] Name of the MCP server
      # @param server_context [Hash] Additional context for the server
      # @return [MCP::Server] The MCP server instance
      #
      # @example
      #   server = Langop::Dsl.create_server(server_name: "my-tools")
      def create_server(server_name: 'langop-tools', server_context: {})
        Adapter.create_mcp_server(registry, server_name: server_name, server_context: server_context)
      end
    end
  end
end
