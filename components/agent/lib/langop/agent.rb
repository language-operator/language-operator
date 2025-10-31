# frozen_string_literal: true

require 'based_client'
require_relative 'agent/version'
require_relative 'agent/base'
require_relative 'agent/executor'
require_relative 'agent/scheduler'

module Langop
  # Agent Framework
  #
  # Provides autonomous execution capabilities for language agents.
  # Extends Based::Client with agent-specific features like scheduling,
  # goal evaluation, and workspace integration.
  #
  # @example Running an agent
  #   Langop::Agent.run
  #
  # @example Creating a custom agent
  #   agent = Langop::Agent::Base.new(config)
  #   agent.execute_goal("Summarize daily news")
  module Agent
    # Run the default agent based on environment configuration
    #
    # @return [void]
    def self.run
      config_path = ENV.fetch('CONFIG_PATH', '/app/agent/config/config.yaml')
      config = Based::Client::Config.load_with_fallback(config_path)

      agent = Langop::Agent::Base.new(config)
      agent.run
    end
  end
end
