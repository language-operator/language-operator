# frozen_string_literal: true

module Langop
  module Dsl
    # Workflow definition for agent execution
    #
    # Defines a series of steps that an agent executes to achieve objectives.
    # Steps can depend on other steps, call tools, or perform LLM processing.
    #
    # @example Define a workflow
    #   workflow do
    #     step :search do
    #       tool "web_search"
    #       params query: "latest news"
    #     end
    #
    #     step :summarize do
    #       depends_on :search
    #       prompt "Summarize: {search.output}"
    #     end
    #   end
    class WorkflowDefinition
      attr_reader :steps

      def initialize
        @steps = {}
        @step_order = []
      end

      # Define a workflow step
      #
      # @param name [Symbol] Step name
      # @param tool [String, nil] Tool to use (optional)
      # @param params [Hash] Tool parameters (optional)
      # @param depends_on [Symbol, Array<Symbol>] Dependencies (optional)
      # @yield Step definition block
      # @return [void]
      def step(name, tool: nil, params: {}, depends_on: nil, &block)
        step_def = StepDefinition.new(name)

        if tool
          step_def.tool(tool)
          step_def.params(params) unless params.empty?
        end

        if depends_on
          step_def.depends_on(depends_on)
        end

        step_def.instance_eval(&block) if block
        @steps[name] = step_def
        @step_order << name
      end

      # Execute the workflow
      #
      # @param context [Object] Execution context
      # @return [Hash] Results from each step
      def execute(context = nil)
        results = {}

        log_info "Executing workflow with #{@steps.size} steps"

        @step_order.each do |step_name|
          step_def = @steps[step_name]

          # Check dependencies
          if step_def.dependencies.any?
            step_def.dependencies.each do |dep|
              unless results.key?(dep)
                raise "Step #{step_name} depends on #{dep}, but #{dep} has not been executed"
              end
            end
          end

          # Execute step
          log_info "Executing step: #{step_name}"
          result = step_def.execute(results, context)
          results[step_name] = result
        end

        results
      end

      private

      def log_info(message)
        puts "[Workflow] #{message}"
      end
    end

    # Individual step definition
    class StepDefinition
      attr_reader :name, :dependencies

      def initialize(name)
        @name = name
        @tool_name = nil
        @tool_params = {}
        @prompt_template = nil
        @dependencies = []
        @execute_block = nil
      end

      # Set the tool to use
      #
      # @param name [String] Tool name
      # @return [void]
      def tool(name = nil)
        return @tool_name if name.nil?
        @tool_name = name
      end

      # Set tool parameters
      #
      # @param hash [Hash] Parameters
      # @return [Hash] Current parameters
      def params(hash = nil)
        return @tool_params if hash.nil?
        @tool_params = hash
      end

      # Set prompt template (for LLM processing)
      #
      # @param template [String] Prompt template
      # @return [String] Current prompt
      def prompt(template = nil)
        return @prompt_template if template.nil?
        @prompt_template = template
      end

      # Declare dependencies on other steps
      #
      # @param steps [Symbol, Array<Symbol>] Step names this depends on
      # @return [Array<Symbol>] Current dependencies
      def depends_on(*steps)
        return @dependencies if steps.empty?
        @dependencies = steps.flatten
      end

      # Define custom execution logic
      #
      # @yield Execution block
      # @return [void]
      def execute(&block)
        @execute_block = block if block
      end

      # Execute this step
      #
      # @param results [Hash] Results from previous steps
      # @param context [Object] Execution context
      # @return [Object] Step result
      def execute_step(results, context)
        if @execute_block
          # Custom execution logic
          @execute_block.call(results, context)
        elsif @tool_name
          # Tool execution
          params = interpolate_params(@tool_params, results)
          log_debug "Calling tool #{@tool_name} with params: #{params.inspect}"
          # In real implementation, this would call the actual tool
          "Tool #{@tool_name} executed with #{params.inspect}"
        elsif @prompt_template
          # LLM processing
          prompt = interpolate_template(@prompt_template, results)
          log_debug "LLM prompt: #{prompt}"
          # In real implementation, this would call the LLM
          "LLM processed: #{prompt}"
        else
          # No-op step
          log_debug "Step #{@name} has no execution logic"
          nil
        end
      end

      alias execute execute_step

      private

      # Interpolate parameters with results from previous steps
      #
      # @param params [Hash] Parameter template
      # @param results [Hash] Previous results
      # @return [Hash] Interpolated parameters
      def interpolate_params(params, results)
        params.transform_values do |value|
          if value.is_a?(String) && value.match?(/\{(\w+)\.(\w+)\}/)
            # Replace {step.field} with actual value
            value.gsub(/\{(\w+)\.(\w+)\}/) do
              step_name = Regexp.last_match(1).to_sym
              field = Regexp.last_match(2)
              results.dig(step_name, field) || value
            end
          else
            value
          end
        end
      end

      # Interpolate template string with results
      #
      # @param template [String] Template string
      # @param results [Hash] Previous results
      # @return [String] Interpolated string
      def interpolate_template(template, results)
        template.gsub(/\{(\w+)(?:\.(\w+))?\}/) do
          step_name = Regexp.last_match(1).to_sym
          field = Regexp.last_match(2)

          if field
            results.dig(step_name, field)&.to_s || "{#{step_name}.#{field}}"
          else
            results[step_name]&.to_s || "{#{step_name}}"
          end
        end
      end

      def log_debug(message)
        puts "[Step:#{@name}] #{message}" if ENV['DEBUG']
      end
    end
  end
end
