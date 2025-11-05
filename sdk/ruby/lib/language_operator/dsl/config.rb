# frozen_string_literal: true

module LanguageOperator
  module Dsl
    # Configuration helper for managing environment variables
    #
    # Provides utilities for reading and managing environment variables with fallback support,
    # type conversion, and validation. All methods are class methods.
    #
    # @example Basic usage
    #   Config.get('SMTP_HOST', 'MAIL_HOST', default: 'localhost')
    #   Config.require('DATABASE_URL')
    #   Config.get_bool('USE_TLS', default: true)
    class Config
      # Get environment variable with multiple fallback keys
      #
      # @param keys [Array<String>] Environment variable names to try
      # @param default [Object, nil] Default value if none found
      # @return [String, nil] The first non-nil value or default
      def self.get(*keys, default: nil)
        keys.each do |key|
          value = ENV[key.to_s]
          return value if value
        end
        default
      end

      # Get required environment variable with fallback keys
      #
      # @param keys [Array<String>] Environment variable names to try
      # @return [String] The first non-nil value
      # @raise [ArgumentError] If none of the keys are set
      def self.require(*keys)
        value = get(*keys)
        raise ArgumentError, "Missing required configuration: #{keys.join(' or ')}" unless value
        value
      end

      # Get environment variable as integer
      #
      # @param keys [Array<String>] Environment variable names to try
      # @param default [Integer, nil] Default value if none found
      # @return [Integer, nil] The value converted to integer, or default
      def self.get_int(*keys, default: nil)
        value = get(*keys)
        return default if value.nil?
        value.to_i
      end

      # Get environment variable as boolean
      #
      # Treats 'true', '1', 'yes', 'on' as true (case insensitive).
      #
      # @param keys [Array<String>] Environment variable names to try
      # @param default [Boolean] Default value if none found
      # @return [Boolean] The value as boolean
      def self.get_bool(*keys, default: false)
        value = get(*keys)
        return default if value.nil?

        value.to_s.downcase.match?(/^(true|1|yes|on)$/)
      end

      # Get environment variable as array (split by separator)
      #
      # @param keys [Array<String>] Environment variable names to try
      # @param default [Array] Default value if none found
      # @param separator [String] Character to split on (default: ',')
      # @return [Array<String>] The value split into array
      def self.get_array(*keys, default: [], separator: ',')
        value = get(*keys)
        return default if value.nil? || value.empty?

        value.split(separator).map(&:strip).reject(&:empty?)
      end

      # Check if all required keys are present
      #
      # @param keys [Array<String>] Environment variable names to check
      # @return [Array<String>] Array of missing keys (empty if all present)
      def self.check_required(*keys)
        keys.reject { |key| ENV[key.to_s] }
      end

      # Check if environment variable is set (even if empty string)
      #
      # @param keys [Array<String>] Environment variable names to check
      # @return [Boolean] True if any key is set
      def self.set?(*keys)
        keys.any? { |key| ENV.key?(key.to_s) }
      end

      # Get all environment variables matching a prefix
      #
      # @param prefix [String] Prefix to match
      # @return [Hash<String, String>] Hash with prefix removed from keys
      def self.with_prefix(prefix)
        ENV.select { |key, _| key.start_with?(prefix) }
           .transform_keys { |key| key.sub(prefix, '') }
      end

      # Build a configuration hash from environment variables
      #
      # @param mappings [Hash{Symbol => Array<String>, String}] Config key to env var(s) mapping
      # @return [Hash{Symbol => String}] Configuration hash with values found
      def self.build(mappings)
        config = {}
        mappings.each do |config_key, env_keys|
          env_keys = [env_keys] unless env_keys.is_a?(Array)
          value = get(*env_keys)
          config[config_key] = value if value
        end
        config
      end
    end
  end
end
