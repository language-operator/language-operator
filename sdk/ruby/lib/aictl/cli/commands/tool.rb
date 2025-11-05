# frozen_string_literal: true

require 'thor'
require 'yaml'
require 'erb'
require 'net/http'
require_relative '../formatters/progress_formatter'
require_relative '../formatters/table_formatter'
require_relative '../helpers/cluster_validator'
require_relative '../../config/cluster_config'
require_relative '../../kubernetes/client'

module Aictl
  module CLI
    module Commands
      # Tool management commands
      class Tool < Thor
        include Helpers::ClusterValidator

        desc 'list', 'List all tools in current cluster'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def list
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          tools = k8s.list_resources('LanguageTool', namespace: cluster_config[:namespace])

          if tools.empty?
            Formatters::ProgressFormatter.info("No tools found in cluster '#{cluster}'")
            puts
            puts 'Tools provide MCP server capabilities for agents.'
            puts
            puts 'Install a tool with:'
            puts '  aictl tool install <name>'
            return
          end

          # Get agents to count usage
          agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])

          table_data = tools.map do |tool|
            name = tool.dig('metadata', 'name')
            type = tool.dig('spec', 'type') || 'unknown'
            status = tool.dig('status', 'phase') || 'Unknown'

            # Count agents using this tool
            agents_using = agents.count do |agent|
              agent_tools = agent.dig('spec', 'tools') || []
              agent_tools.include?(name)
            end

            # Get health status
            health = tool.dig('status', 'health') || 'unknown'
            health_indicator = case health.downcase
                               when 'healthy' then '✓'
                               when 'unhealthy' then '✗'
                               else '?'
                               end

            {
              name: name,
              type: type,
              status: status,
              agents_using: agents_using,
              health: "#{health_indicator} #{health}"
            }
          end

          Formatters::TableFormatter.tools(table_data)
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to list tools: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'delete NAME', 'Delete a tool'
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :force, type: :boolean, default: false, desc: 'Skip confirmation'
        def delete(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get tool
          begin
            tool = k8s.get_resource('LanguageTool', name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Tool '#{name}' not found in cluster '#{cluster}'")
            exit 1
          end

          # Check for agents using this tool
          agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])
          agents_using = agents.select do |agent|
            agent_tools = agent.dig('spec', 'tools') || []
            agent_tools.include?(name)
          end

          if agents_using.any? && !options[:force]
            Formatters::ProgressFormatter.warn("Tool '#{name}' is in use by #{agents_using.count} agent(s)")
            puts
            puts 'Agents using this tool:'
            agents_using.each do |agent|
              puts "  - #{agent.dig('metadata', 'name')}"
            end
            puts
            puts 'Delete these agents first, or use --force to delete anyway.'
            puts
            print 'Are you sure? (y/N): '
            confirmation = $stdin.gets.chomp
            unless confirmation.downcase == 'y'
              puts 'Deletion cancelled'
              return
            end
          end

          # Confirm deletion unless --force
          unless options[:force] || agents_using.any?
            puts "This will delete tool '#{name}' from cluster '#{cluster}':"
            puts "  Type:   #{tool.dig('spec', 'type')}"
            puts "  Status: #{tool.dig('status', 'phase')}"
            puts
            print 'Are you sure? (y/N): '
            confirmation = $stdin.gets.chomp
            unless confirmation.downcase == 'y'
              puts 'Deletion cancelled'
              return
            end
          end

          # Delete tool
          Formatters::ProgressFormatter.with_spinner("Deleting tool '#{name}'") do
            k8s.delete_resource('LanguageTool', name, cluster_config[:namespace])
          end

          Formatters::ProgressFormatter.success("Tool '#{name}' deleted successfully")
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to delete tool: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'install NAME', 'Install a tool from the registry'
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :deployment_mode, type: :string, enum: %w[service sidecar], desc: 'Deployment mode (service or sidecar)'
        option :replicas, type: :numeric, desc: 'Number of replicas'
        option :dry_run, type: :boolean, default: false, desc: 'Preview without installing'
        def install(tool_name)
          # For dry-run mode, allow operation without a real cluster
          if options[:dry_run]
            cluster = options[:cluster] || 'preview'
            cluster_config = { namespace: 'default' }
          else
            cluster = Helpers::ClusterValidator.get_cluster(options[:cluster])
            cluster_config = Helpers::ClusterValidator.get_cluster_config(cluster)
          end

          # Load tool patterns registry
          patterns_path = File.join(__dir__, '..', '..', 'config', 'tool_patterns.yaml')
          unless File.exist?(patterns_path)
            Formatters::ProgressFormatter.error('Tool registry not found')
            exit 1
          end

          patterns = YAML.load_file(patterns_path)

          # Resolve aliases
          tool_key = tool_name
          while patterns[tool_key]&.key?('alias')
            tool_key = patterns[tool_key]['alias']
          end

          # Look up tool in registry
          tool_config = patterns[tool_key]
          unless tool_config
            Formatters::ProgressFormatter.error("Tool '#{tool_name}' not found in registry")
            puts
            puts 'Available tools:'
            patterns.each do |key, config|
              next if config['alias']

              puts "  #{key.ljust(15)} - #{config['description']}"
            end
            exit 1
          end

          # Check if template exists
          template_name = tool_key
          template_path = File.join(__dir__, '..', 'templates', 'tools', "#{template_name}.yaml")
          unless File.exist?(template_path)
            Formatters::ProgressFormatter.warn("No template found for '#{tool_name}', using generic template")
            template_path = File.join(__dir__, '..', 'templates', 'tools', 'generic.yaml')
          end

          # Build template variables
          vars = {
            name: tool_name,
            namespace: cluster_config[:namespace],
            deployment_mode: options[:deployment_mode] || tool_config['deploymentMode'],
            replicas: options[:replicas] || 1,
            auth_secret: nil # Will be set by auth command
          }

          # Render template
          template = ERB.new(File.read(template_path))
          yaml_content = template.result_with_hash(vars)

          # Dry run mode
          if options[:dry_run]
            puts "Would install tool '#{tool_name}' to cluster '#{cluster}':"
            puts
            puts "Display Name:    #{tool_config['displayName']}"
            puts "Description:     #{tool_config['description']}"
            puts "Deployment Mode: #{vars[:deployment_mode]}"
            puts "Replicas:        #{vars[:replicas]}"
            puts "Auth Required:   #{tool_config['authRequired'] ? 'Yes' : 'No'}"
            puts
            puts 'Generated YAML:'
            puts '---'
            puts yaml_content
            puts
            puts 'To install for real, run without --dry-run'
            return
          end

          # Connect to cluster
          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Check if already exists
          begin
            existing = k8s.get_resource('LanguageTool', tool_name, cluster_config[:namespace])
            Formatters::ProgressFormatter.warn("Tool '#{tool_name}' already exists in cluster '#{cluster}'")
            puts
            print 'Do you want to update it? (y/N): '
            confirmation = $stdin.gets.chomp
            unless confirmation.downcase == 'y'
              puts 'Installation cancelled'
              return
            end
          rescue K8s::Error::NotFound
            # Tool doesn't exist, proceed with creation
          end

          # Install tool
          Formatters::ProgressFormatter.with_spinner("Installing tool '#{tool_name}'") do
            resource = YAML.safe_load(yaml_content, permitted_classes: [Symbol])
            k8s.apply_resource(resource)
          end

          Formatters::ProgressFormatter.success("Tool '#{tool_name}' installed successfully")
          puts
          puts "Tool '#{tool_name}' is now available in cluster '#{cluster}'"
          if tool_config['authRequired']
            puts
            puts "This tool requires authentication. Configure it with:"
            puts "  aictl tool auth #{tool_name}"
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to install tool: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'auth NAME', 'Configure authentication for a tool'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def auth(tool_name)
          cluster = Helpers::ClusterValidator.get_cluster(options[:cluster])
          cluster_config = Helpers::ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Check if tool exists
          begin
            tool = k8s.get_resource('LanguageTool', tool_name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Tool '#{tool_name}' not found in cluster '#{cluster}'")
            puts
            puts 'Install the tool first with:'
            puts "  aictl tool install #{tool_name}"
            exit 1
          end

          puts "Configure authentication for tool '#{tool_name}'"
          puts

          # Determine auth type based on tool
          case tool_name
          when 'email', 'gmail'
            puts 'Email/Gmail Configuration'
            puts '-' * 40
            print 'SMTP Server: '
            smtp_server = $stdin.gets.chomp
            print 'SMTP Port (587): '
            smtp_port = $stdin.gets.chomp
            smtp_port = '587' if smtp_port.empty?
            print 'Email Address: '
            email = $stdin.gets.chomp
            print 'Password: '
            password = $stdin.noecho(&:gets).chomp
            puts

            secret_data = {
              'SMTP_SERVER' => smtp_server,
              'SMTP_PORT' => smtp_port,
              'EMAIL_ADDRESS' => email,
              'EMAIL_PASSWORD' => password
            }

          when 'github'
            puts 'GitHub Configuration'
            puts '-' * 40
            print 'GitHub Token: '
            token = $stdin.noecho(&:gets).chomp
            puts

            secret_data = {
              'GITHUB_TOKEN' => token
            }

          when 'slack'
            puts 'Slack Configuration'
            puts '-' * 40
            print 'Slack Bot Token: '
            token = $stdin.noecho(&:gets).chomp
            puts

            secret_data = {
              'SLACK_BOT_TOKEN' => token
            }

          when 'gdrive'
            puts 'Google Drive Configuration'
            puts '-' * 40
            puts 'Note: You need OAuth credentials from Google Cloud Console'
            print 'Client ID: '
            client_id = $stdin.gets.chomp
            print 'Client Secret: '
            client_secret = $stdin.noecho(&:gets).chomp
            puts

            secret_data = {
              'GDRIVE_CLIENT_ID' => client_id,
              'GDRIVE_CLIENT_SECRET' => client_secret
            }

          else
            puts 'Generic API Key Configuration'
            puts '-' * 40
            print 'API Key: '
            api_key = $stdin.noecho(&:gets).chomp
            puts

            secret_data = {
              'API_KEY' => api_key
            }
          end

          # Create secret
          secret_name = "#{tool_name}-auth"
          secret_resource = {
            'apiVersion' => 'v1',
            'kind' => 'Secret',
            'metadata' => {
              'name' => secret_name,
              'namespace' => cluster_config[:namespace]
            },
            'type' => 'Opaque',
            'stringData' => secret_data
          }

          Formatters::ProgressFormatter.with_spinner('Creating authentication secret') do
            k8s.apply_resource(secret_resource)
          end

          # Update tool to use secret
          tool['spec']['envFrom'] ||= []
          tool['spec']['envFrom'] << { 'secretRef' => { 'name' => secret_name } }

          Formatters::ProgressFormatter.with_spinner('Updating tool configuration') do
            k8s.apply_resource(tool)
          end

          Formatters::ProgressFormatter.success('Authentication configured successfully')
          puts
          puts "Tool '#{tool_name}' is now authenticated and ready to use"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to configure auth: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'test NAME', 'Test tool connectivity and health'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def test(tool_name)
          cluster = Helpers::ClusterValidator.get_cluster(options[:cluster])
          cluster_config = Helpers::ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get tool
          begin
            tool = k8s.get_resource('LanguageTool', tool_name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Tool '#{tool_name}' not found in cluster '#{cluster}'")
            exit 1
          end

          puts "Testing tool '#{tool_name}' in cluster '#{cluster}'"
          puts

          # Check phase
          phase = tool.dig('status', 'phase') || 'Unknown'
          status_indicator = case phase
                             when 'Running' then '✓'
                             when 'Pending' then '⏳'
                             when 'Failed' then '✗'
                             else '?'
                             end

          puts "Status:   #{status_indicator} #{phase}"

          # Check replicas
          ready_replicas = tool.dig('status', 'readyReplicas') || 0
          desired_replicas = tool.dig('spec', 'replicas') || 1
          puts "Replicas: #{ready_replicas}/#{desired_replicas} ready"

          # Check endpoint
          endpoint = tool.dig('status', 'endpoint')
          if endpoint
            puts "Endpoint: #{endpoint}"
          else
            puts 'Endpoint: Not available yet'
          end

          # Get pod status
          puts
          puts 'Pod Status:'

          label_selector = "langop.io/tool=#{tool_name}"
          pods = k8s.list_resources('Pod', namespace: cluster_config[:namespace], label_selector: label_selector)

          if pods.empty?
            puts '  No pods found'
          else
            pods.each do |pod|
              pod_name = pod.dig('metadata', 'name')
              pod_phase = pod.dig('status', 'phase') || 'Unknown'
              pod_indicator = case pod_phase
                              when 'Running' then '✓'
                              when 'Pending' then '⏳'
                              when 'Failed' then '✗'
                              else '?'
                              end

              puts "  #{pod_indicator} #{pod_name}: #{pod_phase}"

              # Check container status
              container_statuses = pod.dig('status', 'containerStatuses') || []
              container_statuses.each do |status|
                ready = status['ready'] ? '✓' : '✗'
                puts "    #{ready} #{status['name']}: #{status['state']&.keys&.first || 'unknown'}"
              end
            end
          end

          # Test connectivity if endpoint is available
          if endpoint && phase == 'Running'
            puts
            puts 'Testing connectivity...'
            begin
              uri = URI(endpoint)
              response = Net::HTTP.get_response(uri)
              if response.code.to_i < 400
                Formatters::ProgressFormatter.success('Connectivity test passed')
              else
                Formatters::ProgressFormatter.warn("HTTP #{response.code}: #{response.message}")
              end
            rescue StandardError => e
              Formatters::ProgressFormatter.error("Connectivity test failed: #{e.message}")
            end
          end

          # Overall health
          puts
          if phase == 'Running' && ready_replicas == desired_replicas
            Formatters::ProgressFormatter.success("Tool '#{tool_name}' is healthy and operational")
          elsif phase == 'Pending'
            Formatters::ProgressFormatter.info("Tool '#{tool_name}' is starting up, please wait")
          else
            Formatters::ProgressFormatter.warn("Tool '#{tool_name}' has issues, check logs for details")
            puts
            puts 'View logs with:'
            puts "  kubectl logs -n #{cluster_config[:namespace]} -l langop.io/tool=#{tool_name}"
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to test tool: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end
      end
    end
  end
end
