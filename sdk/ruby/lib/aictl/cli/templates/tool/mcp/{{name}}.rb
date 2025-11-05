# frozen_string_literal: true

require "aictl'

# Define your MCP tools here
Langop.define do
  tool '{{name}}' do
    description 'A sample tool - replace this with your implementation'

    parameter :input do
      type :string
      required true
      description 'Input for the tool'
    end

    execute do |params|
      "Received: #{params['input']}"
    end
  end

  # Add more tools here
  # tool 'another_tool' do
  #   description 'Another tool'
  #   execute do |params|
  #     "Result"
  #   end
  # end
end
