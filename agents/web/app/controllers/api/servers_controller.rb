# frozen_string_literal: true

module Api
  class ServersController < ActionController::API
    before_action :load_mcp_config

    def index
      # Create a temporary client to get server info
      client = Based::Client::Base.new(@mcp_config)
      client.connect!
      servers = client.servers_info

      render json: { servers: servers, count: servers.length }
    rescue StandardError => e
      render json: { error: e.message }, status: :internal_server_error
    end

    private

    def load_mcp_config
      config_path = Rails.root.join('config', 'mcp.yml')
      @mcp_config = YAML.load_file(config_path, aliases: true)
    end
  end
end
