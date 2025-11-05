# frozen_string_literal: true

require 'ruby_llm'
require 'ruby_llm/mcp'
require 'json'
require_relative 'config'
require_relative '../logger'
require_relative '../loggable'
require_relative '../retryable'

module LanguageOperator
  module Client
    # Core MCP client that connects to multiple servers and manages LLM chat
    #
    # This class handles all the backend logic for connecting to MCP servers,
    # configuring the LLM, and managing chat sessions. It's designed to be
    # UI-agnostic and reusable across different interfaces (CLI, web, headless).
    #
    # @example Basic usage
    #   config = Config.load('config.yaml')
    #   client = Base.new(config)
    #   client.connect!
    #   response = client.send_message("What tools are available?")
    #
    # @example Streaming responses
    #   client.stream_message("Search for Ruby news") do |chunk|
    #     print chunk
    #   end
    class Base
      include LanguageOperator::Loggable
      include LanguageOperator::Retryable

      attr_reader :config, :clients, :chat

      # Initialize the client with configuration
      #
      # @param config [Hash, String] Configuration hash or path to YAML file
      def initialize(config)
        @config = config.is_a?(String) ? Config.load(config) : config
        @clients = []
        @chat = nil
        @debug = @config['debug'] || false

        logger.debug("Client initialized",
                      debug: @debug,
                      llm_provider: @config.dig('llm', 'provider'),
                      llm_model: @config.dig('llm', 'model'))
      end

      # Connect to all enabled MCP servers and configure LLM
      #
      # @return [Hash] Connection results with status and tool counts
      # @raise [RuntimeError] If LLM configuration fails
      def connect!
        configure_llm
        connect_mcp_servers
      end

      # Send a message and get the full response
      #
      # @param message [String] User message
      # @return [String] Assistant response
      # @raise [StandardError] If message fails
      def send_message(message)
        raise 'Not connected. Call #connect! first.' unless @chat

        @chat.ask(message)
      end

      # Stream a message and yield each chunk
      #
      # @param message [String] User message
      # @yield [String] Each chunk of the response
      # @raise [StandardError] If streaming fails
      def stream_message(message, &block)
        raise 'Not connected. Call #connect! first.' unless @chat

        # Note: RubyLLM may not support streaming yet, so we'll call ask and yield the full response
        response = @chat.ask(message)

        # Convert response to string if it's a RubyLLM::Message object
        response_text = response.respond_to?(:content) ? response.content : response.to_s

        block.call(response_text) if block_given?
        response_text
      end

      # Get all available tools from connected servers
      #
      # @return [Array] Array of tool objects
      def tools
        @clients.flat_map(&:tools)
      end

      # Get information about connected servers
      #
      # @return [Array<Hash>] Server information (name, url, tool_count)
      def servers_info
        @clients.map do |client|
          {
            name: client.name,
            tool_count: client.tools.length,
            tools: client.tools.map(&:name)
          }
        end
      end

      # Clear chat history while keeping MCP connections
      #
      # @return [void]
      def clear_history!
        llm_config = @config['llm']
        chat_params = build_chat_params(llm_config)
        @chat = RubyLLM.chat(**chat_params)

        all_tools = tools
        @chat.with_tools(*all_tools) unless all_tools.empty?
      end

      # Check if the client is connected
      #
      # @return [Boolean] True if connected to at least one server
      def connected?
        !@clients.empty? && !@chat.nil?
      end

      # Get debug mode status
      #
      # @return [Boolean] True if debug mode is enabled
      def debug?
        @debug
      end

      private

      def logger_component
        'Client'
      end

      # Configure RubyLLM with provider settings
      #
      # @raise [RuntimeError] If provider is unknown
      def configure_llm
        llm_config = @config['llm']
        provider = llm_config['provider']
        model = llm_config['model']
        timeout = llm_config['timeout'] || 300

        logger.info("Configuring LLM",
                     provider: provider,
                     model: model,
                     timeout: timeout)

        if provider == 'openai_compatible' && llm_config['endpoint']
          logger.debug("Using custom endpoint", endpoint: llm_config['endpoint'])
        end

        RubyLLM.configure do |config|
          case provider
          when 'openai'
            config.openai_api_key = llm_config['api_key']
          when 'openai_compatible'
            config.openai_api_key = llm_config['api_key'] || 'not-needed'
            config.openai_api_base = llm_config['endpoint']
          when 'anthropic'
            config.anthropic_api_key = llm_config['api_key']
          else
            logger.error("Unknown LLM provider", provider: provider)
            raise "Unknown provider: #{provider}"
          end

          # Set timeout for LLM inference (default 300 seconds for slow local models)
          # RubyLLM uses request_timeout to control HTTP request timeouts
          config.request_timeout = timeout if config.respond_to?(:request_timeout=)
        end

        # Configure MCP timeout separately (MCP has its own timeout setting)
        # MCP request_timeout is in milliseconds, default is 300000ms (5 minutes)
        RubyLLM::MCP.configure do |config|
          mcp_timeout_ms = timeout * 1000
          config.request_timeout = mcp_timeout_ms if config.respond_to?(:request_timeout=)
        end

        logger.info("LLM configuration complete")
      end

      # Connect to all enabled MCP servers
      #
      # @return [void]
      def connect_mcp_servers
        enabled_servers = @config['mcp_servers'].select { |s| s['enabled'] }

        raise 'No MCP servers enabled in config' if enabled_servers.empty?

        logger.info("Connecting to MCP servers", count: enabled_servers.length)

        all_tools = []

        enabled_servers.each do |server_config|
          client = connect_with_retry(server_config)
          next unless client

          @clients << client
          tool_count = client.tools.length
          all_tools.concat(client.tools)

          logger.info("MCP server connected",
                       server: server_config['name'],
                       tool_count: tool_count,
                       tools: client.tools.map(&:name))
        rescue StandardError => e
          logger.error("Error connecting to MCP server",
                       server: server_config['name'],
                       error: e.message)
          logger.debug("Connection error backtrace",
                       server: server_config['name'],
                       backtrace: e.backtrace.join("\n")) if @debug
        end

        logger.info("MCP connection summary",
                     connected_servers: @clients.length,
                     total_tools: all_tools.length)

        # Create chat with all collected tools
        llm_config = @config['llm']
        chat_params = build_chat_params(llm_config)
        @chat = RubyLLM.chat(**chat_params)

        @chat.with_tools(*all_tools) unless all_tools.empty?

        logger.info("Chat session initialized", with_tools: !all_tools.empty?)
      end

      # Connect to MCP server with exponential backoff retry logic
      #
      # @param server_config [Hash] Server configuration
      # @return [RubyLLM::MCP::Client, nil] Client if successful, nil if all retries failed
      def connect_with_retry(server_config)
        logger.debug("Attempting to connect to MCP server",
                     server: server_config['name'],
                     transport: server_config['transport'],
                     url: server_config['url'])

        with_retry_or_nil(
          max_attempts: 4, # 1 initial attempt + 3 retries
          base_delay: 1.0,
          max_delay: 30.0,
          on_retry: lambda { |error, attempt, delay|
            logger.warn("MCP server connection failed, retrying",
                        server: server_config['name'],
                        attempt: attempt,
                        max_attempts: 4,
                        error: error.message,
                        retry_delay: delay)
          },
          on_failure: lambda { |error, attempts|
            logger.error("MCP server connection failed after all retries",
                         server: server_config['name'],
                         attempts: attempts,
                         error: error.message)
            logger.debug("Final connection error backtrace",
                         server: server_config['name'],
                         backtrace: error.backtrace.join("\n")) if @debug
          }
        ) do
          client = RubyLLM::MCP.client(
            name: server_config['name'],
            transport_type: server_config['transport'].to_sym,
            config: {
              url: server_config['url']
            }
          )

          logger.info("Successfully connected to MCP server",
                      server: server_config['name'])
          client
        end
      end

      # Build chat parameters based on LLM config
      #
      # @param llm_config [Hash] LLM configuration
      # @return [Hash] Chat parameters for RubyLLM.chat
      def build_chat_params(llm_config)
        chat_params = { model: llm_config['model'] }
        if llm_config['provider'] == 'openai_compatible'
          chat_params[:provider] = :openai
          chat_params[:assume_model_exists] = true
        end

        chat_params
      end
    end
  end
end
