# frozen_string_literal: true

require 'reline'

# Terminal interface for Based MCP Client
#
# Provides an interactive REPL (Read-Eval-Print Loop) for chatting with the LLM
# and using MCP tools. This interface is designed to be simple and familiar to
# users of command-line tools.
#
# @example Basic usage
#   client = BasedClient.create('config/config.yaml')
#   interface = CLIInterface.new(client)
#   interface.run
class CLIInterface
  # Initialize the CLI interface
  #
  # @param client [Based::Client::Base] Configured client instance
  def initialize(client)
    @client = client
  end

  # Run the interactive chat loop
  #
  # @return [void]
  def run
    display_welcome
    connect_client
    display_ready_message
    interactive_loop
    display_goodbye
  end

  private

  # Display welcome banner
  #
  # @return [void]
  def display_welcome
    puts '🚀 Based MCP Chat Client Starting...'
    puts
  end

  # Connect to MCP servers and LLM
  #
  # @return [void]
  def connect_client
    @client.connect!

    llm_config = @client.config['llm']
    if llm_config['provider'] == 'openai_compatible'
      puts "📝 LLM: #{llm_config['model']} (custom endpoint: #{llm_config['endpoint']})"
    else
      puts "📝 LLM: #{llm_config['model']} (#{llm_config['provider']})"
    end
    puts
  rescue StandardError => e
    puts "❌ Error during connection: #{e.message}"
    puts e.backtrace.join("\n") if @client.debug?
    exit 1
  end

  # Display ready message with server and tool counts
  #
  # @return [void]
  def display_ready_message
    servers_info = @client.servers_info

    if servers_info.empty?
      puts '⚠️  No MCP servers connected'
      return
    end

    puts "🔌 Connected to #{servers_info.length} MCP server(s):"
    puts

    servers_info.each do |server|
      puts "  • #{server[:name]}: #{server[:url] rescue 'N/A'}"
      if server[:tool_count].zero?
        puts '    └─ No tools available'
      else
        puts "    └─ #{server[:tool_count]} tool(s): #{server[:tools].join(', ')}"
      end
    end

    puts
    total_tools = servers_info.sum { |s| s[:tool_count] }
    if total_tools.zero?
      puts '⚠️  No tools available from any MCP server'
    else
      puts "✅ Ready with #{total_tools} tool(s) from #{servers_info.length} server(s)"
    end
    puts
  end

  # Main interactive loop
  #
  # @return [void]
  def interactive_loop
    display_help_commands

    loop do
      prompt = Reline.readline('You: ', true)
      break if prompt.nil?

      prompt = prompt.strip
      next if prompt.empty?

      break if handle_command(prompt)

      send_message(prompt)
    end
  end

  # Display available commands
  #
  # @return [void]
  def display_help_commands
    puts 'Type your messages below. Commands:'
    puts '  /help     - Show available tools'
    puts '  /servers  - Show connected servers'
    puts '  /clear    - Clear chat history'
    puts '  /exit     - Exit the chat'
    puts
  end

  # Handle special commands
  #
  # @param input [String] User input
  # @return [Boolean] True if should exit, false otherwise
  def handle_command(input)
    case input
    when '/exit', '/quit', '/q'
      true
    when '/help'
      show_help
      false
    when '/servers'
      show_servers
      false
    when '/clear'
      clear_history
      false
    else
      false
    end
  end

  # Send message to LLM and display response
  #
  # @param message [String] User message
  # @return [void]
  def send_message(message)
    puts
    print 'Assistant: '

    response = @client.send_message(message)
    puts response
    puts
  rescue StandardError => e
    puts "❌ Error: #{e.message}"
    puts e.backtrace.join("\n") if @client.debug?
    puts
  end

  # Display available tools from all connected servers
  #
  # @return [void]
  def show_help
    puts
    puts 'Available tools from MCP servers:'

    servers_info = @client.servers_info

    if servers_info.empty?
      puts '  (none)'
    else
      # Need to get actual tool objects for descriptions
      @client.clients.each do |mcp_client|
        tools = mcp_client.tools
        next if tools.empty?

        puts "\n  #{mcp_client.name}:"
        tools.each do |tool|
          puts "    • #{tool.name}"
          puts "      #{tool.description}" if tool.description
        end
      end
    end
    puts
  end

  # Display connected MCP servers
  #
  # @return [void]
  def show_servers
    puts
    puts 'Connected MCP servers:'

    servers_info = @client.servers_info

    if servers_info.empty?
      puts '  (none)'
    else
      servers_info.each do |server|
        puts "  • #{server[:name]} - #{server[:tool_count]} tool(s)"
      end
    end
    puts
  end

  # Clear chat history
  #
  # @return [void]
  def clear_history
    @client.clear_history!
    puts '✅ Chat history cleared'
  end

  # Display goodbye message
  #
  # @return [void]
  def display_goodbye
    puts "\n👋 Goodbye!"
  end
end
