#!/usr/bin/env ruby
# frozen_string_literal: true

require 'ruby_llm'
require 'ruby_llm/mcp'
require 'reline'
require 'json'
require 'yaml'

# Based MCP Chat Client
#
# An interactive chat interface that connects to multiple MCP (Model Context Protocol)
# servers and provides access to their tools through a conversational LLM interface.
#
# @example Basic usage
#   client = BasedChat.new
#   client.run
#
# @example With custom config
#   ENV['CONFIG_PATH'] = '/path/to/config.yaml'
#   client = BasedChat.new
#   client.run
class BasedChat
  # Initialize the chat client
  #
  # Loads configuration from YAML file or environment variables
  def initialize
    @config = load_config
    @clients = []
    @chat = nil
  end

  # Load configuration from YAML file or fallback to environment variables
  #
  # @return [Hash] Configuration hash
  # @example
  #   config = load_config
  #   config['llm']['model'] #=> "llama3.2"
  def load_config
    config_path = ENV.fetch('CONFIG_PATH', '/app/config/config.yaml')

    unless File.exist?(config_path)
      puts "⚠️  Config file not found at #{config_path}"
      puts 'Using environment variable fallback mode...'
      return build_env_config
    end

    YAML.load_file(config_path)
  rescue StandardError => e
    puts "❌ Error loading config: #{e.message}"
    puts 'Using environment variable fallback mode...'
    build_env_config
  end

  # Build configuration from environment variables
  #
  # @return [Hash] Configuration hash built from ENV
  # @example
  #   ENV['OPENAI_ENDPOINT'] = 'http://localhost:11434/v1'
  #   config = build_env_config
  #   config['llm']['endpoint'] #=> "http://localhost:11434/v1"
  def build_env_config
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

  # Detect LLM provider from environment variables
  #
  # @return [String] Provider name (openai_compatible, openai, or anthropic)
  # @raise [RuntimeError] If no API key or endpoint is found
  def detect_provider_from_env
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
  def default_model_from_env
    {
      'openai' => 'gpt-4',
      'openai_compatible' => 'gpt-3.5-turbo',
      'anthropic' => 'claude-3-5-sonnet-20241022'
    }[detect_provider_from_env]
  end

  # Configure RubyLLM with provider settings
  #
  # @raise [RuntimeError] If provider is unknown
  def configure_llm
    llm_config = @config['llm']
    provider = llm_config['provider']

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
        raise "Unknown provider: #{provider}"
      end
    end
  end

  # Connect to all enabled MCP servers
  #
  # @raise [RuntimeError] If connection fails
  def connect
    puts '🚀 Based MCP Chat Client Starting...'
    puts

    configure_llm
    llm_config = @config['llm']

    if llm_config['provider'] == 'openai_compatible'
      puts "📝 LLM: #{llm_config['model']} (custom endpoint: #{llm_config['endpoint']})"
    else
      puts "📝 LLM: #{llm_config['model']} (#{llm_config['provider']})"
    end
    puts

    enabled_servers = @config['mcp_servers'].select { |s| s['enabled'] }

    if enabled_servers.empty?
      puts '⚠️  No MCP servers enabled in config'
      return
    end

    puts "🔌 Connecting to #{enabled_servers.length} MCP server(s)..."
    puts

    all_tools = []

    enabled_servers.each do |server_config|
      puts "  • #{server_config['name']}: #{server_config['url']}"

      client = RubyLLM::MCP.client(
        name: server_config['name'],
        transport_type: server_config['transport'].to_sym,
        config: {
          url: server_config['url']
        }
      )

      @clients << client
      tools = client.tools

      if tools.empty?
        puts '    └─ No tools available'
      else
        puts "    └─ #{tools.length} tool(s): #{tools.map(&:name).join(', ')}"
        all_tools.concat(tools)
      end
    rescue StandardError => e
      puts "    └─ ❌ Error: #{e.message}"
      puts e.backtrace.join("\n") if @config['debug']
    end

    puts

    # Create chat with custom model support
    chat_params = { model: llm_config['model'] }
    if llm_config['provider'] == 'openai_compatible'
      chat_params[:provider] = :openai
      chat_params[:assume_model_exists] = true
    end
    @chat = RubyLLM.chat(**chat_params)

    if all_tools.empty?
      puts '⚠️  No tools available from any MCP server'
    else
      @chat.with_tools(*all_tools)
      puts "✅ Ready with #{all_tools.length} tool(s) from #{@clients.length} server(s)"
    end

    puts
  rescue StandardError => e
    puts "❌ Error during connection: #{e.message}"
    puts e.backtrace.join("\n") if @config['debug']
    exit 1
  end

  # Run the interactive chat loop
  #
  # @example
  #   client = BasedChat.new
  #   client.run
  def run
    connect

    puts 'Type your messages below. Commands:'
    puts '  /help     - Show available tools'
    puts '  /servers  - Show connected servers'
    puts '  /clear    - Clear chat history'
    puts '  /exit     - Exit the chat'
    puts

    loop do
      prompt = Reline.readline('You: ', true)
      break if prompt.nil?

      prompt = prompt.strip
      next if prompt.empty?

      case prompt
      when '/exit', '/quit', '/q'
        break
      when '/help'
        show_help
        next
      when '/servers'
        show_servers
        next
      when '/clear'
        clear_history
        next
      end

      begin
        puts
        print 'Assistant: '
        response = @chat.ask(prompt)
        puts response
        puts
      rescue StandardError => e
        puts "❌ Error: #{e.message}"
        puts e.backtrace.join("\n") if @config['debug']
        puts
      end
    end

    puts "\n👋 Goodbye!"
  end

  # Display available tools from all connected servers
  def show_help
    puts
    puts 'Available tools from MCP servers:'

    if @clients.empty?
      puts '  (none)'
    else
      @clients.each do |client|
        tools = client.tools
        next if tools.empty?

        puts "\n  #{client.name}:"
        tools.each do |tool|
          puts "    • #{tool.name}"
          puts "      #{tool.description}" if tool.description
        end
      end
    end
    puts
  end

  # Display connected MCP servers
  def show_servers
    puts
    puts 'Connected MCP servers:'

    if @clients.empty?
      puts '  (none)'
    else
      @clients.each do |client|
        tool_count = client.tools.length
        puts "  • #{client.name} - #{tool_count} tool(s)"
      end
    end
    puts
  end

  # Clear chat history and reinitialize with tools
  def clear_history
    llm_config = @config['llm']
    chat_params = { model: llm_config['model'] }
    if llm_config['provider'] == 'openai_compatible'
      chat_params[:provider] = :openai
      chat_params[:assume_model_exists] = true
    end
    @chat = RubyLLM.chat(**chat_params)

    all_tools = @clients.flat_map(&:tools)
    @chat.with_tools(*all_tools) unless all_tools.empty?

    puts '✅ Chat history cleared'
  end
end

# Run the chat client
if __FILE__ == $PROGRAM_NAME
  client = BasedChat.new
  client.run
end
