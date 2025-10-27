# frozen_string_literal: true

require 'yaml'

module Based
  module Client
    # Configuration management for Based MCP Client
    #
    # Handles loading configuration from YAML files or environment variables,
    # with automatic provider detection and sensible defaults.
    #
    # @example Load from YAML file
    #   config = Config.load('/path/to/config.yaml')
    #
    # @example Load from environment variables
    #   config = Config.from_env
    #
    # @example Load with fallback
    #   config = Config.load_with_fallback('/path/to/config.yaml')
    class Config
      # Load configuration from a YAML file
      #
      # @param path [String] Path to YAML configuration file
      # @return [Hash] Configuration hash
      # @raise [Errno::ENOENT] If file doesn't exist
      def self.load(path)
        YAML.load_file(path)
      end

      # Load configuration from environment variables
      #
      # @return [Hash] Configuration hash built from ENV
      def self.from_env
        {
          'llm' => {
            'provider' => detect_provider_from_env,
            'model' => ENV.fetch('LLM_MODEL') { default_model_from_env },
            'endpoint' => ENV.fetch('OPENAI_ENDPOINT', nil),
            'api_key' => ENV.fetch('OPENAI_API_KEY') { ENV.fetch('ANTHROPIC_API_KEY', nil) }
          },
          'mcp_servers' => [
            {
              'name' => 'default',
              'url' => ENV.fetch('MCP_URL', 'http://server:80/mcp'),
              'transport' => 'streamable',
              'enabled' => true
            }
          ],
          'debug' => ENV['DEBUG'] == 'true'
        }
      end

      # Load configuration with automatic fallback to environment variables
      #
      # @param path [String] Path to YAML configuration file
      # @return [Hash] Configuration hash
      def self.load_with_fallback(path)
        return from_env unless File.exist?(path)

        load(path)
      rescue StandardError => e
        warn "⚠️  Error loading config from #{path}: #{e.message}"
        warn 'Using environment variable fallback mode...'
        from_env
      end

      # Detect LLM provider from environment variables
      #
      # @return [String] Provider name (openai_compatible, openai, or anthropic)
      # @raise [RuntimeError] If no API key or endpoint is found
      def self.detect_provider_from_env
        if ENV['OPENAI_ENDPOINT']
          'openai_compatible'
        elsif ENV['OPENAI_API_KEY']
          'openai'
        elsif ENV['ANTHROPIC_API_KEY']
          'anthropic'
        else
          raise 'No API key or endpoint found. Set OPENAI_ENDPOINT for local LLM, ' \
                'or OPENAI_API_KEY/ANTHROPIC_API_KEY for cloud providers.'
        end
      end

      # Get default model for detected provider
      #
      # @return [String] Default model name
      def self.default_model_from_env
        {
          'openai' => 'gpt-4',
          'openai_compatible' => 'gpt-3.5-turbo',
          'anthropic' => 'claude-3-5-sonnet-20241022'
        }[detect_provider_from_env]
      end
    end
  end
end
