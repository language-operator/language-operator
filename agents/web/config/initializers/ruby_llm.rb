# frozen_string_literal: true

# Configure HTTP timeouts for RubyLLM HTTP requests
# This prevents Net::ReadTimeout errors on long-running LLM requests

puts "=" * 80
puts "🔧 INITIALIZER STARTING: ruby_llm.rb"
puts "=" * 80

# Monkey-patch Net::HTTP to use longer timeouts for LLM requests
require 'net/http'

module Net
  class HTTP
    # Store original methods
    alias_method :original_initialize, :initialize
    alias_method :original_read_timeout=, :read_timeout= if method_defined?(:read_timeout=)

    def initialize(*args, **kwargs)
      original_initialize(*args, **kwargs)

      # Set longer timeouts for LLM requests (10 minutes)
      @open_timeout = 30      # Connection timeout
      @read_timeout = 600     # Read timeout (10 minutes)
      @write_timeout = 60     # Write timeout

      puts "🌐 Net::HTTP instance created with timeouts: open=#{@open_timeout}s, read=#{@read_timeout}s, write=#{@write_timeout}s"
    end

    # Override the read_timeout setter to prevent it from being changed
    def read_timeout=(value)
      if value && value < 600
        puts "⚠️  WARNING: Attempt to set read_timeout to #{value}s, enforcing minimum of 600s"
        original_read_timeout=(600) if respond_to?(:original_read_timeout=)
      else
        original_read_timeout=(value) if respond_to?(:original_read_timeout=)
      end
    end

    # Override the getter to ensure it always returns at least 600
    alias_method :original_read_timeout, :read_timeout if method_defined?(:read_timeout)

    def read_timeout
      current = original_read_timeout if respond_to?(:original_read_timeout)
      current ||= @read_timeout
      if current && current < 600
        puts "⚠️  WARNING: read_timeout is #{current}s, returning 600s instead"
        600
      else
        current
      end
    end
  end
end

puts "✓ Net::HTTP monkey-patch applied"

# Monkey-patch HTTPX to use longer timeouts by default
begin
  puts "🔧 Attempting to monkey-patch HTTPX..."

  module HTTPX
    # Override the session creation to inject custom timeout settings
    class << self
      alias_method :original_new_session, :new_session if method_defined?(:new_session)

      def new_session(*, **options)
        # Merge our custom timeout settings
        custom_options = {
          timeout: {
            connect_timeout: 30,
            read_timeout: 600,      # 10 minutes
            write_timeout: 60,
            request_timeout: 600    # 10 minutes total
          }
        }

        merged_options = custom_options.merge(options)

        puts "🌐 Creating HTTPX session with timeouts: #{merged_options[:timeout]}"

        if respond_to?(:original_new_session)
          original_new_session(*, **merged_options)
        else
          # Fallback if method doesn't exist
          session_class.new(**merged_options)
        end
      end
    end
  end

  puts "✓ HTTPX session monkey-patch applied"
rescue => e
  puts "❌ ERROR patching HTTPX session: #{e.class} - #{e.message}"
  puts e.backtrace.first(5).join("\n")
end

# Also configure default timeout options for HTTPX
begin
  puts "🔧 Attempting to patch HTTPX::Options..."

  HTTPX::Options.instance_eval do
    alias_method :original_initialize, :initialize if method_defined?(:initialize)

    define_method(:initialize) do |options = {}|
      default_timeouts = {
        connect_timeout: 30,
        read_timeout: 600,
        write_timeout: 60,
        request_timeout: 600
      }

      options[:timeout] = default_timeouts.merge(options[:timeout] || {})

      puts "⚙️  HTTPX::Options initialized with timeout: #{options[:timeout]}"

      if respond_to?(:original_initialize)
        original_initialize(options)
      else
        super(options)
      end
    end
  end

  puts "✓ HTTPX::Options monkey-patch applied"
rescue => e
  puts "❌ ERROR patching HTTPX::Options: #{e.class} - #{e.message}"
  puts e.backtrace.first(5).join("\n")
end

puts "=" * 80
puts "✅ INITIALIZER COMPLETE: ruby_llm.rb"
puts "=" * 80
