# frozen_string_literal: true

module Api
  class ServersController < BaseController
    def index
      with_mcp_client do |client|
        servers = client.servers_info
        render json: { servers: servers, count: servers.length }
      end
    end
  end
end
