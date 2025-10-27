# frozen_string_literal: true

module Api
  class MessagesController < ActionController::API
    include ActionController::Live

    before_action :find_session

    def create
      response.headers['Content-Type'] = 'text/event-stream'
      response.headers['Cache-Control'] = 'no-cache'
      response.headers['X-Accel-Buffering'] = 'no'

      puts "=" * 80
      puts "🎯 CONTROLLER: Starting SSE stream"
      puts "=" * 80

      sse = SSE.new(response.stream)

      begin
        content = message_params[:content]

        puts "🎯 CONTROLLER: Got message content"
        puts "   Session ID: #{@session.id}"
        puts "   Content: #{content[0..100]}#{'...' if content.length > 100}"

        # Send user message confirmation immediately to establish connection
        sse.write({ type: 'user_message', content: content }, event: 'message')
        puts "✓ Sent user_message event"

        # Stream the assistant response
        accumulated_text = ''

        puts "🔄 CONTROLLER: Calling @session.stream_message..."
        stream_start_time = Time.current

        # Start a heartbeat thread to keep the connection alive
        last_activity = Time.current
        heartbeat_thread = Thread.new do
          loop do
            sleep 10  # Send heartbeat every 10 seconds
            if Time.current - last_activity > 10
              begin
                sse.write({ type: 'heartbeat', timestamp: Time.current.to_i }, event: 'ping')
              rescue IOError
                # Connection closed, stop heartbeat
                break
              end
            end
          end
        end

        begin
          @session.stream_message(content) do |chunk|
            accumulated_text += chunk
            last_activity = Time.current
            sse.write({ type: 'chunk', content: chunk }, event: 'message')
          end
        ensure
          # Stop heartbeat thread
          heartbeat_thread.kill if heartbeat_thread&.alive?
        end

        stream_elapsed = Time.current - stream_start_time
        puts "=" * 80
        puts "✅ CONTROLLER: stream_message returned after #{stream_elapsed.round(2)}s"
        puts "   Accumulated #{accumulated_text.length} chars"
        puts "=" * 80

        # Send completion event with final message
        sse.write({
          type: 'complete',
          message: {
            role: 'assistant',
            content: accumulated_text,
            timestamp: Time.current.iso8601
          },
          session_id: @session.id
        }, event: 'message')

      rescue StandardError => e
        puts "=" * 80
        puts "❌ CONTROLLER: Error occurred"
        puts "   Error: #{e.class} - #{e.message}"
        puts "   Backtrace:"
        puts e.backtrace.first(15).map { |line| "     #{line}" }.join("\n")
        puts "=" * 80
        sse.write({ type: 'error', error: e.message }, event: 'message')
      ensure
        sse.close
      end
    end

    private

    def find_session
      @session = ChatSession.find(params[:chat_id])
    rescue StandardError => e
      render json: { error: e.message }, status: :not_found
    end

    def message_params
      if request.get?
        # For GET requests (EventSource), params come from query string
        { content: params[:message][:content] }
      else
        # For POST requests, params come from request body
        params.require(:message).permit(:content)
      end
    end

    class SSE
      def initialize(stream)
        @stream = stream
      end

      def write(object, options = {})
        event = options[:event] || 'message'
        @stream.write("event: #{event}\n")
        @stream.write("data: #{object.to_json}\n\n")
      end

      def close
        @stream.close
      rescue IOError
        # Stream already closed
      end
    end
  end
end
