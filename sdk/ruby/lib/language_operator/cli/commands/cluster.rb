# frozen_string_literal: true

require 'thor'
require_relative '../formatters/progress_formatter'
require_relative '../formatters/table_formatter'
require_relative '../../config/cluster_config'
require_relative '../../kubernetes/client'
require_relative '../../kubernetes/resource_builder'

module LanguageOperator
  module CLI
    module Commands
      # Cluster management commands
      class Cluster < Thor
        desc 'create NAME', 'Create a new language cluster'
        option :namespace, type: :string, desc: 'Kubernetes namespace (defaults to current context namespace)'
        option :kubeconfig, type: :string, desc: 'Path to kubeconfig file'
        option :context, type: :string, desc: 'Kubernetes context to use'
        option :switch, type: :boolean, default: true, desc: 'Switch to new cluster context'
        def create(name)
          kubeconfig = options[:kubeconfig]
          context = options[:context]

          # Check if cluster already exists
          if Config::ClusterConfig.cluster_exists?(name)
            Formatters::ProgressFormatter.error("Cluster '#{name}' already exists")
            exit 1
          end

          # Create Kubernetes client
          k8s = Formatters::ProgressFormatter.with_spinner('Connecting to Kubernetes cluster') do
            Kubernetes::Client.new(kubeconfig: kubeconfig, context: context)
          end

          # Determine namespace: use --namespace flag, or current context namespace, or 'default'
          namespace = options[:namespace] || k8s.current_namespace || 'default'

          # Check if operator is installed
          unless k8s.operator_installed?
            Formatters::ProgressFormatter.error('Language Operator not found in cluster')
            puts "\nInstall the operator first:"
            puts '  helm install language-operator oci://git.theryans.io/langop/charts/language-operator'
            exit 1
          end

          # Create namespace if it doesn't exist
          unless k8s.namespace_exists?(namespace)
            Formatters::ProgressFormatter.with_spinner("Creating namespace '#{namespace}'") do
              k8s.create_namespace(namespace, labels: {
                                     'app.kubernetes.io/managed-by' => 'aictl',
                                     'langop.io/cluster' => name
                                   })
            end
          else
            Formatters::ProgressFormatter.info("Namespace '#{namespace}' already exists")
          end

          # Create LanguageCluster resource
          cluster_resource = Formatters::ProgressFormatter.with_spinner('Creating LanguageCluster resource') do
            resource = Kubernetes::ResourceBuilder.language_cluster(name, namespace: namespace)
            k8s.apply_resource(resource)
            resource
          end

          # Save cluster to config
          Formatters::ProgressFormatter.with_spinner('Saving cluster configuration') do
            Config::ClusterConfig.add_cluster(
              name,
              namespace,
              kubeconfig || ENV.fetch('KUBECONFIG', File.expand_path('~/.kube/config')),
              context
            )
          end

          # Switch to new cluster if requested
          if options[:switch]
            Config::ClusterConfig.set_current_cluster(name)
            Formatters::ProgressFormatter.success("Created and switched to cluster '#{name}'")
          else
            Formatters::ProgressFormatter.success("Created cluster '#{name}'")
            puts "\nSwitch to this cluster with:"
            puts "  aictl use #{name}"
          end

          puts "\nCluster Details:"
          puts "  Name:      #{name}"
          puts "  Namespace: #{namespace}"
          puts "  Context:   #{context || 'default'}"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to create cluster: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'list', 'List all clusters'
        option :all, type: :boolean, default: false, desc: 'Show all clusters including inactive'
        def list
          clusters = Config::ClusterConfig.list_clusters
          current = Config::ClusterConfig.current_cluster

          if clusters.empty?
            Formatters::ProgressFormatter.info('No clusters found')
            puts "\nCreate a cluster with:"
            puts '  aictl cluster create <name>'
            return
          end

          # Build table data
          table_data = clusters.map do |cluster|
            k8s = Kubernetes::Client.new(
              kubeconfig: cluster[:kubeconfig],
              context: cluster[:context]
            )

            # Get cluster stats
            agents = k8s.list_resources('LanguageAgent', namespace: cluster[:namespace])
            tools = k8s.list_resources('LanguageTool', namespace: cluster[:namespace])
            models = k8s.list_resources('LanguageModel', namespace: cluster[:namespace])

            # Get cluster status
            cluster_resource = k8s.get_resource('LanguageCluster', cluster[:name], cluster[:namespace])
            status = cluster_resource.dig('status', 'phase') || 'Unknown'

            name_display = cluster[:name]
            name_display += ' *' if cluster[:name] == current

            {
              name: name_display,
              namespace: cluster[:namespace],
              agents: agents.count,
              tools: tools.count,
              models: models.count,
              status: status
            }
          rescue StandardError => e
            {
              name: cluster[:name],
              namespace: cluster[:namespace],
              agents: '?',
              tools: '?',
              models: '?',
              status: 'Error'
            }
          end

          Formatters::TableFormatter.clusters(table_data)

          if current
            puts "\nCurrent cluster: #{current} (*)"
          else
            puts "\nNo cluster selected. Use 'aictl use <cluster>' to select one."
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to list clusters: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'current', 'Show current cluster context'
        def current
          cluster_name = Config::ClusterConfig.current_cluster

          unless cluster_name
            Formatters::ProgressFormatter.info('No cluster selected')
            puts "\nSelect a cluster with:"
            puts '  aictl use <cluster>'
            return
          end

          cluster = Config::ClusterConfig.get_cluster(cluster_name)

          unless cluster
            Formatters::ProgressFormatter.error("Cluster '#{cluster_name}' not found in config")
            exit 1
          end

          puts "Current Cluster:"
          puts "  Name:      #{cluster[:name]}"
          puts "  Namespace: #{cluster[:namespace]}"
          puts "  Context:   #{cluster[:context] || 'default'}"
          puts "  Created:   #{cluster[:created]}"

          # Check cluster health
          begin
            k8s = Kubernetes::Client.new(
              kubeconfig: cluster[:kubeconfig],
              context: cluster[:context]
            )

            if k8s.operator_installed?
              version = k8s.operator_version
              Formatters::ProgressFormatter.success("Operator: #{version || 'installed'}")
            else
              Formatters::ProgressFormatter.warn('Operator: not found')
            end
          rescue StandardError => e
            Formatters::ProgressFormatter.error("Connection: #{e.message}")
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to show current cluster: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'delete NAME', 'Delete a cluster'
        option :force, type: :boolean, default: false, desc: 'Skip confirmation'
        def delete(name)
          unless Config::ClusterConfig.cluster_exists?(name)
            Formatters::ProgressFormatter.error("Cluster '#{name}' not found")
            exit 1
          end

          cluster = Config::ClusterConfig.get_cluster(name)

          # Confirm deletion
          unless options[:force]
            puts "This will delete cluster '#{name}' and all its resources:"
            puts "  Namespace: #{cluster[:namespace]}"
            print "\nAre you sure? (y/N): "
            confirmation = $stdin.gets.chomp
            unless confirmation.downcase == 'y'
              puts 'Deletion cancelled'
              return
            end
          end

          # Delete LanguageCluster resource
          begin
            k8s = Kubernetes::Client.new(
              kubeconfig: cluster[:kubeconfig],
              context: cluster[:context]
            )

            Formatters::ProgressFormatter.with_spinner('Deleting LanguageCluster resource') do
              k8s.delete_resource('LanguageCluster', name, cluster[:namespace])
            end
          rescue StandardError => e
            Formatters::ProgressFormatter.warn("Failed to delete cluster resource: #{e.message}")
          end

          # Remove from config
          Formatters::ProgressFormatter.with_spinner('Removing cluster from configuration') do
            Config::ClusterConfig.remove_cluster(name)
          end

          # Clear current cluster if this was it
          if Config::ClusterConfig.current_cluster == name
            Config::ClusterConfig.set_current_cluster(nil)
          end

          Formatters::ProgressFormatter.success("Deleted cluster '#{name}'")
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to delete cluster: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'inspect NAME', 'Show detailed cluster information'
        def inspect(name)
          unless Config::ClusterConfig.cluster_exists?(name)
            Formatters::ProgressFormatter.error("Cluster '#{name}' not found")
            exit 1
          end

          cluster = Config::ClusterConfig.get_cluster(name)

          puts "Cluster: #{name}"
          puts "  Namespace: #{cluster[:namespace]}"
          puts "  Context:   #{cluster[:context] || 'default'}"
          puts "  Created:   #{cluster[:created]}"
          puts

          # Get detailed cluster info
          begin
            k8s = Kubernetes::Client.new(
              kubeconfig: cluster[:kubeconfig],
              context: cluster[:context]
            )

            # Get cluster resource
            cluster_resource = k8s.get_resource('LanguageCluster', name, cluster[:namespace])
            status = cluster_resource.dig('status', 'phase') || 'Unknown'

            puts "Status: #{status}"
            puts

            # Get agents
            agents = k8s.list_resources('LanguageAgent', namespace: cluster[:namespace])
            puts "Agents: #{agents.count}"
            agents.each do |agent|
              agent_status = agent.dig('status', 'phase') || 'Unknown'
              puts "  - #{agent.dig('metadata', 'name')} (#{agent_status})"
            end
            puts

            # Get tools
            tools = k8s.list_resources('LanguageTool', namespace: cluster[:namespace])
            puts "Tools: #{tools.count}"
            tools.each do |tool|
              tool_type = tool.dig('spec', 'type')
              puts "  - #{tool.dig('metadata', 'name')} (#{tool_type})"
            end
            puts

            # Get models
            models = k8s.list_resources('LanguageModel', namespace: cluster[:namespace])
            puts "Models: #{models.count}"
            models.each do |model|
              provider = model.dig('spec', 'provider')
              model_name = model.dig('spec', 'model')
              puts "  - #{model.dig('metadata', 'name')} (#{provider}/#{model_name})"
            end
            puts

            # Get personas
            personas = k8s.list_resources('LanguagePersona', namespace: cluster[:namespace])
            puts "Personas: #{personas.count}"
            personas.each do |persona|
              tone = persona.dig('spec', 'tone')
              puts "  - #{persona.dig('metadata', 'name')} (#{tone})"
            end
          rescue StandardError => e
            Formatters::ProgressFormatter.error("Failed to get cluster details: #{e.message}")
            raise if ENV['DEBUG']
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to inspect cluster: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end
      end
    end
  end
end
