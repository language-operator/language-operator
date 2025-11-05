require 'sinatra'
require 'sinatra/json'
require 'json'
require 'rack/protection'

# Monkeypatch Rack::Protection::HostAuthorization to allow all hosts
# This is needed for Docker internal hostnames (doc, email, sms, web, etc.)
module Rack
  module Protection
    class HostAuthorization
      def accepts?(env)
        true  # Always accept all hosts for Docker networking
      end
    end
  end
end

# Load Aictl DSL wrapper (which loads langop gem including ToolLoader)
require "aictl/dsl'

# Aictl MCP Server using official Ruby SDK
module Aictl
  class Server < Sinatra::Base
  helpers Sinatra::JSON

  set :port, ENV.fetch('PORT', 80)
  set :bind, '0.0.0.0'
  set :public_folder, '/app/doc'
  set :static, true

  # Completely disable Rack::Protection for Docker networking
  # Don't use any protection middleware
  set :protection, nil

  # Explicitly skip loading protection middleware
  def self.protection?
    false
  end

  # Enable CORS for MCP clients
  before do
    headers['Access-Control-Allow-Origin'] = '*'
    headers['Access-Control-Allow-Methods'] = 'GET, POST, DELETE, OPTIONS'
    headers['Access-Control-Allow-Headers'] = 'Content-Type, Accept, MCP-Session-ID'
  end

  options '*' do
    200
  end

  # Initialize registry and load tools
  configure do
    # Create our DSL registry and load tools
    registry = Aictl::Dsl::Registry.new
    loader = ToolLoader.new(registry)
    loader.load_tools

    # Create MCP::Server from our registry
    mcp_server = Aictl::Dsl::Adapter.create_mcp_server(
      registry,
      server_name: 'based-mcp',
      server_context: {}
    )

    # Set up Streamable HTTP transport
    transport = MCP::Server::Transports::StreamableHTTPTransport.new(mcp_server)
    mcp_server.transport = transport

    # Store both for flexibility
    set :registry, registry
    set :loader, loader
    set :mcp_server, mcp_server
    set :transport, transport
  end

  # Health check
  get '/health' do
    json status: 'ok', tool_count: settings.registry.all.length
  end

  # Serve documentation - redirect /doc to /doc/
  get '/doc' do
    redirect '/doc/'
  end

  # Serve documentation index
  get '/doc/' do
    send_file File.join(settings.public_folder, 'index.html')
  end

  # Serve documentation files
  get '/doc/*' do
    path = params['splat'].first
    file_path = File.join(settings.public_folder, path)
    if File.exist?(file_path) && File.file?(file_path)
      send_file file_path
    else
      halt 404
    end
  end

  # MCP Protocol: Handle all HTTP methods via StreamableHTTPTransport
  # GET - SSE stream for a session
  # POST - Initialize or send JSON-RPC requests
  # DELETE - Close session
  ['/mcp', '/*'].each do |path|
    # GET - SSE stream
    get path do
      status_code, headers_hash, body_array = settings.transport.handle_request(request)
      status status_code
      headers_hash.each { |k, v| headers[k] = v }

      # If it's an SSE stream, body_array will contain a stream object
      if headers_hash['Content-Type'] == 'text/event-stream'
        body_array.first  # Return the stream proc
      else
        body_array.join
      end
    end

    # POST - JSON-RPC
    post path do
      status_code, headers_hash, body_array = settings.transport.handle_request(request)
      status status_code
      headers_hash.each { |k, v| headers[k] = v }
      body_array.join
    end

    # DELETE - Close session
    delete path do
      status_code, headers_hash, body_array = settings.transport.handle_request(request)
      status status_code
      headers_hash.each { |k, v| headers[k] = v }
      body_array.join
    end
  end

  # Reload tools (useful for development) - custom endpoint
  post '/reload' do
    # Reload tools in registry
    settings.loader.reload

    # Recreate MCP server with updated tools
    mcp_server = Aictl::Dsl::Adapter.create_mcp_server(
      settings.registry,
      server_name: 'based-mcp',
      server_context: {}
    )

    set :mcp_server, mcp_server

    json status: 'reloaded', tool_count: settings.registry.all.length
  end

  # List loaded tools (debug endpoint)
  get '/tools' do
    content_type :json

    tools = settings.registry.all.map do |tool|
      {
        name: tool.name,
        description: tool.description,
        parameter_count: tool.parameters.length
      }
    end

    json tools: tools
  end

  run! if app_file == $0
  end
end
