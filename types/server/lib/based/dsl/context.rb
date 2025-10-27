require_relative 'tool_definition'
require_relative 'helpers'

module Based
  module Dsl
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
