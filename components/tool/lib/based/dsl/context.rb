# frozen_string_literal: true

module Based
  module Dsl
    # DSL context for defining tools
    #
    # This class provides the context for tool definitions and delegates
    # to the SDK gem's ToolDefinition and Helpers classes.
    class Context
      include Based::Dsl::Helpers

      def initialize(registry)
        @registry = registry
      end

      def tool(name, &block)
        tool_def = ToolDefinition.new(name)
        tool_def.instance_eval(&block) if block_given?
        @registry.register(tool_def)
      end
    end
  end
end
