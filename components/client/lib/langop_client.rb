# frozen_string_literal: true

require 'langop'

# Langop Client - convenience wrapper
#
# This provides a simple interface to the Langop SDK for client functionality.
# All logic is in the langop gem.
#
# @example Quick start
#   require 'langop_client'
#
#   config = Langop::Client::Config.load('config.yaml')
#   client = Langop::Client::Base.new(config)
#   client.connect!
#
#   puts client.send_message("What tools are available?")
#
# @example With environment variables
#   ENV['OPENAI_ENDPOINT'] = 'http://localhost:11434/v1'
#   ENV['LLM_MODEL'] = 'llama3.2'
#
#   config = Langop::Client::Config.from_env
#   client = Langop::Client::Base.new(config)
#   client.connect!

# Convenience module for creating clients
module LangopClient
  # Convenience method to create a client from config file or ENV
  #
  # @param config_path [String, nil] Path to config file, or nil for ENV
  # @return [Langop::Client::Base] Configured client instance
  def self.create(config_path = nil)
    config = if config_path
               Langop::Client::Config.load_with_fallback(config_path)
             else
               Langop::Client::Config.from_env
             end

    Langop::Client::Base.new(config)
  end
end
