# frozen_string_literal: true

require 'rack'
require 'json'
require 'securerandom'
require 'time'
require 'erb'

module LanguageOperator
  module Dashboard
    # Dashboard web server for LanguageCluster management
    # rubocop:disable Metrics/ClassLength
    class Server
      attr_reader :cluster_name, :namespace, :port

      def initialize(cluster_name:, namespace: 'default', port: 8080)
        @cluster_name = cluster_name
        @namespace = namespace
        @port = port
        @k8s_client = nil # Will be initialized when needed
      end

      # Start the Rack server
      def start
        puts 'Starting LanguageCluster Dashboard'
        puts "Cluster: #{cluster_name}"
        puts "Namespace: #{namespace}"
        puts "Port: #{port}"
        puts "Access at: http://localhost:#{port}"
        puts

        Rack::Handler::WEBrick.run(
          method(:call),
          Port: port,
          Host: '0.0.0.0',
          Logger: WEBrick::Log.new(File::NULL),
          AccessLog: []
        )
      end

      # Rack application interface
      # rubocop:disable Metrics/CyclomaticComplexity
      def call(env)
        request = Rack::Request.new(env)
        method = request.request_method
        path = request.path_info

        # Route the request
        case [method, path]
        when ['GET', '/']
          serve_static_file('public/index.html', 'text/html')
        when ['GET', '/health']
          json_response({ status: 'healthy', cluster: cluster_name })
        when ['GET', '/api/cluster/info']
          handle_cluster_info
        when ['GET', '/api/cluster/metrics']
          handle_cluster_metrics
        when ['GET', '/api/agents']
          handle_list_agents
        when ['POST', '/api/agents']
          handle_create_agent(request)
        when ['GET', '/components/agents']
          handle_agents_component
        when ['GET', '/components/metrics']
          handle_metrics_component
        when ['GET', '/components/synthesis']
          handle_synthesis_component
        else
          # Check for dynamic routes
          if path =~ %r{^/components/agent/(.+)$}
            handle_agent_detail_component(Regexp.last_match(1))
          elsif path =~ %r{^/api/agents/(.+)/logs$}
            handle_agent_logs(Regexp.last_match(1))
          elsif path =~ %r{^/api/agents/(.+)/run$} && method == 'POST'
            handle_run_agent(Regexp.last_match(1))
          elsif path =~ %r{^/api/agents/(.+)$}
            handle_get_agent(Regexp.last_match(1))
          else
            not_found_response
          end
        end
      rescue StandardError => e
        error_response(e)
      end
      # rubocop:enable Metrics/CyclomaticComplexity

      private

      # Serve static files
      def serve_static_file(path, content_type)
        file_path = File.join(__dir__, '..', path)
        if File.exist?(file_path)
          content = File.read(file_path)
          [200, { 'Content-Type' => content_type }, [content]]
        else
          not_found_response
        end
      end

      # Get cluster info
      def handle_cluster_info
        html = cluster_name
        [200, { 'Content-Type' => 'text/html' }, [html]]
      end

      # Get cluster metrics (HTML fragment for navbar)
      def handle_cluster_metrics
        agents = mock_list_agents

        running = agents.count { |a| a[:phase] == 'Running' }
        total_cost = agents.sum { |a| a.dig(:synthesis_info, :estimated_cost).to_f }

        html = <<~HTML
          <div class="flex items-center space-x-6">
              <div class="flex items-center space-x-2">
                  <span class="text-gray-600">Agents:</span>
                  <span class="font-semibold text-gray-900">#{running}/#{agents.length}</span>
                  <span class="w-2 h-2 bg-green-500 rounded-full status-pulse"></span>
              </div>
              <div class="flex items-center space-x-2">
                  <span class="text-gray-600">Total Cost:</span>
                  <span class="font-semibold text-gray-900">$#{format('%.2f', total_cost)}</span>
              </div>
          </div>
        HTML

        [200, { 'Content-Type' => 'text/html' }, [html]]
      end

      # List all agents (JSON)
      def handle_list_agents
        agents = mock_list_agents
        json_response(agents)
      end

      # Create a new agent
      def handle_create_agent(request)
        params = request.params

        agent_name = generate_agent_name(params['instructions'])

        # Mock agent creation (in real implementation, create K8s resource)
        {
          name: agent_name,
          instructions: params['instructions'],
          mode: params['mode'] || 'autonomous',
          phase: 'Pending',
          created_at: Time.now.iso8601
        }

        puts "Created agent: #{agent_name}"

        # Return updated agent list HTML fragment
        handle_agents_component
      end

      # Get agent list HTML component
      # rubocop:disable Metrics/MethodLength
      def handle_agents_component
        agents = mock_list_agents

        html = ERB.new(<<~ERB, trim_mode: '-').result(binding)
          <% if agents.empty? %>
            <div class="p-8 text-center text-gray-500">
              <svg class="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
              </svg>
              <p class="text-lg font-medium">No agents yet</p>
              <p class="text-sm mt-2">Create your first agent to get started</p>
            </div>
          <% else %>
            <table class="w-full">
              <thead class="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Agent</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Last Run</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Success Rate</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Cost</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-gray-200">
                <% agents.each do |agent| %>
                  <tr class="hover:bg-gray-50 transition-colors">
                    <td class="px-6 py-4 whitespace-nowrap">
                      <div class="font-medium text-gray-900"><%= agent[:name] %></div>
                      <div class="text-xs text-gray-500"><%= agent[:mode] %></div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                      <span class="px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full <%= status_class(agent[:phase]) %>">
                        <%= agent[:phase] %>
                      </span>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      <%= format_time(agent[:last_run]) %>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                      <div class="flex items-center">
                        <div class="text-sm font-medium text-gray-900"><%= agent[:success_rate] %>%</div>
                        <div class="ml-2 w-16 bg-gray-200 rounded-full h-2">
                          <div class="bg-green-500 h-2 rounded-full" style="width: <%= agent[:success_rate] %>%"></div>
                        </div>
                      </div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      $<%= format('%.2f', agent.dig(:synthesis_info, :estimated_cost) || 0) %>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button @click="agentDetail = '<%= agent[:name] %>'"
                              class="text-blue-600 hover:text-blue-900 mr-4">View</button>
                      <button hx-post="/api/agents/<%= agent[:name] %>/run"
                              hx-swap="none"
                              class="text-green-600 hover:text-green-900">Run</button>
                    </td>
                  </tr>
                <% end %>
              </tbody>
            </table>
          <% end %>
        ERB

        [200, { 'Content-Type' => 'text/html' }, [html]]
      end
      # rubocop:enable Metrics/MethodLength

      # Get metrics HTML component
      # rubocop:disable Metrics/MethodLength
      def handle_metrics_component
        agents = mock_list_agents

        total_cost = agents.sum { |a| a.dig(:synthesis_info, :estimated_cost).to_f }
        total_executions = agents.sum { |a| a[:execution_count] || 0 }
        successful_executions = agents.sum { |a| a[:successful_executions] || 0 }

        html = <<~HTML
          <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
            <div class="bg-white rounded-lg shadow p-6">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-600">Total Agents</p>
                  <p class="text-3xl font-bold text-gray-900">#{agents.length}</p>
                </div>
                <svg class="w-12 h-12 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"></path>
                </svg>
              </div>
            </div>

            <div class="bg-white rounded-lg shadow p-6">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-600">Total Cost</p>
                  <p class="text-3xl font-bold text-gray-900">$#{format('%.2f', total_cost)}</p>
                </div>
                <svg class="w-12 h-12 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
              </div>
            </div>

            <div class="bg-white rounded-lg shadow p-6">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-600">Success Rate</p>
                  <p class="text-3xl font-bold text-gray-900">#{total_executions.zero? ? 0 : ((successful_executions.to_f / total_executions) * 100).round(1)}%</p>
                </div>
                <svg class="w-12 h-12 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
              </div>
            </div>
          </div>

          <div class="bg-white rounded-lg shadow p-6">
            <h3 class="text-lg font-bold text-gray-900 mb-4">Agent Status Distribution</h3>
            <div class="space-y-3">
              <div>
                <div class="flex justify-between text-sm mb-1">
                  <span class="text-gray-600">Running</span>
                  <span class="font-medium">#{agents.count { |a| a[:phase] == 'Running' }}</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div class="bg-green-500 h-2 rounded-full" style="width: #{(agents.count { |a| a[:phase] == 'Running' }.to_f / [agents.length, 1].max * 100).round}%"></div>
                </div>
              </div>
              <div>
                <div class="flex justify-between text-sm mb-1">
                  <span class="text-gray-600">Pending</span>
                  <span class="font-medium">#{agents.count { |a| a[:phase] == 'Pending' }}</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div class="bg-yellow-500 h-2 rounded-full" style="width: #{(agents.count { |a| a[:phase] == 'Pending' }.to_f / [agents.length, 1].max * 100).round}%"></div>
                </div>
              </div>
              <div>
                <div class="flex justify-between text-sm mb-1">
                  <span class="text-gray-600">Failed</span>
                  <span class="font-medium">#{agents.count { |a| a[:phase] == 'Failed' }}</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2">
                  <div class="bg-red-500 h-2 rounded-full" style="width: #{(agents.count { |a| a[:phase] == 'Failed' }.to_f / [agents.length, 1].max * 100).round}%"></div>
                </div>
              </div>
            </div>
          </div>
        HTML

        [200, { 'Content-Type' => 'text/html' }, [html]]
      end
      # rubocop:enable Metrics/MethodLength

      # Get synthesis status HTML component
      def handle_synthesis_component
        html = <<~HTML
          <div class="bg-white rounded-lg shadow p-6">
            <p class="text-gray-500">Synthesis monitoring will be available in the next release.</p>
            <p class="text-sm text-gray-400 mt-2">This panel will show real-time synthesis progress for agents being created.</p>
          </div>
        HTML

        [200, { 'Content-Type' => 'text/html' }, [html]]
      end

      # Get agent detail HTML component
      def handle_agent_detail_component(agent_name)
        agent = mock_list_agents.find { |a| a[:name] == agent_name }

        return not_found_response unless agent

        html = <<~HTML
          <div class="space-y-6">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <p class="text-sm font-medium text-gray-600">Status</p>
                <p class="text-lg font-semibold text-gray-900">#{agent[:phase]}</p>
              </div>
              <div>
                <p class="text-sm font-medium text-gray-600">Mode</p>
                <p class="text-lg font-semibold text-gray-900">#{agent[:mode]}</p>
              </div>
              <div>
                <p class="text-sm font-medium text-gray-600">Success Rate</p>
                <p class="text-lg font-semibold text-gray-900">#{agent[:success_rate]}%</p>
              </div>
              <div>
                <p class="text-sm font-medium text-gray-600">Total Cost</p>
                <p class="text-lg font-semibold text-gray-900">$#{format('%.2f', agent.dig(:synthesis_info, :estimated_cost) || 0)}</p>
              </div>
            </div>

            <div>
              <h4 class="text-sm font-medium text-gray-600 mb-2">Instructions</h4>
              <p class="text-gray-900 bg-gray-50 p-4 rounded-lg">#{agent[:instructions] || 'No instructions provided'}</p>
            </div>

            <div>
              <h4 class="text-sm font-medium text-gray-600 mb-2">Recent Logs</h4>
              <div class="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm">
                <p>[#{Time.now.strftime('%Y-%m-%d %H:%M:%S')}] Agent initialized</p>
                <p>[#{Time.now.strftime('%Y-%m-%d %H:%M:%S')}] Waiting for execution trigger...</p>
              </div>
            </div>
          </div>
        HTML

        [200, { 'Content-Type' => 'text/html' }, [html]]
      end

      # Get agent details (JSON)
      def handle_get_agent(agent_name)
        agent = mock_list_agents.find { |a| a[:name] == agent_name }
        agent ? json_response(agent) : not_found_response
      end

      # Get agent logs
      def handle_agent_logs(agent_name)
        logs = "Sample logs for #{agent_name}\n" * 10
        json_response({ logs: logs })
      end

      # Run an agent
      def handle_run_agent(agent_name)
        puts "Triggering execution for agent: #{agent_name}"
        json_response({ status: 'triggered', agent: agent_name })
      end

      # Helper: Generate agent name from instructions
      def generate_agent_name(instructions)
        words = instructions.downcase.split(/\W+/).reject { |w| w.length < 3 }
        "agent-#{words.first(3).join('-')}-#{SecureRandom.hex(3)}"
      end

      # Helper: Format time for display
      def format_time(time)
        return 'Never' unless time

        Time.parse(time).strftime('%Y-%m-%d %H:%M')
      rescue StandardError
        'Unknown'
      end

      # Helper: Get CSS class for status
      def status_class(phase)
        case phase
        when 'Running'
          'bg-green-100 text-green-800'
        when 'Failed'
          'bg-red-100 text-red-800'
        when 'Pending'
          'bg-yellow-100 text-yellow-800'
        else
          'bg-gray-100 text-gray-800'
        end
      end

      # Mock: List agents (replace with K8s API call)
      def mock_list_agents
        [
          {
            name: 'spreadsheet-review',
            instructions: 'Review my spreadsheet at 4pm daily and email me any errors',
            mode: 'autonomous',
            phase: 'Running',
            last_run: (Time.now - 120).iso8601,
            execution_count: 45,
            successful_executions: 44,
            success_rate: 98,
            synthesis_info: { estimated_cost: 1.23 }
          },
          {
            name: 'email-summarizer',
            instructions: 'Summarize my daily emails and send me a digest',
            mode: 'autonomous',
            phase: 'Running',
            last_run: (Time.now - 3600).iso8601,
            execution_count: 30,
            successful_executions: 30,
            success_rate: 100,
            synthesis_info: { estimated_cost: 0.89 }
          },
          {
            name: 'code-reviewer',
            instructions: 'Review pull requests and comment on code quality issues',
            mode: 'event-driven',
            phase: 'Failed',
            last_run: (Time.now - 300).iso8601,
            execution_count: 12,
            successful_executions: 0,
            success_rate: 0,
            synthesis_info: { estimated_cost: 0.45 }
          }
        ]
      end

      # Response helpers
      def json_response(data, status: 200)
        [status, { 'Content-Type' => 'application/json' }, [JSON.generate(data)]]
      end

      def not_found_response
        [404, { 'Content-Type' => 'text/plain' }, ['Not Found']]
      end

      def error_response(error)
        puts "Error: #{error.message}"
        puts error.backtrace.join("\n")
        json_response({ error: error.message }, status: 500)
      end
    end
    # rubocop:enable Metrics/ClassLength
  end
end
