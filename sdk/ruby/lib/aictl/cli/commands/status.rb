# frozen_string_literal: true

require 'thor'
require_relative '../formatters/progress_formatter'
require_relative '../formatters/table_formatter'
require_relative '../../config/cluster_config'
require_relative '../../kubernetes/client'

module Aictl
  module CLI
    module Commands
      # System status and overview command
      class Status < Thor
        desc 'overview', 'Show system status and overview'
        def overview
          current_cluster = Config::ClusterConfig.current_cluster
          clusters = Config::ClusterConfig.list_clusters

          puts
          puts '═' * 80
          puts '  Language Operator CLI Status'
          puts '═' * 80
          puts

          # Current cluster context
          if current_cluster
            puts "Current Cluster: #{current_cluster}"
            cluster_config = Config::ClusterConfig.get_cluster(current_cluster)
            puts "  Namespace:   #{cluster_config[:namespace]}"
            puts "  Context:     #{cluster_config[:context] || 'default'}"
            puts

            # Check cluster health
            begin
              k8s = Kubernetes::Client.new(
                kubeconfig: cluster_config[:kubeconfig],
                context: cluster_config[:context]
              )

              # Operator status
              if k8s.operator_installed?
                version = k8s.operator_version || 'installed'
                Formatters::ProgressFormatter.success("Operator: v#{version}")
              else
                Formatters::ProgressFormatter.error('Operator: not installed')
                puts
                puts 'Install the operator with:'
                puts '  helm install language-operator oci://git.theryans.io/langop/charts/language-operator'
                puts
                return
              end

              # Agent statistics
              agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])
              agent_stats = categorize_by_status(agents)

              puts
              puts "Agents (#{agents.count} total):"
              agent_stats.each do |status, count|
                puts "  #{format_status(status)}: #{count}"
              end

              # Tool statistics
              tools = k8s.list_resources('LanguageTool', namespace: cluster_config[:namespace])
              puts
              puts "Tools (#{tools.count} total):"
              if tools.any?
                tool_types = tools.group_by { |t| t.dig('spec', 'type') }
                tool_types.each do |type, items|
                  puts "  #{type}: #{items.count}"
                end
              else
                puts '  (none)'
              end

              # Model statistics
              models = k8s.list_resources('LanguageModel', namespace: cluster_config[:namespace])
              puts
              puts "Models (#{models.count} total):"
              if models.any?
                model_providers = models.group_by { |m| m.dig('spec', 'provider') }
                model_providers.each do |provider, items|
                  puts "  #{provider}: #{items.count}"
                end
              else
                puts '  (none)'
              end

              # Persona statistics
              personas = k8s.list_resources('LanguagePersona', namespace: cluster_config[:namespace])
              puts
              puts "Personas (#{personas.count} total):"
              if personas.any?
                personas.each do |persona|
                  tone = persona.dig('spec', 'tone')
                  puts "  - #{persona.dig('metadata', 'name')} (#{tone})"
                end
              else
                puts '  (none)'
              end

            rescue StandardError => e
              Formatters::ProgressFormatter.error("Connection failed: #{e.message}")
              puts
              puts 'Check your cluster connection and try again'
            end
          else
            Formatters::ProgressFormatter.warn('No cluster selected')
            puts
            if clusters.any?
              puts 'Available clusters:'
              clusters.each do |cluster|
                puts "  - #{cluster[:name]}"
              end
              puts
              puts 'Select a cluster with:'
              puts '  aictl use <cluster>'
            else
              puts 'No clusters found. Create one with:'
              puts '  aictl cluster create <name>'
            end
          end

          # Multi-cluster summary if multiple clusters exist
          if clusters.length > 1
            puts
            puts '─' * 80
            puts
            puts "All Clusters (#{clusters.length} total):"
            puts

            cluster_summary = []
            clusters.each do |cluster|
              begin
                k8s = Kubernetes::Client.new(
                  kubeconfig: cluster[:kubeconfig],
                  context: cluster[:context]
                )

                agents = k8s.list_resources('LanguageAgent', namespace: cluster[:namespace])
                tools = k8s.list_resources('LanguageTool', namespace: cluster[:namespace])
                models = k8s.list_resources('LanguageModel', namespace: cluster[:namespace])

                cluster_summary << {
                  name: cluster[:name],
                  agents: agents.count,
                  tools: tools.count,
                  models: models.count,
                  status: k8s.operator_installed? ? 'Ready' : 'No Operator'
                }
              rescue StandardError
                cluster_summary << {
                  name: cluster[:name],
                  agents: '?',
                  tools: '?',
                  models: '?',
                  status: 'Error'
                }
              end
            end

            headers = ['CLUSTER', 'AGENTS', 'TOOLS', 'MODELS', 'STATUS']
            rows = cluster_summary.map do |c|
              name = c[:name]
              name += ' *' if c[:name] == current_cluster
              [name, c[:agents], c[:tools], c[:models], format_status(c[:status])]
            end

            require 'tty-table'
            table = TTY::Table.new(headers, rows)
            puts table.render(:unicode, padding: [0, 1])

            if current_cluster
              puts
              puts "* = current cluster"
            end
          end

          puts
          puts '═' * 80
          puts
        end

        default_task :overview

        private

        def format_status(status)
          require 'pastel'
          pastel = Pastel.new

          status_str = status.to_s
          case status_str.downcase
          when 'ready', 'running', 'active'
            "#{pastel.green('●')} #{status_str}"
          when 'pending', 'creating', 'synthesizing'
            "#{pastel.yellow('●')} #{status_str}"
          when 'failed', 'error'
            "#{pastel.red('●')} #{status_str}"
          when 'paused', 'stopped'
            "#{pastel.dim('●')} #{status_str}"
          else
            "#{pastel.dim('●')} #{status_str}"
          end
        end

        def categorize_by_status(resources)
          stats = Hash.new(0)
          resources.each do |resource|
            status = resource.dig('status', 'phase') || 'Unknown'
            stats[status] += 1
          end
          stats
        end
      end
    end
  end
end
