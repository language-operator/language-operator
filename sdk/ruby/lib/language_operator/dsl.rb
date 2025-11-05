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
require_relative 'dsl/agent_definition'
require_relative 'dsl/agent_context'
require_relative 'dsl/workflow_definition'

module LanguageOperator
  # DSL for defining MCP tools and autonomous agents
  #
  # Provides a clean, Ruby-like DSL for defining tools that can be served
  # via the Model Context Protocol (MCP) and agents that can execute autonomously.
  #
  # @example Define a tool
  #   LanguageOperator::Dsl.define do
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
  #   registry = LanguageOperator::Dsl.registry
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

      # Global registry for agents
      #
      # @return [AgentRegistry] The global agent registry
      def agent_registry
        @agent_registry ||= AgentRegistry.new
      end

      # Define tools using the DSL
      #
      # @yield Block containing tool definitions
      # @return [Registry] The global registry with defined tools
      #
      # @example
      #   LanguageOperator::Dsl.define do
      #     tool "example" do
      #       # ...
      #     end
      #   end
      def define(&block)
        context = Context.new(registry)
        context.instance_eval(&block)
        registry
      end

      # Define agents using the DSL
      #
      # @yield Block containing agent definitions
      # @return [AgentRegistry] The global agent registry
      #
      # @example
      #   LanguageOperator::Dsl.define_agents do
      #     agent "news-summarizer" do
      #       # ...
      #     end
      #   end
      def define_agents(&block)
        context = AgentContext.new(agent_registry)
        context.instance_eval(&block)
        agent_registry
      end

      # Load tools from a file
      #
      # @param file_path [String] Path to the tool definition file
      # @return [Registry] The global registry with loaded tools
      #
      # @example
      #   LanguageOperator::Dsl.load_file("mcp/tools.rb")
      def load_file(file_path)
        code = File.read(file_path)
        context = Context.new(registry)
        context.instance_eval(code, file_path)
        registry
      end

      # Load agents from a file
      #
      # @param file_path [String] Path to the agent definition file
      # @return [AgentRegistry] The global agent registry
      #
      # @example
      #   LanguageOperator::Dsl.load_agent_file("agents/news-summarizer.rb")
      def load_agent_file(file_path)
        code = File.read(file_path)
        context = AgentContext.new(agent_registry)
        context.instance_eval(code, file_path)
        agent_registry
      end

      # Clear all defined tools
      #
      # @return [void]
      def clear!
        registry.clear
      end

      # Clear all defined agents
      #
      # @return [void]
      def clear_agents!
        agent_registry.clear
      end

      # Create an MCP server from the defined tools
      #
      # @param server_name [String] Name of the MCP server
      # @param server_context [Hash] Additional context for the server
      # @return [MCP::Server] The MCP server instance
      #
      # @example
      #   server = LanguageOperator::Dsl.create_server(server_name: "my-tools")
      def create_server(server_name: 'langop-tools', server_context: {})
        Adapter.create_mcp_server(registry, server_name: server_name, server_context: server_context)
      end
    end
  end
end
