# frozen_string_literal: true

require 'yaml'

module LanguageOperator
  module Client
    # Configuration management for Langop MCP Client
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
            'endpoint' => parse_model_endpoint_from_env,
            'api_key' => ENV.fetch('OPENAI_API_KEY') { ENV.fetch('ANTHROPIC_API_KEY') { 'dummy-key-for-local-proxy' } }
          },
          'mcp_servers' => parse_mcp_servers_from_env,
          'debug' => ENV['DEBUG'] == 'true'
        }
      end

      # Parse model endpoint from environment variable
      #
      # Supports both MODEL_ENDPOINTS (comma-separated, uses first) and OPENAI_ENDPOINT
      #
      # @return [String, nil] Model endpoint URL
      def self.parse_model_endpoint_from_env
        # Support MODEL_ENDPOINTS (operator sets this)
        endpoints_env = ENV['MODEL_ENDPOINTS']
        if endpoints_env && !endpoints_env.empty?
          # Take the first endpoint from comma-separated list
          endpoints_env.split(',').first.strip
        else
          # Fallback to legacy OPENAI_ENDPOINT
          ENV.fetch('OPENAI_ENDPOINT', nil)
        end
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
        if ENV['OPENAI_ENDPOINT'] || ENV['MODEL_ENDPOINTS']
          'openai_compatible'
        elsif ENV['OPENAI_API_KEY']
          'openai'
        elsif ENV['ANTHROPIC_API_KEY']
          'anthropic'
        else
          raise 'No API key or endpoint found. Set OPENAI_ENDPOINT or MODEL_ENDPOINTS for local LLM, ' \
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

      # Parse MCP servers from environment variables
      #
      # Supports MCP_SERVERS env var as comma-separated URLs or single MCP_URL
      #
      # @return [Array<Hash>] Array of MCP server configurations
      def self.parse_mcp_servers_from_env
        # Support both MCP_SERVERS (comma-separated) and legacy MCP_URL
        servers_env = ENV['MCP_SERVERS']
        if servers_env && !servers_env.empty?
          # Parse comma-separated URLs
          servers_env.split(',').map.with_index do |url, index|
            {
              'name' => "default-tools-#{index}",
              'url' => url.strip,
              'transport' => 'streamable',
              'enabled' => true
            }
          end
        elsif ENV['MCP_URL']
          [{
            'name' => 'default-tools',
            'url' => ENV['MCP_URL'],
            'transport' => 'streamable',
            'enabled' => true
          }]
        else
          []
        end
      end
    end
  end
end
