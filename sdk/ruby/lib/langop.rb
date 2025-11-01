# frozen_string_literal: true

require_relative 'langop/version'
require_relative 'langop/dsl'
require_relative 'langop/client'
require_relative 'langop/agent'

# Langop - Ruby SDK for building MCP tools and language agents
#
# This gem provides:
# - DSL for defining MCP tools with a clean, Ruby-like syntax
# - Client library for connecting to MCP servers
# - Agent framework for autonomous task execution
# - CLI for generating and running tools and agents
#
# @example Define a tool
#   Langop::Dsl.define do
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
#   config = Langop::Client::Config.from_env
#   client = Langop::Client::Base.new(config)
#   client.connect!
#   response = client.send_message("What can you do?")
#
# @example Run an agent
#   agent = Langop::Agent::Base.new(config)
#   agent.run
module Langop
  class Error < StandardError; end

  # Convenience method to define tools
  #
  # @yield Block containing tool definitions
  # @return [Langop::Dsl::Registry] The global tool registry
  def self.define(&block)
    Dsl.define(&block)
  end

  # Convenience method to load tools from a file
  #
  # @param file_path [String] Path to tool definition file
  # @return [Langop::Dsl::Registry] The global tool registry
  def self.load_file(file_path)
    Dsl.load_file(file_path)
  end
end
