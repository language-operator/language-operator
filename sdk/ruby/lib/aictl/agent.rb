# frozen_string_literal: true

require_relative 'agent/base'
require_relative 'agent/executor'
require_relative 'agent/scheduler'

module Aictl
  # Agent Framework
  #
  # Provides autonomous execution capabilities for language agents.
  # Extends Aictl::Client with agent-specific features like scheduling,
  # goal evaluation, and workspace integration.
  #
  # @example Running an agent
  #   config = Aictl::Client::Config.from_env
  #   agent = Aictl::Agent::Base.new(config)
  #   agent.run
  #
  # @example Creating a custom agent
  #   agent = Aictl::Agent::Base.new(config)
  #   agent.execute_goal("Summarize daily news")
  module Agent
    # Run the default agent based on environment configuration
    #
    # @param config_path [String] Path to configuration file
    # @return [void]
    def self.run(config_path: nil)
      # Disable stdout buffering for real-time logging in containers
      $stdout.sync = true
      $stderr.sync = true

      config_path ||= ENV.fetch('CONFIG_PATH', 'config.yaml')
      config = Aictl::Client::Config.load_with_fallback(config_path)

      agent = Aictl::Agent::Base.new(config)
      agent.run
    end
  end
end
