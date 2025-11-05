# frozen_string_literal: true

require 'mcp'

module Aictl
  module Dsl
    # Adapter to bridge our DSL to the official MCP Ruby SDK
    #
    # Converts Aictl::Dsl::ToolDefinition objects into MCP::Tool classes
    # that can be used with the official MCP Ruby SDK.
    #
    # @example Convert a tool definition
    #   tool_class = Adapter.tool_definition_to_mcp_tool(tool_def)
    #   server = MCP::Server.new(tools: [tool_class])
    class Adapter
      # Convert our ToolDefinition to an MCP::Tool class
      def self.tool_definition_to_mcp_tool(tool_def)
        # Build input schema from our parameter definitions
        schema = build_input_schema(tool_def.parameters)

        # Create a dynamic class that extends MCP::Tool
        tool_class = Class.new(MCP::Tool) do
          # Set the tool name
          define_singleton_method(:name) { tool_def.name }

          # Set the description
          description tool_def.description

          # Set input schema
          input_schema(schema)

          # Define the call method
          define_singleton_method(:call) do |server_context: {}, **args|
            # Convert args to string keys to match our DSL expectations
            params = args.transform_keys(&:to_s)

            # Execute the tool using our DSL
            result = tool_def.call(params)

            # Convert result to MCP::Tool::Response
            MCP::Tool::Response.new([
              { type: 'text', text: result.to_s }
            ])
          rescue ArgumentError => e
            # Return error as text response
            MCP::Tool::Response.new([
              { type: 'text', text: "Error: #{e.message}" }
            ])
          rescue StandardError => e
            # Return error as text response
            MCP::Tool::Response.new([
              { type: 'text', text: "Error: #{e.message}" }
            ])
          end
        end

        tool_class
      end

      # Build MCP input schema from our parameter definitions
      def self.build_input_schema(parameters)
        properties = {}
        required = []

        parameters.each do |name, param_def|
          properties[name.to_sym] = build_parameter_schema(param_def)
          required << name if param_def.required
        end

        schema = { properties: properties }
        schema[:required] = required unless required.empty?
        schema
      end

      # Build schema for a single parameter
      def self.build_parameter_schema(param_def)
        # Access instance variables directly since the DSL methods are setters
        param_type = param_def.instance_variable_get(:@type)
        param_desc = param_def.instance_variable_get(:@description)
        param_enum = param_def.instance_variable_get(:@enum)
        param_default = param_def.instance_variable_get(:@default)

        schema = {
          type: map_type(param_type),
          description: param_desc
        }

        schema[:enum] = param_enum if param_enum
        schema[:default] = param_default if param_default

        schema
      end

      # Map our type symbols to JSON schema types
      def self.map_type(ruby_type)
        case ruby_type
        when :string then 'string'
        when :number, :integer then 'number'
        when :boolean then 'boolean'
        when :array then 'array'
        when :object then 'object'
        else 'string'
        end
      end

      # Convert a registry of tools to MCP::Tool classes
      def self.registry_to_mcp_tools(registry)
        registry.all.map do |tool_def|
          tool_definition_to_mcp_tool(tool_def)
        end
      end

      # Create an MCP::Server from our registry
      def self.create_mcp_server(registry, server_name: 'langop-mcp', server_context: {})
        tools = registry_to_mcp_tools(registry)

        MCP::Server.new(
          name: server_name,
          version: Aictl::VERSION,
          tools: tools,
          server_context: server_context
        )
      end
    end
  end
end
