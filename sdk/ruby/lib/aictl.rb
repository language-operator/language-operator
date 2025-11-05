# frozen_string_literal: true

require "aictl/version"
require "aictl/dsl"
require "aictl/client"
require "aictl/tool_loader"

# Agent module is optional - only load if dependencies are available
# This allows the SDK to be used in environments without agent dependencies
begin
  require "aictl/agent"
rescue LoadError => e
  # Agent dependencies not available, skip loading
  warn "Langop: Agent module not loaded (missing dependency: #{e.message})" if ENV['DEBUG']
end

# Langop - Ruby SDK for building MCP tools and language agents
#
# This gem provides:
# - DSL for defining MCP tools with a clean, Ruby-like syntax
# - Client library for connecting to MCP servers
# - Agent framework for autonomous task execution
# - CLI for generating and running tools and agents
#
# @example Define a tool
#   Aictl::Dsl.define do
#     tool "greet" do
#       description "Greet a user"
#       parameter :name do
#         type :string
#         required true
#       end
#       execute do |params|
#         "Hello, #{params['name']}!"
#       end
#     end
#   end
#
# @example Use the client
#   config = Aictl::Client::Config.from_env
#   client = Aictl::Client::Base.new(config)
#   client.connect!
#   response = client.send_message("What can you do?")
#
# @example Run an agent
#   agent = Aictl::Agent::Base.new(config)
#   agent.run
module Aictl
  class Error < StandardError; end

  # Convenience method to define tools
  #
  # @yield Block containing tool definitions
  # @return [Aictl::Dsl::Registry] The global tool registry
  def self.define(&block)
    Dsl.define(&block)
  end

  # Convenience method to load tools from a file
  #
  # @param file_path [String] Path to tool definition file
  # @return [Aictl::Dsl::Registry] The global tool registry
  def self.load_file(file_path)
    Dsl.load_file(file_path)
  end
end
