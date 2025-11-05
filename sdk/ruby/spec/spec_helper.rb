# frozen_string_literal: true

require 'bundler/setup'
require 'language_operator'
require 'webmock/rspec'
require 'tmpdir'
require 'tempfile'

# Disable external HTTP requests
WebMock.disable_net_connect!(allow_localhost: false)

RSpec.configure do |config|
  # Enable flags like --only-failures and --next-failure
  config.example_status_persistence_file_path = '.rspec_status'

  # Disable RSpec exposing methods globally on `Module` and `main`
  config.disable_monkey_patching!

  config.expect_with :rspec do |c|
    c.syntax = :expect
  end

  # Note: Registry cleanup not needed as each test creates its own tool classes

  # Mock RubyLLM and RubyLLM::MCP configuration to avoid real API calls
  config.before(:each) do
    allow(RubyLLM).to receive(:configure).and_yield(double(
                                                       anthropic_api_key: nil,
                                                       openai_api_key: nil,
                                                       request_timeout: nil,
                                                       respond_to?: true
                                                     ))

    allow(RubyLLM::MCP).to receive(:configure).and_yield(double(
                                                            request_timeout: nil,
                                                            respond_to?: true
                                                          ))
  end

  # Clean up environment variables after each test
  config.after(:each) do
    %w[
      WORKSPACE_PATH
      AGENT_MODE
      AGENT_INSTRUCTIONS
      TEST_VAR
      SHOW_FULL_RESPONSES
      LOG_LEVEL
      LOG_FORMAT
    ].each { |var| ENV.delete(var) }
  end
end
