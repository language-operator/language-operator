# frozen_string_literal: true

require 'based_client'

class ChatSession
  MAX_SESSIONS = 100
  SESSION_EXPIRY = 24.hours

  attr_reader :id, :client, :messages, :created_at, :last_activity_at

  @@sessions = {}
  @@mutex = Mutex.new

  def self.create(config)
    cleanup_expired_sessions

    @@mutex.synchronize do
      raise 'Maximum sessions reached' if @@sessions.length >= MAX_SESSIONS

      session = new(config)
      @@sessions[session.id] = session
      session
    end
  end

  def self.find(id)
    @@mutex.synchronize do
      session = @@sessions[id]
      raise "Session not found: #{id}" unless session
      raise "Session expired: #{id}" if session.expired?

      session
    end
  end

  def self.destroy(id)
    @@mutex.synchronize do
      @@sessions.delete(id)
    end
  end

  def self.cleanup_expired_sessions
    @@mutex.synchronize do
      @@sessions.delete_if { |_id, session| session.expired? }
    end
  end

  def initialize(config)
    @id = SecureRandom.uuid
    @client = Based::Client::Base.new(config)
    @client.connect!
    @messages = []
    @created_at = Time.current
    @last_activity_at = Time.current
  end

  def send_message(content)
    touch

    user_message = {
      role: 'user',
      content: content,
      timestamp: Time.current
    }
    @messages << user_message

    response_content = @client.send_message(content)

    assistant_message = {
      role: 'assistant',
      content: response_content,
      timestamp: Time.current
    }
    @messages << assistant_message

    { message: assistant_message, session_id: @id }
  end

  def stream_message(content, &block)
    touch

    user_message = {
      role: 'user',
      content: content,
      timestamp: Time.current
    }
    @messages << user_message

    accumulated_content = ''

    puts "=" * 80
    puts "📨 Starting LLM request for session #{@id}"
    puts "   Content: #{content[0..100]}#{'...' if content.length > 100}"
    puts "=" * 80
    start_time = Time.current

    begin
      @client.stream_message(content) do |chunk|
        accumulated_content += chunk
        block.call(chunk) if block_given?
      end

      elapsed = Time.current - start_time
      puts "=" * 80
      puts "✅ LLM request completed in #{elapsed.round(2)}s for session #{@id}"
      puts "   Response length: #{accumulated_content.length} chars"
      puts "=" * 80
    rescue => e
      elapsed = Time.current - start_time
      puts "=" * 80
      puts "❌ LLM request failed after #{elapsed.round(2)}s for session #{@id}"
      puts "   Error: #{e.class} - #{e.message}"
      puts "   Backtrace:"
      puts e.backtrace.first(10).map { |line| "     #{line}" }.join("\n")
      puts "=" * 80
      raise
    end

    assistant_message = {
      role: 'assistant',
      content: accumulated_content,
      timestamp: Time.current
    }
    @messages << assistant_message

    assistant_message
  end

  def clear_history!
    touch
    @messages.clear
    @client.clear_history!
  end

  def info
    {
      id: @id,
      created_at: @created_at,
      last_activity_at: @last_activity_at,
      message_count: @messages.length,
      servers: @client.servers_info
    }
  end

  def expired?
    Time.current - @last_activity_at > SESSION_EXPIRY
  end

  private

  def touch
    @last_activity_at = Time.current
  end
end
