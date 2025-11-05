# frozen_string_literal: true

module LanguageOperator
  module Dsl
    # DSL context for defining tools
    #
    # Provides the evaluation context for tool definition files. Tools are
    # defined using the `tool` method within this context.
    #
    # @example Tool definition file
    #   tool "search" do
    #     description "Search the web"
    #
    #     parameter :query do
    #       description "Search query"
    #       type :string
    #       required true
    #     end
    #
    #     execute do |params|
    #       http_get("https://api.search.com?q=#{params[:query]}")
    #     end
    #   end
    class Context
      include LanguageOperator::Dsl::Helpers

      # Initialize context with registry
      #
      # @param registry [LanguageOperator::Dsl::Registry] Tool registry
      def initialize(registry)
        @registry = registry
      end

      # Define a tool
      #
      # @param name [String] Tool name
      # @yield Tool definition block
      # @return [void]
      def tool(name, &block)
        tool_def = ToolDefinition.new(name)
        tool_def.instance_eval(&block) if block_given?
        @registry.register(tool_def)
      end
    end
  end
end
