# frozen_string_literal: true

require_relative '../logger'

module Langop
  module Agent
    # Task Executor
    #
    # Handles autonomous task execution with retry logic and error handling.
    #
    # @example
    #   executor = Executor.new(agent)
    #   executor.execute("Complete the task")
    class Executor
      attr_reader :agent, :iteration_count

      # Initialize the executor
      #
      # @param agent [Langop::Agent::Base] The agent instance
      def initialize(agent)
        @agent = agent
        @iteration_count = 0
        @max_iterations = 100
        @logger = Langop::Logger.new(component: 'Agent::Executor')
        @show_full_responses = ENV.fetch('SHOW_FULL_RESPONSES', 'false') == 'true'
      end

      # Execute a single task
      #
      # @param task [String] The task to execute
      # @return [String] The result
      def execute(task)
        @iteration_count += 1

        @logger.info("Starting iteration #{@iteration_count}")
        @logger.debug("Prompt", prompt: task[0..200])

        result = @logger.timed("LLM request completed") do
          @agent.send_message(task)
        end

        result
      rescue StandardError => e
        handle_error(e)
      end

      # Run continuous execution loop
      #
      # @return [void]
      def run_loop
        @logger.info("Agent starting in autonomous mode")
        @logger.info("Workspace: #{@agent.workspace_path}")
        @logger.info("Connected to #{@agent.servers_info.length} MCP server(s)")

        # Log MCP server details
        if @agent.servers_info.any?
          @agent.servers_info.each do |server|
            @logger.info("  MCP server", name: server[:name], url: server[:url])
          end
        end

        # Get initial instructions from config or environment
        instructions = @agent.config.dig('agent', 'instructions') ||
                      ENV['AGENT_INSTRUCTIONS'] ||
                      "Monitor workspace and respond to changes"

        @logger.info("Instructions: #{instructions}")
        @logger.info("")

        loop do
          break if @iteration_count >= @max_iterations

          result = execute(instructions)
          result_text = result.is_a?(String) ? result : result.content

          # Log result based on verbosity settings
          if @show_full_responses
            @logger.info("Iteration #{@iteration_count} result:\n#{result_text}")
          else
            preview = result_text[0..200]
            preview += '...' if result_text.length > 200
            @logger.info("Iteration #{@iteration_count} result: #{preview}")
          end

          @logger.info("")

          # Rate limiting
          sleep 5
        end

        @logger.warn("Maximum iterations (#{@max_iterations}) reached")
      end

      private

      def handle_error(error)
        case error
        when Timeout::Error, /timeout/i.match?(error.message)
          @logger.error("Request timeout",
                       error: error.class.name,
                       message: error.message,
                       iteration: @iteration_count)
        when /connection refused|operation not permitted/i.match?(error.message)
          @logger.error("Connection failed",
                       error: error.class.name,
                       message: error.message,
                       hint: "Check if model service is healthy and accessible")
        else
          @logger.error("Task execution failed",
                       error: error.class.name,
                       message: error.message)
          @logger.debug("Backtrace", trace: error.backtrace[0..5].join("\n")) if error.backtrace
        end

        "Error executing task: #{error.message}"
      end
    end
  end
end
