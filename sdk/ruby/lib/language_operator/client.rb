# frozen_string_literal: true

require_relative 'client/base'
require_relative 'client/config'

module LanguageOperator
  # MCP Client for connecting to and using MCP servers
  #
  # Provides a high-level interface for connecting to MCP servers,
  # querying available tools, and sending messages to language models
  # with tool calling capabilities.
  #
  # @example Basic usage
  #   client = LanguageOperator::Client::Base.new(config)
  #   client.connect!
  #   response = client.send_message("What can you do?")
  module Client
  end
end
