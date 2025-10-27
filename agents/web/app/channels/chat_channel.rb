# frozen_string_literal: true

class ChatChannel < ApplicationCable::Channel
  def subscribed
    @session_id = params[:session_id]
    stream_for @session_id
  end

  def unsubscribed
    # Cleanup when channel is unsubscribed
  end

  def receive(data)
    session = ChatSession.find(@session_id)

    # Echo user message
    transmit(type: 'user_message', content: data['message'])

    # Send to LLM and stream response
    result = session.send_message(data['message'])

    # Echo assistant message
    transmit(type: 'assistant_message', content: result[:message][:content])
  rescue StandardError => e
    transmit(type: 'error', message: e.message)
  end
end
