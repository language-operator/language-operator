# frozen_string_literal: true

require "aictl'

# Aictl Client - convenience wrapper
#
# This provides a simple interface to the Aictl SDK for client functionality.
# All logic is in the langop gem.
#
# @example Quick start
#   require "aictl_client'
#
#   config = Aictl::Client::Config.load('config.yaml')
#   client = Aictl::Client::Base.new(config)
#   client.connect!
#
#   puts client.send_message("What tools are available?")
#
# @example With environment variables
#   ENV['OPENAI_ENDPOINT'] = 'http://localhost:11434/v1'
#   ENV['LLM_MODEL'] = 'llama3.2'
#
#   config = Aictl::Client::Config.from_env
#   client = Aictl::Client::Base.new(config)
#   client.connect!

# Convenience module for creating clients
module AictlClient
  # Convenience method to create a client from config file or ENV
  #
  # @param config_path [String, nil] Path to config file, or nil for ENV
  # @return [Aictl::Client::Base] Configured client instance
  def self.create(config_path = nil)
    config = if config_path
               Aictl::Client::Config.load_with_fallback(config_path)
             else
               Aictl::Client::Config.from_env
             end

    Aictl::Client::Base.new(config)
  end
end
