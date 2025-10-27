# frozen_string_literal: true

module Api
  class ChatsController < ActionController::API
    before_action :load_mcp_config
    before_action :find_session, only: %i[show destroy history clear_history]

    def create
      session = ChatSession.create(@mcp_config)
      render json: session.info, status: :created
    rescue StandardError => e
      render json: { error: e.message }, status: :unprocessable_entity
    end

    def show
      render json: @session.info
    end

    def destroy
      ChatSession.destroy(@session.id)
      head :no_content
    end

    def history
      render json: { session_id: @session.id, messages: @session.messages }
    end

    def clear_history
      @session.clear_history!
      head :no_content
    end

    private

    def load_mcp_config
      config_path = Rails.root.join('config', 'mcp.yml')
      config_yaml = File.read(config_path)

      # Substitute environment variables in the YAML
      config_yaml.gsub!(/\$\{(\w+)\}/) { ENV.fetch($1, '') }

      @mcp_config = YAML.safe_load(config_yaml, aliases: true)
    end

    def find_session
      @session = ChatSession.find(params[:id])
    rescue StandardError => e
      render json: { error: e.message }, status: :not_found
    end
  end
end
