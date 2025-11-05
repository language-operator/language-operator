# frozen_string_literal: true

module LanguageOperator
  module Dsl
    # Registry for storing and retrieving tool definitions
    #
    # Manages a collection of tools defined using the DSL.
    # Tools can be registered, retrieved by name, or accessed as a collection.
    #
    # @example Using the registry
    #   registry = Registry.new
    #   registry.register(tool_definition)
    #   all_tools = registry.all
    class Registry
      def initialize
        @tools = {}
      end

      def register(tool)
        @tools[tool.name] = tool
      end

      def get(name)
        @tools[name]
      end

      def all
        @tools.values
      end

      def clear
        @tools.clear
      end
    end
  end
end
