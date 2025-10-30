# frozen_string_literal: true

require_relative 'based/client/version'
require_relative 'based/client/config'
require_relative 'based/client/base'

# Based MCP Client Library
#
# A reusable backend library for connecting to MCP (Model Context Protocol)
# servers and managing LLM chat sessions. This library is UI-agnostic and
# can be used by CLI, web, or headless agents.
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
