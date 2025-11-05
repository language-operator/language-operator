# frozen_string_literal: true

require 'thor'
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
      end
    end
  end
end
