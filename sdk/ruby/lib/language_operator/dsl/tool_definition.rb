# frozen_string_literal: true

require_relative 'parameter_definition'

module LanguageOperator
  module Dsl
    # Tool definition for MCP tools
    #
    # Defines an MCP tool with parameters, description, and execution logic.
    # Used within the DSL to create tools that can be registered and served.
    #
    # @example Define a simple tool
    #   tool "greet" do
    #     description "Greet a user by name"
    #
    #     parameter :name do
    #       type :string
    #       required true
    #       description "Name to greet"
    #     end
    #
    #     execute do |params|
    #       "Hello, #{params['name']}!"
    #     end
    #   end
    class ToolDefinition
      attr_reader :name, :parameters, :execute_block

      def initialize(name)
        @name = name
        @parameters = {}
        @execute_block = nil
        @description = nil
      end

      def description(val = nil)
        return @description if val.nil?
        @description = val
      end

      def parameter(name, &block)
        param = ParameterDefinition.new(name)
        param.instance_eval(&block) if block_given?
        @parameters[name.to_s] = param
      end

      def execute(&block)
        @execute_block = block
      end

      def call(params)
        log_debug "Calling tool '#{@name}' with params: #{params.inspect}"

        # Apply default values for missing optional parameters
        @parameters.each do |name, param_def|
          default_value = param_def.instance_variable_get(:@default)
          if !params.key?(name) && !default_value.nil?
            params[name] = default_value
          end
        end

        # Validate required parameters
        @parameters.each do |name, param_def|
          if param_def.required && !params.key?(name)
            raise ArgumentError, "Missing required parameter: #{name}"
          end

          # Validate parameter format if validator is set and value is present
          if params.key?(name)
            error = param_def.validate_value(params[name])
            raise ArgumentError, error if error
          end
        end

        # Call the execute block with parameters
        result = @execute_block.call(params) if @execute_block

        log_debug "Tool '#{@name}' completed: #{truncate_for_log(result)}"
        result
      end

      def to_schema
        {
          'name' => @name,
          'description' => @description,
          'inputSchema' => {
            'type' => 'object',
            'properties' => @parameters.transform_values(&:to_schema),
            'required' => @parameters.select { |_, p| p.required }.keys
          }
        }
      end

      private

      def log_debug(message)
        puts "[DEBUG] #{message}" if ENV['DEBUG'] || ENV['MCP_DEBUG']
      end

      def truncate_for_log(text)
        return text.inspect if text.nil?
        str = text.to_s
        str.length > 100 ? "#{str[0..100]}..." : str
      end
    end
  end
end
