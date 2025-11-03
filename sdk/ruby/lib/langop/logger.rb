# frozen_string_literal: true

require 'logger'

module Langop
  # Structured logger with configurable output formats and levels
  #
  # Supports multiple output formats:
  # - :pretty (default): Human-readable with emojis and colors
  # - :text: Plain text with timestamps
  # - :json: Structured JSON output
  #
  # Environment variables:
  # - LOG_LEVEL: DEBUG, INFO, WARN, ERROR (default: INFO)
  # - LOG_FORMAT: pretty, text, json (default: pretty)
  # - LOG_TIMING: true/false - Include operation timing (default: true)
  class Logger
    LEVELS = {
      'DEBUG' => ::Logger::DEBUG,
      'INFO' => ::Logger::INFO,
      'WARN' => ::Logger::WARN,
      'ERROR' => ::Logger::ERROR
    }.freeze

    LEVEL_EMOJI = {
      'DEBUG' => '🔍',
      'INFO' => 'ℹ️ ',
      'WARN' => '⚠️ ',
      'ERROR' => '❌'
    }.freeze

    attr_reader :logger, :format, :show_timing

    def initialize(component: 'Langop', format: nil, level: nil)
      @component = component
      @format = format || ENV.fetch('LOG_FORMAT', 'pretty').to_sym
      @show_timing = ENV.fetch('LOG_TIMING', 'true') == 'true'

      log_level_name = level || ENV.fetch('LOG_LEVEL', 'INFO')
      log_level = LEVELS[log_level_name.upcase] || ::Logger::INFO

      @logger = ::Logger.new($stdout)
      @logger.level = log_level
      @logger.formatter = method(:format_message)
    end

    def debug(message, **metadata)
      log(:debug, message, **metadata)
    end

    def info(message, **metadata)
      log(:info, message, **metadata)
    end

    def warn(message, **metadata)
      log(:warn, message, **metadata)
    end

    def error(message, **metadata)
      log(:error, message, **metadata)
    end

    # Log with timing information
    def timed(message, **metadata)
      start_time = Time.now
      result = yield if block_given?
      duration = Time.now - start_time

      info(message, **metadata.merge(duration_s: duration.round(3)))
      result
    end

    private

    def log(severity, message, **metadata)
      @logger.send(severity) do
        case @format
        when :json
          format_json(severity, message, **metadata)
        when :text
          format_text(severity, message, **metadata)
        else
          format_pretty(severity, message, **metadata)
        end
      end
    end

    def format_message(severity, timestamp, progname, msg)
      msg # Already formatted by log method
    end

    def format_json(severity, message, **metadata)
      require 'json'
      JSON.generate({
        timestamp: Time.now.iso8601,
        level: severity.to_s.upcase,
        component: @component,
        message: message,
        **metadata
      })
    end

    def format_text(severity, message, **metadata)
      parts = [
        Time.now.strftime('%Y-%m-%d %H:%M:%S'),
        severity.to_s.upcase.ljust(5),
        "[#{@component}]",
        message
      ]

      metadata_str = format_metadata(**metadata)
      parts << metadata_str unless metadata_str.empty?

      parts.join(' ')
    end

    def format_pretty(severity, message, **metadata)
      emoji = LEVEL_EMOJI[severity.to_s.upcase] || '•'
      parts = [emoji, message]

      metadata_str = format_metadata(**metadata)
      parts << "(#{metadata_str})" unless metadata_str.empty?

      parts.join(' ')
    end

    def format_metadata(**metadata)
      return '' if metadata.empty?

      metadata.map do |key, value|
        if key == :duration_s && @show_timing
          "#{value}s"
        elsif value.is_a?(String) && value.length > 100
          "#{key}=#{value[0..97]}..."
        else
          "#{key}=#{value}"
        end
      end.join(', ')
    end
  end
end
