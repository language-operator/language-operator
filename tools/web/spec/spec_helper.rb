require 'bundler/setup'

# Start SimpleCov before loading application code
# require 'simplecov'
# SimpleCov.start do
#   add_filter '/spec/'
#   add_filter '/vendor/'
# 
#   add_group 'Tools', 'tools'
# 
#   minimum_coverage 80
#   minimum_coverage_by_file 70
# end

require 'language_operator'
require 'webmock/rspec'

# Disable real HTTP connections
WebMock.disable_net_connect!(allow_localhost: false)

RSpec.configure do |config|
  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups
  config.filter_run_when_matching :focus
  config.example_status_persistence_file_path = "spec/examples.txt"
  config.disable_monkey_patching!
  config.warnings = true

  if config.files_to_run.one?
    config.default_formatter = "doc"
  end

  config.order = :random
  Kernel.srand config.seed
end
