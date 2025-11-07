# frozen_string_literal: true

module Api
  class ToolsController < ActionController::API
    before_action :load_mcp_config

    def index
      # Create a temporary client to get tools
      client = Langop::Client::Base.new(@mcp_config)
      client.connect!
      tools = client.tools

      render json: {
        tools: tools.map { |tool|
          {
            name: tool.name,
            description: tool.description,
            input_schema: tool.input_schema
          }
        },
        count: tools.length
      }
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
