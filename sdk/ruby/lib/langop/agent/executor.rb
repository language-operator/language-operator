# frozen_string_literal: true

require_relative '../logger'
require_relative '../loggable'

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
      include Langop::Loggable

      attr_reader :agent, :iteration_count

      # Initialize the executor
      #
      # @param agent [Langop::Agent::Base] The agent instance
      def initialize(agent)
        @agent = agent
        @iteration_count = 0
        @max_iterations = 100
        @show_full_responses = ENV.fetch('SHOW_FULL_RESPONSES', 'false') == 'true'

        logger.debug("Executor initialized",
                     max_iterations: @max_iterations,
                     show_full_responses: @show_full_responses,
                     workspace: @agent.workspace_path)
      end

      # Execute a single task or workflow
      #
      # @param task [String] The task to execute
      # @param agent_definition [Langop::Dsl::AgentDefinition, nil] Optional agent definition with workflow
      # @return [String] The result
      def execute(task, agent_definition: nil)
        @iteration_count += 1

        # Route to workflow execution if agent has a workflow defined
        if agent_definition&.workflow
          return execute_workflow(agent_definition)
        end

        # Standard instruction-based execution
        logger.info("Starting iteration",
                    iteration: @iteration_count,
                    max_iterations: @max_iterations)
        logger.debug("Prompt", prompt: task[0..200])

        result = logger.timed("LLM request") do
          @agent.send_message(task)
        end

        result_text = result.is_a?(String) ? result : result.content
        logger.info("Iteration completed",
                    iteration: @iteration_count,
                    response_length: result_text.length)
        logger.debug("Response preview", response: result_text[0..200])

        result
      rescue StandardError => e
        handle_error(e)
      end

      # Run continuous execution loop
      #
      # @return [void]
      def run_loop
        logger.info("Agent starting in autonomous mode")
        logger.info("Configuration",
                    workspace: @agent.workspace_path,
                    mcp_servers: @agent.servers_info.length,
                    max_iterations: @max_iterations)

        # Log MCP server details
        if @agent.servers_info.any?
          @agent.servers_info.each do |server|
            logger.info("  MCP server", name: server[:name], tool_count: server[:tool_count])
          end
        end

        # Get initial instructions from config or environment
        instructions = @agent.config.dig('agent', 'instructions') ||
                      ENV['AGENT_INSTRUCTIONS'] ||
                      "Monitor workspace and respond to changes"

        logger.info("Instructions", instructions: instructions[0..200])
        logger.info("Starting autonomous execution loop")

        loop do
          break if @iteration_count >= @max_iterations

          progress_pct = ((@iteration_count.to_f / @max_iterations) * 100).round(1)
          logger.debug("Loop progress",
                       iteration: @iteration_count,
                       max: @max_iterations,
                       progress: "#{progress_pct}%")

          result = execute(instructions)
          result_text = result.is_a?(String) ? result : result.content

          # Log result based on verbosity settings
          if @show_full_responses
            logger.info("Full iteration result",
                        iteration: @iteration_count,
                        result: result_text)
          else
            preview = result_text[0..200]
            preview += '...' if result_text.length > 200
            logger.info("Iteration result",
                        iteration: @iteration_count,
                        preview: preview)
          end

          # Rate limiting
          logger.debug("Rate limit pause", duration: 5)
          sleep 5
        end

        logger.warn("Maximum iterations reached",
                    iterations: @max_iterations,
                    reason: "Hit max_iterations limit")
      end

      # Execute a workflow-based agent
      #
      # @param agent_def [Langop::Dsl::AgentDefinition] The agent definition
      # @return [RubyLLM::Message] The final response
      def execute_workflow(agent_def)
        logger.info("🔄 Starting workflow execution: #{agent_def.name}")

        # Build orchestration prompt from agent definition
        prompt = build_workflow_prompt(agent_def)
        logger.debug("Workflow prompt", prompt: prompt[0..300])

        # Register workflow steps as tools (placeholder - will implement after tool converter)
        # For now, just execute with instructions
        result = logger.timed("Workflow execution completed") do
          @agent.send_message(prompt)
        end

        # Write output if configured
        if agent_def.output_config && result
          write_output(agent_def, result)
        end

        logger.info("✅ Workflow execution completed")
        result
      rescue StandardError => e
        logger.error("Workflow execution failed", error: e.message)
        handle_error(e)
      end

      # Build orchestration prompt from agent definition
      #
      # @param agent_def [Langop::Dsl::AgentDefinition] The agent definition
      # @return [String] The prompt
      def build_workflow_prompt(agent_def)
        prompt = "# Task: #{agent_def.description}\n\n"

        if agent_def.objectives && agent_def.objectives.any?
          prompt += "## Objectives:\n"
          agent_def.objectives.each { |obj| prompt += "- #{obj}\n" }
          prompt += "\n"
        end

        if agent_def.workflow && agent_def.workflow.steps.any?
          prompt += "## Workflow Steps:\n"
          agent_def.workflow.step_order.each do |step_name|
            step = agent_def.workflow.steps[step_name]
            prompt += "#{step_name.to_s.tr('_', ' ').capitalize}"
            if step.tool_name
              prompt += " (using tool: #{step.tool_name})"
            end
            if step.dependencies && step.dependencies.any?
              prompt += " - depends on: #{step.dependencies.join(', ')}"
            end
            prompt += "\n"
          end
          prompt += "\n"
        end

        if agent_def.constraints
          prompt += "## Constraints:\n"
          if agent_def.constraints[:max_iterations]
            prompt += "- Maximum iterations: #{agent_def.constraints[:max_iterations]}\n"
          end
          if agent_def.constraints[:timeout]
            prompt += "- Timeout: #{agent_def.constraints[:timeout]}\n"
          end
          prompt += "\n"
        end

        prompt += "Please complete this task following the workflow steps."
        prompt
      end

      # Write output to configured destinations
      #
      # @param agent_def [Langop::Dsl::AgentDefinition] The agent definition
      # @param result [RubyLLM::Message] The result to write
      def write_output(agent_def, result)
        return unless agent_def.output_config

        content = result.is_a?(String) ? result : result.content

        if workspace_path = agent_def.output_config[:workspace]
          full_path = File.join(@agent.workspace_path, workspace_path)

          begin
            FileUtils.mkdir_p(File.dirname(full_path))
            File.write(full_path, content)
            logger.info("📝 Wrote output to #{workspace_path}")
          rescue Errno::EACCES, Errno::EPERM => e
            # Permission denied - try writing to workspace root
            fallback_path = File.join(@agent.workspace_path, 'output.txt')
            begin
              File.write(fallback_path, content)
              logger.warn("⚠️  Could not write to #{workspace_path}, wrote to output.txt instead")
            rescue StandardError => e2
              logger.warn("⚠️  Could not write output to workspace: #{e2.message}")
              logger.info("📄 Output (first 500 chars): #{content[0..500]}")
            end
          end
        end

        # Future: Handle Slack, email outputs
      rescue StandardError => e
        logger.warn("Output writing failed", error: e.message)
      end

      private

      def logger_component
        'Agent::Executor'
      end

      def handle_error(error)
        case error
        when Timeout::Error, /timeout/i.match?(error.message)
          logger.error("Request timeout",
                       error: error.class.name,
                       message: error.message,
                       iteration: @iteration_count)
        when /connection refused|operation not permitted/i.match?(error.message)
          logger.error("Connection failed",
                       error: error.class.name,
                       message: error.message,
                       hint: "Check if model service is healthy and accessible")
        else
          logger.error("Task execution failed",
                       error: error.class.name,
                       message: error.message)
          logger.debug("Backtrace", trace: error.backtrace[0..5].join("\n")) if error.backtrace
        end

        "Error executing task: #{error.message}"
      end
    end
  end
end
