# frozen_string_literal: true

module LanguageOperator
  # Mixin module to provide automatic logger initialization for classes
  # that need logging capabilities.
  #
  # @example
  #   class MyClass
  #     include LanguageOperator::Loggable
  #
  #     def process
  #       logger.info("Processing started")
  #       # ...
  #     end
  #   end
  #
  # The logger component name is automatically derived from the class name.
  # You can override the component name by defining a `logger_component` method.
  #
  # @example Custom component name
  #   class MyClass
  #     include LanguageOperator::Loggable
  #
  #     def logger_component
  #       'CustomName'
  #     end
  #   end
  module Loggable
    # Returns a logger instance for this class.
    # Lazily initializes the logger on first access.
    #
    # @return [LanguageOperator::Logger] Logger instance
    def logger
      @logger ||= LanguageOperator::Logger.new(component: logger_component)
    end

    private

    # Returns the component name to use for the logger.
    # Defaults to the class name, but can be overridden.
    #
    # @return [String] Component name for the logger
    def logger_component
      self.class.name
    end
  end
end
