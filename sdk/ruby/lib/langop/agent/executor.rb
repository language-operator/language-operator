# frozen_string_literal: true

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
      end

      # Execute a single task
      #
      # @param task [String] The task to execute
      # @return [String] The result
      def execute(task)
        @iteration_count += 1
        @agent.send_message(task)
      rescue StandardError => e
        "Error executing task: #{e.message}"
      end

      # Run continuous execution loop
      #
      # @return [void]
      def run_loop
        puts "🤖 Agent starting in autonomous mode..."
        puts "📁 Workspace: #{@agent.workspace_path}"
        puts "🔌 Connected to #{@agent.servers_info.length} MCP server(s)"
        puts

        # Get initial instructions from config or environment
        instructions = @agent.config.dig('agent', 'instructions') ||
                      ENV['AGENT_INSTRUCTIONS'] ||
                      "Monitor workspace and respond to changes"

        puts "📋 Instructions: #{instructions}"
        puts

        loop do
          break if @iteration_count >= @max_iterations

          result = execute(instructions)
          puts "#{@iteration_count}. ✅ #{result[0..200]}#{'...' if result.length > 200}"
          puts

          # Rate limiting
          sleep 5
        end

        puts "⚠️  Maximum iterations (#{@max_iterations}) reached"
      end
    end
  end
end
