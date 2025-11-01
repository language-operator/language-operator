# frozen_string_literal: true

require 'rufus-scheduler'
require_relative 'executor'

module Langop
  module Agent
    # Task Scheduler
    #
    # Handles scheduled and event-driven task execution using rufus-scheduler.
    #
    # @example
    #   scheduler = Scheduler.new(agent)
    #   scheduler.start
    class Scheduler
      attr_reader :agent, :rufus_scheduler

      # Initialize the scheduler
      #
      # @param agent [Langop::Agent::Base] The agent instance
      def initialize(agent)
        @agent = agent
        @rufus_scheduler = Rufus::Scheduler.new
        @executor = Executor.new(agent)
      end

      # Start the scheduler
      #
      # @return [void]
      def start
        puts "🕐 Agent starting in scheduled mode..."
        puts "📁 Workspace: #{@agent.workspace_path}"
        puts "🔌 Connected to #{@agent.servers_info.length} MCP server(s)"
        puts

        setup_schedules
        @rufus_scheduler.join
      end

      # Stop the scheduler
      #
      # @return [void]
      def stop
        @rufus_scheduler.shutdown
      end

      private

      # Setup schedules from config
      #
      # @return [void]
      def setup_schedules
        schedules = @agent.config.dig('agent', 'schedules') || []

        if schedules.empty?
          puts "⚠️  No schedules configured, using default daily schedule"
          setup_default_schedule
          return
        end

        schedules.each do |schedule|
          add_schedule(schedule)
        end
      end

      # Add a single schedule
      #
      # @param schedule [Hash] Schedule configuration
      # @return [void]
      def add_schedule(schedule)
        cron = schedule['cron']
        task = schedule['task']

        @rufus_scheduler.cron(cron) do
          puts "🕐 Executing scheduled task: #{task}"
          result = @executor.execute(task)
          puts "✅ Result: #{result[0..200]}#{'...' if result.length > 200}"
          puts
        end

        puts "📅 Scheduled: #{task} (#{cron})"
      end

      # Setup default daily schedule
      #
      # @return [void]
      def setup_default_schedule
        instructions = @agent.config.dig('agent', 'instructions') ||
                      "Check for updates and report status"

        @rufus_scheduler.cron('0 6 * * *') do
          puts "🕐 Executing daily task"
          result = @executor.execute(instructions)
          puts "✅ Result: #{result[0..200]}#{'...' if result.length > 200}"
          puts
        end

        puts "📅 Scheduled: Daily at 6:00 AM"
      end
    end
  end
end
