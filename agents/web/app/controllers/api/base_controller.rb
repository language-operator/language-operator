# frozen_string_literal: true

module Api
  class BaseController < ActionController::API
    before_action :load_mcp_config

    private

    def load_mcp_config
      config_path = Rails.root.join('config', 'mcp.yml')
      @mcp_config = YAML.load_file(config_path, aliases: true)
    end

    def with_mcp_client
      client = Langop::Client::Base.new(@mcp_config)
      client.connect!
      yield(client)
    rescue StandardError => e
      render json: { error: e.message }, status: :internal_server_error
    end
  end
end
