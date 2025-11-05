# frozen_string_literal: true

require_relative '../client'

module Aictl
  module Agent
    # Base Agent Class
    #
    # Extends Aictl::Client::Base with agent-specific functionality including:
    # - Workspace integration
    # - Goal-directed execution
    # - Autonomous operation modes
    #
    # @example Basic agent
    #   agent = Aictl::Agent::Base.new(config)
    #   agent.connect!
    #   agent.execute_goal("Complete the task")
    class Base < Aictl::Client::Base
      attr_reader :workspace_path, :mode

      # Initialize the agent
      #
      # @param config [Hash] Configuration hash
      def initialize(config)
        super(config)
        @workspace_path = ENV.fetch('WORKSPACE_PATH', '/workspace')
        @mode = ENV.fetch('AGENT_MODE', 'autonomous')
        @executor = nil
        @scheduler = nil
      end

      # Run the agent in its configured mode
      #
      # @return [void]
      def run
        connect!

        case @mode
        when 'autonomous', 'interactive'
          run_autonomous
        when 'scheduled', 'event-driven'
          run_scheduled
        else
          raise "Unknown agent mode: #{@mode}"
        end
      end

      # Execute a single goal
      #
      # @param goal [String] The goal to achieve
      # @return [String] The result
      def execute_goal(goal)
        @executor ||= Executor.new(self)
        @executor.execute(goal)
      end

      # Check if workspace is available
      #
      # @return [Boolean]
      def workspace_available?
        File.directory?(@workspace_path) && File.writable?(@workspace_path)
      end

      private

      # Run in autonomous mode
      #
      # @return [void]
      def run_autonomous
        @executor = Executor.new(self)
        @executor.run_loop
      end

      # Run in scheduled mode
      #
      # @return [void]
      def run_scheduled
        @scheduler = Scheduler.new(self)
        @scheduler.start
      end
    end
  end
end
