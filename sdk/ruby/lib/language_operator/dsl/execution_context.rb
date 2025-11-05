# frozen_string_literal: true

module LanguageOperator
  module Dsl
    # Execution context that includes helpers for tool execution
    #
    # Provides helper methods during tool execution, allowing tools to access
    # HTTP, Shell, Config and other utilities directly.
    #
    # @example Using in tool execution
    #   context = LanguageOperator::Dsl::ExecutionContext.new(params)
    #   context.http_get('https://example.com')
    #   context.shell('ls -la')
    class ExecutionContext
      include LanguageOperator::Dsl::Helpers

      # Initialize execution context with parameters
      #
      # @param params [Hash] Tool execution parameters
      def initialize(params)
        @params = params
      end

      # Forward missing methods to helpers
      #
      # @param method [Symbol] Method name
      # @param args [Array] Method arguments
      def method_missing(method, *args)
        # Allow helper methods to be called directly
        super
      end

      # Check if method is available
      #
      # @param method [Symbol] Method name
      # @param include_private [Boolean] Include private methods
      # @return [Boolean]
      def respond_to_missing?(method, include_private = false)
        LanguageOperator::Dsl::Helpers.instance_methods.include?(method) || super
      end
    end
  end
end
