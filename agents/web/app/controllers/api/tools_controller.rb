# frozen_string_literal: true

module Api
  class ToolsController < BaseController
    def index
      with_mcp_client do |client|
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
      end
    end
  end
end
