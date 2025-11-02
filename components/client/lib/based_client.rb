# frozen_string_literal: true

require 'langop/client'
require_relative 'based/client/version'

# Based MCP Client Library
#
# This is now a thin wrapper around the Langop SDK gem for backwards compatibility.
# All actual client logic lives in the langop gem which is pre-installed in the base image.
#
# @example Quick start
#   require 'based_client'
#
#   config = Based::Client::Config.load('config.yaml')
#   client = Based::Client::Base.new(config)
#   client.connect!
#
#   puts client.send_message("What tools are available?")
#
# @example With environment variables
#   ENV['OPENAI_ENDPOINT'] = 'http://localhost:11434/v1'
#   ENV['LLM_MODEL'] = 'llama3.2'
#
#   config = Based::Client::Config.from_env
#   client = Based::Client::Base.new(config)
#   client.connect!
module Based
  # Namespace wrapper for backwards compatibility
  #
  # Based::Client is now an alias for Langop::Client
  module Client
    # Use Langop::Client classes directly
    Base = ::Langop::Client::Base
    Config = ::Langop::Client::Config
  end
end

# Convenience module for creating clients
module BasedClient
  # Convenience method to create a client from config file or ENV
  #
  # @param config_path [String, nil] Path to config file, or nil for ENV
  # @return [Based::Client::Base] Configured client instance
  def self.create(config_path = nil)
    config = if config_path
               Based::Client::Config.load_with_fallback(config_path)
             else
               Based::Client::Config.from_env
             end

    Based::Client::Base.new(config)
  end
end
