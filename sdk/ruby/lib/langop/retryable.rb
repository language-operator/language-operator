# frozen_string_literal: true

module Langop
  # Mixin module to provide retry logic with exponential backoff for operations
  # that may fail transiently.
  #
  # @example Basic usage
  #   class MyService
  #     include Langop::Retryable
  #
  #     def connect
  #       with_retry do
  #         # Connection logic that might fail
  #         make_connection
  #       end
  #     end
  #   end
  #
  # @example Custom retry configuration
  #   with_retry(max_attempts: 5, base_delay: 2.0) do
  #     risky_operation
  #   end
  #
  # @example With custom error handling
  #   with_retry(on_retry: ->(error, attempt) { log_error(error, attempt) }) do
  #     api_call
  #   end
  module Retryable
    # Default retry configuration
    DEFAULT_MAX_ATTEMPTS = 3
    DEFAULT_BASE_DELAY = 1.0
    DEFAULT_MAX_DELAY = 30.0

    # Execute a block with retry logic and exponential backoff.
    #
    # @param max_attempts [Integer] Maximum number of attempts (default: 3)
    # @param base_delay [Float] Base delay in seconds for exponential backoff (default: 1.0)
    # @param max_delay [Float] Maximum delay between retries in seconds (default: 30.0)
    # @param rescue_errors [Array<Class>] Array of error classes to rescue (default: StandardError)
    # @param on_retry [Proc] Optional callback called on each retry attempt with (error, attempt, delay)
    # @yield Block to execute with retry logic
    # @return Result of the block if successful
    # @raise Last error encountered if all retries are exhausted
    #
    # @example
    #   result = with_retry(max_attempts: 5) do
    #     fetch_from_api
    #   end
    def with_retry(
      max_attempts: DEFAULT_MAX_ATTEMPTS,
      base_delay: DEFAULT_BASE_DELAY,
      max_delay: DEFAULT_MAX_DELAY,
      rescue_errors: [StandardError],
      on_retry: nil
    )
      attempt = 0
      last_error = nil

      loop do
        attempt += 1

        begin
          return yield
        rescue *rescue_errors => e
          last_error = e

          if attempt >= max_attempts
            raise e
          end

          # Calculate delay with exponential backoff: base_delay * 2^(attempt-1)
          delay = [base_delay * (2**(attempt - 1)), max_delay].min

          # Call the retry callback if provided
          on_retry&.call(e, attempt, delay)

          sleep(delay)
        end
      end
    end

    # Execute a block with retry logic and return nil on failure instead of raising.
    #
    # @param max_attempts [Integer] Maximum number of attempts (default: 3)
    # @param base_delay [Float] Base delay in seconds for exponential backoff (default: 1.0)
    # @param max_delay [Float] Maximum delay between retries in seconds (default: 30.0)
    # @param rescue_errors [Array<Class>] Array of error classes to rescue (default: StandardError)
    # @param on_retry [Proc] Optional callback called on each retry attempt with (error, attempt, delay)
    # @param on_failure [Proc] Optional callback called when all retries are exhausted with (error, attempts)
    # @yield Block to execute with retry logic
    # @return Result of the block if successful, nil if all retries failed
    #
    # @example
    #   result = with_retry_or_nil(on_failure: ->(err, tries) { log_failure(err) }) do
    #     optional_operation
    #   end
    #
    #   return unless result
    def with_retry_or_nil(
      max_attempts: DEFAULT_MAX_ATTEMPTS,
      base_delay: DEFAULT_BASE_DELAY,
      max_delay: DEFAULT_MAX_DELAY,
      rescue_errors: [StandardError],
      on_retry: nil,
      on_failure: nil
    )
      attempt = 0
      last_error = nil

      loop do
        attempt += 1

        begin
          return yield
        rescue *rescue_errors => e
          last_error = e

          if attempt >= max_attempts
            on_failure&.call(e, attempt)
            return nil
          end

          # Calculate delay with exponential backoff
          delay = [base_delay * (2**(attempt - 1)), max_delay].min

          # Call the retry callback if provided
          on_retry&.call(e, attempt, delay)

          sleep(delay)
        end
      end
    end
  end
end
