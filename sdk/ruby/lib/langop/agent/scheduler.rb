# frozen_string_literal: true

require 'rufus-scheduler'
require_relative 'executor'
require_relative '../logger'
require_relative '../loggable'

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
      include Langop::Loggable

      attr_reader :agent, :rufus_scheduler

      # Initialize the scheduler
      #
      # @param agent [Langop::Agent::Base] The agent instance
      def initialize(agent)
        @agent = agent
        @rufus_scheduler = Rufus::Scheduler.new
        @executor = Executor.new(agent)

        logger.debug("Scheduler initialized",
                      workspace: @agent.workspace_path,
                      servers: @agent.servers_info.length)
      end

      # Start the scheduler
      #
      # @return [void]
      def start
        logger.info("Agent starting in scheduled mode")
        logger.info("Workspace: #{@agent.workspace_path}")
        logger.info("Connected to #{@agent.servers_info.length} MCP server(s)")

        setup_schedules
        logger.info("Scheduler started, waiting for scheduled tasks")
        @rufus_scheduler.join
      end

      # Stop the scheduler
      #
      # @return [void]
      def stop
        logger.info("Shutting down scheduler")
        @rufus_scheduler.shutdown
        logger.info("Scheduler stopped")
      end

      private

      def logger_component
        'Agent::Scheduler'
      end

      # Setup schedules from config
      #
      # @return [void]
      def setup_schedules
        schedules = @agent.config.dig('agent', 'schedules') || []

        logger.debug("Loading schedules from config", count: schedules.length)

        if schedules.empty?
          logger.warn("No schedules configured, using default daily schedule")
          setup_default_schedule
          return
        end

        schedules.each do |schedule|
          add_schedule(schedule)
        end

        logger.info("#{schedules.length} schedule(s) configured")
      end

      # Add a single schedule
      #
      # @param schedule [Hash] Schedule configuration
      # @return [void]
      def add_schedule(schedule)
        cron = schedule['cron']
        task = schedule['task']

        logger.info("Scheduling task", cron: cron, task: task[0..100])

        @rufus_scheduler.cron(cron) do
          logger.timed("Scheduled task execution") do
            logger.info("Executing scheduled task", task: task[0..100])
            result = @executor.execute(task)
            preview = result[0..200]
            preview += '...' if result.length > 200
            logger.info("Task completed", result_preview: preview)
          end
        end
      end

      # Setup default daily schedule
      #
      # @return [void]
      def setup_default_schedule
        instructions = @agent.config.dig('agent', 'instructions') ||
                      "Check for updates and report status"

        logger.info("Setting up default schedule", cron: '0 6 * * *',
                     instructions: instructions[0..100])

        @rufus_scheduler.cron('0 6 * * *') do
          logger.timed("Daily task execution") do
            logger.info("Executing daily task")
            result = @executor.execute(instructions)
            preview = result[0..200]
            preview += '...' if result.length > 200
            logger.info("Daily task completed", result_preview: preview)
          end
        end

        logger.info("Scheduled: Daily at 6:00 AM")
      end
    end
  end
end
