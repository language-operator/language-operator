# frozen_string_literal: true

require 'shellwords'

module Aictl
  module Dsl
    # Common helper methods for MCP tools
    #
    # Provides validation, formatting, and utility methods that can be used
    # within tool execute blocks. All methods are instance methods that get
    # mixed into the tool execution context.
    #
    # @example Using helpers in a tool
    #   tool "send_email" do
    #     execute do |params|
    #       error = validate_email(params["email"])
    #       return error if error
    #       # email is valid, proceed...
    #     end
    #   end
    module Helpers
      # Validate URL format
      #
      # @param url [String] URL to validate
      # @return [String, nil] Error message if invalid, nil if valid
      def validate_url(url)
        return 'Error: Invalid URL. Must start with http:// or https://' unless url =~ %r{^https?://}
        nil
      end

      # Validate phone number in E.164 format
      #
      # @param number [String] Phone number to validate
      # @return [String, nil] Error message if invalid, nil if valid
      def validate_phone(number)
        return 'Error: Invalid phone number format. Use E.164 format (e.g., +1234567890)' unless number =~ /^\+\d{10,15}$/
        nil
      end

      # Validate email address format
      #
      # @param email [String] Email address to validate
      # @return [String, nil] Error message if invalid, nil if valid
      def validate_email(email)
        return 'Error: Invalid email format' unless email =~ /\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i
        nil
      end

      # Shell escape for backtick execution (deprecated - use Shell.run instead)
      def shell_escape(str)
        Shellwords.escape(str.to_s)
      end

      # Run a command and return structured result
      def run_command(cmd)
        output = `#{cmd}`
        {
          success: $?.success?,
          output: output,
          exitcode: $?.exitstatus
        }
      end

      # Check required environment variables
      def env_required(*vars)
        missing = vars.reject { |v| ENV[v] }
        return "Error: Missing required environment variables: #{missing.join(', ')}" unless missing.empty?
        nil
      end

      # Get environment variable with fallbacks
      def env_get(*keys, default: nil)
        keys.each do |key|
          value = ENV[key]
          return value if value
        end
        default
      end

      # Truncate text to a maximum length
      def truncate(text, max_length: 2000, suffix: '...')
        return text if text.length <= max_length
        text[0...max_length] + suffix
      end

      # Parse comma-separated values
      def parse_csv(str)
        return [] if str.nil? || str.empty?
        str.split(',').map(&:strip).reject(&:empty?)
      end

      # Format error message
      def error(message)
        "Error: #{message}"
      end

      # Format success message
      def success(message)
        message
      end
    end
  end
end
