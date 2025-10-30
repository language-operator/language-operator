require_relative 'helpers'

module Based
  module Dsl
    # Execution context that includes helpers
    class ExecutionContext
      include Based::Dsl::Helpers

      def initialize(params)
        @params = params
      end

      def method_missing(method, *args)
        # Allow helper methods to be called directly
        super
      end

      def respond_to_missing?(method, include_private = false)
        Based::Dsl::Helpers.instance_methods.include?(method) || super
      end
    end
  end
end
