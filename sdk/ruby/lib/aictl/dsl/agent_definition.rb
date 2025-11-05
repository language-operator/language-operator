# frozen_string_literal: true

require_relative 'workflow_definition'
require_relative '../logger'
require_relative '../loggable'

module Aictl
  module Dsl
    # Agent definition for autonomous agents
    #
    # Defines an agent with objectives, workflow, schedule, and constraints.
    # Used within the DSL to create agents that can be executed standalone
    # or deployed to Kubernetes.
    #
    # @example Define a simple scheduled agent
    #   agent "news-summarizer" do
    #     description "Daily news summarization agent"
    #
    #     schedule "0 12 * * *"
    #
    #     objectives [
    #       "Search for recent news",
    #       "Summarize findings"
    #     ]
    #
    #     workflow do
    #       step :search, tool: "web_search", params: {query: "latest news"}
    #       step :summarize, depends_on: :search
    #     end
    #   end
    class AgentDefinition
      include Aictl::Loggable

      attr_reader :name, :description, :persona, :schedule, :objectives, :workflow,
                  :constraints, :output_config, :execution_mode

      def initialize(name)
        @name = name
        @description = nil
        @persona = nil
        @schedule = nil
        @objectives = []
        @workflow = nil
        @constraints = {}
        @output_config = {}
        @execution_mode = :autonomous

        logger.debug("Agent definition initialized",
                     name: name,
                     mode: @execution_mode)
      end

      # Set or get description
      #
      # @param val [String, nil] Description text
      # @return [String] Current description
      def description(val = nil)
        return @description if val.nil?
        @description = val
      end

      # Set persona/system prompt
      #
      # @param text [String] Persona text or system prompt
      # @return [String] Current persona
      def persona(text = nil)
        return @persona if text.nil?
        @persona = text
      end

      # Set schedule (cron expression)
      #
      # @param cron [String] Cron expression
      # @return [String] Current schedule
      def schedule(cron = nil)
        return @schedule if cron.nil?
        @schedule = cron
        @execution_mode = :scheduled
      end

      # Set objectives (list of goals)
      #
      # @param list [Array<String>] List of objectives
      # @return [Array<String>] Current objectives
      def objectives(list = nil)
        return @objectives if list.nil?
        @objectives = list
      end

      # Define a single objective
      #
      # @param text [String] Objective text
      # @return [void]
      def objective(text)
        @objectives << text
      end

      # Define workflow with steps
      #
      # @yield Workflow definition block
      # @return [WorkflowDefinition] Current workflow
      def workflow(&block)
        return @workflow if block.nil?
        @workflow = WorkflowDefinition.new
        @workflow.instance_eval(&block) if block
        @workflow
      end

      # Define constraints (max_iterations, timeout, etc.)
      #
      # @yield Constraints block
      # @return [Hash] Current constraints
      def constraints(&block)
        return @constraints if block.nil?
        constraint_builder = ConstraintBuilder.new
        constraint_builder.instance_eval(&block) if block
        @constraints = constraint_builder.to_h
      end

      # Define output configuration
      #
      # @yield Output configuration block
      # @return [Hash] Current output config
      def output(&block)
        return @output_config if block.nil?
        output_builder = OutputBuilder.new
        output_builder.instance_eval(&block) if block
        @output_config = output_builder.to_h
      end

      # Set execution mode
      #
      # @param mode [Symbol] Execution mode (:autonomous, :scheduled, :reactive)
      # @return [Symbol] Current execution mode
      def mode(mode = nil)
        return @execution_mode if mode.nil?
        @execution_mode = mode
      end

      # Execute the agent
      #
      # @return [void]
      def run!
        logger.info("Starting agent",
                    name: @name,
                    mode: @execution_mode,
                    objectives_count: @objectives.size,
                    has_workflow: !@workflow.nil?)

        case @execution_mode
        when :scheduled
          run_scheduled
        when :autonomous
          run_autonomous
        when :reactive
          run_reactive
        else
          logger.error("Unknown execution mode", mode: @execution_mode)
          raise "Unknown execution mode: #{@execution_mode}"
        end
      end

      private

      def logger_component
        "Agent:#{@name}"
      end

      def run_scheduled
        require 'rufus-scheduler'

        scheduler = Rufus::Scheduler.new

        logger.info("Scheduling agent",
                    name: @name,
                    cron: @schedule)

        scheduler.cron(@schedule) do
          logger.timed("Scheduled execution") do
            execute_objectives
          end
        end

        scheduler.join
      end

      def run_autonomous
        logger.info("Running agent in autonomous mode", name: @name)
        execute_objectives
      end

      def run_reactive
        logger.info("Running agent in reactive mode", name: @name)
        # Reactive mode implementation (event-driven)
        # This would be implemented with event listeners
        logger.error("Reactive mode not implemented")
        raise NotImplementedError, "Reactive mode not yet implemented"
      end

      def execute_objectives
        logger.info("Executing objectives",
                    total: @objectives.size,
                    has_workflow: !@workflow.nil?)

        @objectives.each_with_index do |objective, index|
          logger.info("Executing objective",
                      index: index + 1,
                      total: @objectives.size,
                      objective: objective[0..100])

          # If workflow defined, execute it; otherwise just log
          if @workflow
            logger.timed("Objective workflow execution") do
              @workflow.execute(objective)
            end
          else
            logger.warn("No workflow defined, skipping execution")
          end
        end

        logger.info("All objectives completed", total: @objectives.size)
      end
    end

    # Helper class for building constraints
    class ConstraintBuilder
      def initialize
        @constraints = {}
      end

      def max_iterations(value)
        @constraints[:max_iterations] = value
      end

      def timeout(value)
        @constraints[:timeout] = value
      end

      def memory(value)
        @constraints[:memory] = value
      end

      def rate_limit(value)
        @constraints[:rate_limit] = value
      end

      def to_h
        @constraints
      end
    end

    # Helper class for building output configuration
    class OutputBuilder
      def initialize
        @config = {}
      end

      def workspace(path)
        @config[:workspace] = path
      end

      def slack(channel:)
        @config[:slack] = {channel: channel}
      end

      def email(to:)
        @config[:email] = {to: to}
      end

      def to_h
        @config
      end
    end
  end
end
