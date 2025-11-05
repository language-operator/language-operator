# frozen_string_literal: true

require_relative '../spec_helper'
require_relative 'support/aictl_helper'

RSpec.configure do |config|
  # Include E2E helper methods in all E2E specs
  config.include E2E::LanguageOperatorHelper, type: :e2e

  # Skip E2E tests by default (run with: rspec --tag e2e)
  config.filter_run_excluding type: :e2e unless ENV['E2E'] == 'true' || config.inclusion_filter[:e2e]

  # Clean up test resources after E2E tests
  config.after(:each, type: :e2e) do
    # Clean up resources created during test
    cleanup_test_resources('e2e-test') if defined?(cleanup_test_resources)
  end

  # Set longer timeouts for E2E tests
  config.around(:each, type: :e2e) do |example|
    # Default RSpec timeout is usually short; E2E tests need more time
    original_timeout = RSpec.configuration.default_retry_wait_time
    example.run
  ensure
    RSpec.configuration.default_retry_wait_time = original_timeout if defined?(original_timeout)
  end
end

# E2E test configuration
module E2E
  # Configuration for E2E tests
  class Config
    class << self
      # Test resource prefix to avoid collisions
      def test_prefix
        @test_prefix ||= "e2e-test-#{Time.now.to_i}"
      end

      # Reset test prefix (useful between test runs)
      def reset_test_prefix!
        @test_prefix = "e2e-test-#{Time.now.to_i}"
      end

      # Kubernetes namespace for E2E tests
      def test_namespace
        @test_namespace ||= ENV.fetch('E2E_NAMESPACE', 'default')
      end

      # Whether to skip cleanup (useful for debugging)
      def skip_cleanup?
        ENV['E2E_SKIP_CLEANUP'] == 'true'
      end

      # Timeout for agent synthesis
      def synthesis_timeout
        @synthesis_timeout ||= ENV.fetch('E2E_SYNTHESIS_TIMEOUT', '300').to_i
      end

      # Timeout for pod readiness
      def pod_ready_timeout
        @pod_ready_timeout ||= ENV.fetch('E2E_POD_TIMEOUT', '120').to_i
      end
    end
  end
end
