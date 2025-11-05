# frozen_string_literal: true

require_relative '../logger'
require_relative '../loggable'

module LanguageOperator
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
      include LanguageOperator::Loggable

      attr_reader :steps, :step_order

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
        step_def = StepDefinition.new(name, logger: @logger)

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

        logger.info("Executing workflow", step_count: @steps.size)

        @step_order.each do |step_name|
          step_def = @steps[step_name]

          # Check dependencies
          if step_def.dependencies.any?
            logger.debug("Checking dependencies",
                         step: step_name,
                         dependencies: step_def.dependencies)
            step_def.dependencies.each do |dep|
              unless results.key?(dep)
                logger.error("Dependency not satisfied",
                             step: step_name,
                             missing_dependency: dep)
                raise "Step #{step_name} depends on #{dep}, but #{dep} has not been executed"
              end
            end
          end

          # Execute step
          logger.info("Executing step",
                      step: step_name,
                      tool: step_def.tool_name,
                      has_prompt: !step_def.prompt_template.nil?)

          result = logger.timed("Step execution") do
            step_def.execute(results, context)
          end

          results[step_name] = result
          logger.info("Step completed", step: step_name)
        end

        logger.info("Workflow execution completed", total_steps: @steps.size)
        results
      end

      private

      def logger_component
        'Workflow'
      end
    end

    # Individual step definition
    class StepDefinition
      include LanguageOperator::Loggable

      attr_reader :name, :dependencies, :tool_name, :tool_params, :prompt_template

      def initialize(name, logger: nil)
        @name = name
        @tool_name = nil
        @tool_params = {}
        @prompt_template = nil
        @dependencies = []
        @execute_block = nil
        @parent_logger = logger
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
          logger.debug("Executing custom logic", step: @name)
          @execute_block.call(results, context)
        elsif @tool_name
          # Tool execution
          params = interpolate_params(@tool_params, results)
          logger.info("Calling tool",
                      step: @name,
                      tool: @tool_name,
                      params: params)
          # In real implementation, this would call the actual tool
          "Tool #{@tool_name} executed with #{params.inspect}"
        elsif @prompt_template
          # LLM processing
          prompt = interpolate_template(@prompt_template, results)
          logger.debug("LLM prompt",
                       step: @name,
                       prompt: prompt[0..200])
          # In real implementation, this would call the LLM
          "LLM processed: #{prompt}"
        else
          # No-op step
          logger.debug("No execution logic defined", step: @name)
          nil
        end
      end

      alias execute execute_step

      private

      def logger
        @parent_logger || super
      end

      def logger_component
        "Step:#{@name}"
      end

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
    end
  end
end
