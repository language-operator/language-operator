# frozen_string_literal: true

require 'thor'
require_relative '../formatters/progress_formatter'
require_relative '../formatters/table_formatter'
require_relative '../helpers/cluster_validator'
require_relative '../../config/cluster_config'
require_relative '../../kubernetes/client'
require_relative '../../kubernetes/resource_builder'

module Aictl
  module CLI
    module Commands
      # Agent management commands
      class Agent < Thor
        include Helpers::ClusterValidator

        desc 'create DESCRIPTION', 'Create a new agent with natural language description'
        long_desc <<-DESC
          Create a new autonomous agent by describing what you want it to do in natural language.

          The operator will synthesize the agent from your description and deploy it to your cluster.

          Examples:
            aictl agent create "review my spreadsheet at 4pm daily and email me any errors"
            aictl agent create "summarize Hacker News top stories every morning at 8am"
            aictl agent create "monitor my website uptime and alert me if it goes down"
        DESC
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :create_cluster, type: :string, desc: 'Create cluster if it doesn\'t exist'
        option :name, type: :string, desc: 'Agent name (generated from description if not provided)'
        option :persona, type: :string, desc: 'Persona to use for the agent'
        option :tools, type: :array, desc: 'Tools to make available to the agent'
        option :models, type: :array, desc: 'Models to make available to the agent'
        def create(description)
          # Handle --create-cluster flag
          if options[:create_cluster]
            cluster_name = options[:create_cluster]
            unless Config::ClusterConfig.cluster_exists?(cluster_name)
              Formatters::ProgressFormatter.info("Creating cluster '#{cluster_name}'...")
              # Delegate to cluster create command
              require_relative 'cluster'
              Cluster.new.invoke(:create, [cluster_name], switch: true)
            end
            cluster = cluster_name
          else
            # Validate cluster selection (this will exit if none selected)
            cluster = ClusterValidator.get_cluster(options[:cluster])
          end

          cluster_config = ClusterValidator.get_cluster_config(cluster)

          Formatters::ProgressFormatter.info("Creating agent in cluster '#{cluster}'")
          puts

          # Generate agent name from description if not provided
          agent_name = options[:name] || generate_agent_name(description)

          # Connect to Kubernetes
          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Build LanguageAgent resource
          agent_resource = Formatters::ProgressFormatter.with_spinner("Creating agent '#{agent_name}'") do
            resource = Kubernetes::ResourceBuilder.language_agent(
              agent_name,
              instructions: description,
              cluster: cluster_config[:namespace],
              persona: options[:persona],
              tools: options[:tools] || [],
              models: options[:models] || []
            )
            k8s.apply_resource(resource)
            resource
          end

          # Watch synthesis status
          watch_synthesis_status(k8s, agent_name, cluster_config[:namespace])

          # Display success
          Formatters::ProgressFormatter.success("Agent '#{agent_name}' created successfully!")
          puts
          puts 'Agent Details:'
          puts "  Name:         #{agent_name}"
          puts "  Cluster:      #{cluster}"
          puts "  Namespace:    #{cluster_config[:namespace]}"
          puts "  Instructions: #{description}"
          puts "  Persona:      #{options[:persona] || '(auto-selected)'}" if options[:persona]
          puts "  Tools:        #{options[:tools].join(', ')}" if options[:tools] && !options[:tools].empty?
          puts "  Models:       #{options[:models].join(', ')}" if options[:models] && !options[:models].empty?
          puts
          puts 'Next Steps:'
          puts "  aictl agent inspect #{agent_name}    # View agent details"
          puts "  aictl agent logs #{agent_name} -f    # Follow agent logs"
          puts "  aictl agent code #{agent_name}       # View synthesized code"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to create agent: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'list', 'List all agents in current cluster'
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :all_clusters, type: :boolean, default: false, desc: 'Show agents across all clusters'
        def list
          if options[:all_clusters]
            list_all_clusters
          else
            cluster = ClusterValidator.get_cluster(options[:cluster])
            list_cluster_agents(cluster)
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to list agents: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'inspect NAME', 'Show detailed agent information'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def inspect(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          agent = k8s.get_resource('LanguageAgent', name, cluster_config[:namespace])

          puts "Agent: #{name}"
          puts "  Cluster:   #{cluster}"
          puts "  Namespace: #{cluster_config[:namespace]}"
          puts

          # Status
          status = agent.dig('status', 'phase') || 'Unknown'
          puts "Status: #{format_status(status)}"
          puts

          # Spec details
          puts 'Configuration:'
          puts "  Mode:         #{agent.dig('spec', 'mode') || 'autonomous'}"
          puts "  Schedule:     #{agent.dig('spec', 'schedule') || 'N/A'}" if agent.dig('spec', 'schedule')
          puts "  Persona:      #{agent.dig('spec', 'persona') || '(auto-selected)'}"
          puts

          # Instructions
          instructions = agent.dig('spec', 'instructions')
          if instructions
            puts 'Instructions:'
            puts "  #{instructions}"
            puts
          end

          # Tools
          tools = agent.dig('spec', 'tools') || []
          if tools.any?
            puts "Tools (#{tools.length}):"
            tools.each { |tool| puts "  - #{tool}" }
            puts
          end

          # Models
          models = agent.dig('spec', 'models') || []
          if models.any?
            puts "Models (#{models.length}):"
            models.each { |model| puts "  - #{model}" }
            puts
          end

          # Synthesis info
          synthesis = agent.dig('status', 'synthesis')
          if synthesis
            puts 'Synthesis:'
            puts "  Status:       #{synthesis['status']}"
            puts "  Model:        #{synthesis['model']}" if synthesis['model']
            puts "  Completed:    #{synthesis['completedAt']}" if synthesis['completedAt']
            puts "  Duration:     #{synthesis['duration']}" if synthesis['duration']
            puts "  Token Count:  #{synthesis['tokenCount']}" if synthesis['tokenCount']
            puts
          end

          # Execution stats
          execution_count = agent.dig('status', 'executionCount') || 0
          last_execution = agent.dig('status', 'lastExecution')
          next_run = agent.dig('status', 'nextRun')

          puts 'Execution:'
          puts "  Total Runs:   #{execution_count}"
          puts "  Last Run:     #{last_execution || 'Never'}"
          puts "  Next Run:     #{next_run || 'N/A'}" if agent.dig('spec', 'schedule')
          puts

          # Conditions
          conditions = agent.dig('status', 'conditions') || []
          if conditions.any?
            puts "Conditions (#{conditions.length}):"
            conditions.each do |condition|
              status_icon = condition['status'] == 'True' ? '✓' : '✗'
              puts "  #{status_icon} #{condition['type']}: #{condition['message'] || condition['reason']}"
            end
            puts
          end

          # Recent events (if available)
          # This would require querying events, which we can add later

        rescue K8s::Error::NotFound
          Formatters::ProgressFormatter.error("Agent '#{name}' not found in cluster '#{cluster}'")
          exit 1
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to inspect agent: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'delete NAME', 'Delete an agent'
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :force, type: :boolean, default: false, desc: 'Skip confirmation'
        def delete(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get agent to show details before deletion
          begin
            agent = k8s.get_resource('LanguageAgent', name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Agent '#{name}' not found in cluster '#{cluster}'")
            exit 1
          end

          # Confirm deletion unless --force
          unless options[:force]
            puts "This will delete agent '#{name}' from cluster '#{cluster}':"
            puts "  Instructions: #{agent.dig('spec', 'instructions')}"
            puts "  Mode:         #{agent.dig('spec', 'mode') || 'autonomous'}"
            puts
            print 'Are you sure? (y/N): '
            confirmation = $stdin.gets.chomp
            unless confirmation.downcase == 'y'
              puts 'Deletion cancelled'
              return
            end
          end

          # Delete the agent
          Formatters::ProgressFormatter.with_spinner("Deleting agent '#{name}'") do
            k8s.delete_resource('LanguageAgent', name, cluster_config[:namespace])
          end

          Formatters::ProgressFormatter.success("Agent '#{name}' deleted successfully")
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to delete agent: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'logs NAME', 'Show agent execution logs'
        long_desc <<-DESC
          Stream agent execution logs in real-time.

          Use -f to follow logs continuously (like tail -f).

          Examples:
            aictl agent logs my-agent
            aictl agent logs my-agent -f
        DESC
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :follow, type: :boolean, aliases: '-f', default: false, desc: 'Follow logs'
        option :tail, type: :numeric, default: 100, desc: 'Number of lines to show from the end'
        def logs(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get agent to determine the pod name
          begin
            agent = k8s.get_resource('LanguageAgent', name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Agent '#{name}' not found in cluster '#{cluster}'")
            exit 1
          end

          mode = agent.dig('spec', 'mode') || 'autonomous'

          # Build kubectl command for log streaming
          kubeconfig_arg = cluster_config[:kubeconfig] ? "--kubeconfig=#{cluster_config[:kubeconfig]}" : ''
          context_arg = cluster_config[:context] ? "--context=#{cluster_config[:context]}" : ''
          namespace_arg = "-n #{cluster_config[:namespace]}"
          tail_arg = "--tail=#{options[:tail]}"
          follow_arg = options[:follow] ? '-f' : ''

          # For scheduled agents, logs come from CronJob pods
          # For autonomous agents, logs come from Deployment pods
          if mode == 'scheduled'
            # Get most recent job from cronjob
            label_selector = "app.kubernetes.io/name=#{name}"
          else
            # Get pod from deployment
            label_selector = "app.kubernetes.io/name=#{name}"
          end

          # Use kubectl logs with label selector
          cmd = "kubectl #{kubeconfig_arg} #{context_arg} #{namespace_arg} logs -l #{label_selector} #{tail_arg} #{follow_arg} --prefix --all-containers"

          Formatters::ProgressFormatter.info("Streaming logs for agent '#{name}'...")
          puts

          # Execute kubectl logs
          exec(cmd)
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to get logs: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'code NAME', 'Display synthesized agent code'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def code(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get the code ConfigMap for this agent
          configmap_name = "#{name}-code"
          begin
            configmap = k8s.get_resource('ConfigMap', configmap_name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Synthesized code not found for agent '#{name}'")
            puts
            puts 'Possible reasons:'
            puts '  - Agent synthesis not yet complete'
            puts '  - Agent synthesis failed'
            puts
            puts 'Check synthesis status with:'
            puts "  aictl agent inspect #{name}"
            exit 1
          end

          # Get the agent.rb code from the ConfigMap
          code_content = configmap.dig('data', 'agent.rb')
          unless code_content
            Formatters::ProgressFormatter.error('Code content not found in ConfigMap')
            exit 1
          end

          # Display with syntax highlighting if available
          puts "Synthesized Code for Agent: #{name}"
          puts '=' * 80
          puts
          puts code_content
          puts
          puts '=' * 80
          puts
          puts 'This code was automatically synthesized from the agent instructions.'
          puts "View full agent details with: aictl agent inspect #{name}"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to get code: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'edit NAME', 'Edit agent instructions'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def edit(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          # TODO: Implement agent editing
          Formatters::ProgressFormatter.warn('Agent editing not yet implemented')
          exit 1
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to edit agent: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'pause NAME', 'Pause scheduled agent execution'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def pause(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          # TODO: Implement agent pause
          Formatters::ProgressFormatter.warn('Agent pause not yet implemented')
          exit 1
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to pause agent: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'resume NAME', 'Resume paused agent'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def resume(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          # TODO: Implement agent resume
          Formatters::ProgressFormatter.warn('Agent resume not yet implemented')
          exit 1
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to resume agent: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        private

        def format_status(status)
          require 'pastel'
          pastel = Pastel.new

          case status.downcase
          when 'ready', 'running', 'active'
            "#{pastel.green('●')} #{status}"
          when 'pending', 'creating', 'synthesizing'
            "#{pastel.yellow('●')} #{status}"
          when 'failed', 'error'
            "#{pastel.red('●')} #{status}"
          when 'paused', 'stopped'
            "#{pastel.dim('●')} #{status}"
          else
            "#{pastel.dim('●')} #{status}"
          end
        end

        def generate_agent_name(description)
          # Simple name generation from description
          # Take first few words, lowercase, hyphenate
          words = description.downcase.gsub(/[^a-z0-9\s]/, '').split[0..2]
          name = words.join('-')
          # Add random suffix to avoid collisions
          "#{name}-#{Time.now.to_i.to_s[-4..]}"
        end

        def watch_synthesis_status(k8s, agent_name, namespace)
          require 'tty-spinner'
          require 'pastel'

          pastel = Pastel.new
          spinner = TTY::Spinner.new("[:spinner] Waiting for synthesis...", format: :dots)
          spinner.auto_spin

          max_wait = 60 # Wait up to 60 seconds
          interval = 2  # Check every 2 seconds
          elapsed = 0

          loop do
            agent = k8s.get_resource('LanguageAgent', agent_name, namespace)
            conditions = agent.dig('status', 'conditions') || []

            # Check for synthesis completion
            synthesized = conditions.find { |c| c['type'] == 'Synthesized' }
            if synthesized
              if synthesized['status'] == 'True'
                spinner.success("(#{pastel.green('✓')})")
                return true
              elsif synthesized['status'] == 'False'
                spinner.error("(#{pastel.red('✗')})")
                Formatters::ProgressFormatter.error("Synthesis failed: #{synthesized['message']}")
                return false
              end
            end

            # Timeout check
            if elapsed >= max_wait
              spinner.stop
              Formatters::ProgressFormatter.warn('Synthesis taking longer than expected, continuing in background...')
              puts
              puts 'Check synthesis status with:'
              puts "  aictl agent inspect #{agent_name}"
              return true
            end

            sleep interval
            elapsed += interval
          end
        rescue K8s::Error::NotFound
          # Agent not found yet, keep waiting
          sleep interval
          elapsed += interval
          retry if elapsed < max_wait

          spinner.error("(#{pastel.red('✗')})")
          Formatters::ProgressFormatter.error('Agent resource not found')
          false
        rescue StandardError => e
          spinner.error("(#{pastel.red('✗')})")
          Formatters::ProgressFormatter.warn("Could not watch synthesis: #{e.message}")
          true # Continue anyway
        end

        def list_cluster_agents(cluster)
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          Formatters::ProgressFormatter.info("Agents in cluster '#{cluster}'")

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])

          table_data = agents.map do |agent|
            {
              name: agent.dig('metadata', 'name'),
              mode: agent.dig('spec', 'mode') || 'autonomous',
              status: agent.dig('status', 'phase') || 'Unknown',
              next_run: agent.dig('status', 'nextRun') || 'N/A',
              executions: agent.dig('status', 'executionCount') || 0
            }
          end

          Formatters::TableFormatter.agents(table_data)

          if agents.empty?
            puts
            puts 'Create an agent with:'
            puts '  aictl agent create "<description>"'
          end
        end

        def list_all_clusters
          clusters = Config::ClusterConfig.list_clusters

          if clusters.empty?
            Formatters::ProgressFormatter.info('No clusters found')
            puts
            puts 'Create a cluster first:'
            puts '  aictl cluster create <name>'
            return
          end

          all_agents = []

          clusters.each do |cluster|
            begin
              k8s = Kubernetes::Client.new(
                kubeconfig: cluster[:kubeconfig],
                context: cluster[:context]
              )

              agents = k8s.list_resources('LanguageAgent', namespace: cluster[:namespace])

              agents.each do |agent|
                all_agents << {
                  cluster: cluster[:name],
                  name: agent.dig('metadata', 'name'),
                  mode: agent.dig('spec', 'mode') || 'autonomous',
                  status: agent.dig('status', 'phase') || 'Unknown',
                  next_run: agent.dig('status', 'nextRun') || 'N/A',
                  executions: agent.dig('status', 'executionCount') || 0
                }
              end
            rescue StandardError => e
              Formatters::ProgressFormatter.warn("Failed to get agents from cluster '#{cluster[:name]}': #{e.message}")
            end
          end

          if all_agents.empty?
            Formatters::ProgressFormatter.info('No agents found in any cluster')
            return
          end

          # Display with cluster column
          headers = ['CLUSTER', 'NAME', 'MODE', 'STATUS', 'NEXT RUN', 'EXECUTIONS']
          rows = all_agents.map do |agent|
            [
              agent[:cluster],
              agent[:name],
              agent[:mode],
              agent[:status],
              agent[:next_run],
              agent[:executions]
            ]
          end

          require 'tty-table'
          table = TTY::Table.new(headers, rows)
          puts table.render(:unicode, padding: [0, 1])
        end
      end
    end
  end
end
