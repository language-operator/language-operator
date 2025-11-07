# frozen_string_literal: true

require 'language_operator'
require_relative 'goal_evaluator'

# Autonomous headless agent
#
# Runs in a continuous loop to achieve specified goals using MCP tools.
# Exits when all goals are complete or timeout/max iterations reached.
class HeadlessAgent
  # Initialize the headless agent
  #
  # @param config [Hash] MCP client configuration
  # @param goals [Hash] Goals configuration
  def initialize(config, goals)
    @config = config
    @goals = goals
    @client = LanguageOperator::Client::Base.new(config)
    @evaluator = GoalEvaluator.new(goals)
  end

  # Run the autonomous agent loop
  #
  # @return [void]
  def run
    display_startup
    connect_client
    execute_goal_loop
    display_summary
  end

  private

  # Display startup banner
  #
  # @return [void]
  def display_startup
    puts '🤖 Langop Headless Agent Starting...'
    puts
    puts "📋 Objectives (#{@goals['objectives']&.length || 0}):"
    @goals['objectives']&.each_with_index do |objective, index|
      puts "   #{index + 1}. #{objective}"
    end
    puts
  end

  # Connect to MCP servers and LLM
  #
  # @return [void]
  def connect_client
    @client.connect!

    servers = @client.servers_info
    total_tools = servers.sum { |s| s[:tool_count] }

    puts "🔌 Connected to #{servers.length} MCP server(s)"
    puts "🛠️  Available tools: #{total_tools}"
    puts
  rescue StandardError => e
    puts "❌ Connection error: #{e.message}"
    exit 1
  end

  # Execute the main goal achievement loop
  #
  # @return [void]
  def execute_goal_loop
    puts '🚀 Starting autonomous execution...'
    puts

    loop do
      break if @evaluator.all_goals_complete?

      task = @evaluator.next_task
      break unless task

      execute_task(task)

      # Rate limiting
      sleep 1
    end
  end

  # Execute a single task
  #
  # @param task [String] Task description
  # @return [void]
  def execute_task(task)
    iteration = @evaluator.iterations + 1
    puts "#{iteration}. 🎯 Task: #{task}"

    begin
      response = @client.send_message(task)
      puts "   ✅ Response: #{response[0..200]}#{'...' if response.length > 200}"
      puts

      @evaluator.record_result(task, response)
    rescue StandardError => e
      puts "   ❌ Error: #{e.message}"
      puts

      @evaluator.record_result(task, "ERROR: #{e.message}")
    end
  end

  # Display final summary
  #
  # @return [void]
  def display_summary
    progress = @evaluator.progress

    puts '=' * 60
    puts '🎉 Execution Complete!'
    puts
    puts 'Summary:'
    puts "  Iterations: #{progress[:iterations]} / #{progress[:max_iterations]}"
    puts "  Objectives: #{progress[:completed]} / #{progress[:total_objectives]} completed"
    puts "  Elapsed time: #{progress[:elapsed_minutes].round(2)} minutes"
    puts

    if progress[:completed] >= progress[:total_objectives]
      puts '✅ All objectives completed successfully!'
    elsif progress[:iterations] >= progress[:max_iterations]
      puts '⚠️  Stopped: Maximum iterations reached'
    elsif progress[:elapsed_minutes] >= progress[:timeout_minutes]
      puts '⚠️  Stopped: Timeout exceeded'
    else
      puts '⚠️  Stopped: Unknown reason'
    end

    puts '=' * 60
  end
end
