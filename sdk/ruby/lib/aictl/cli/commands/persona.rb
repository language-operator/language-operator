# frozen_string_literal: true

require 'thor'
require 'yaml'
require_relative '../formatters/progress_formatter'
require_relative '../formatters/table_formatter'
require_relative '../helpers/cluster_validator'
require_relative '../../config/cluster_config'
require_relative '../../kubernetes/client'
require_relative '../../kubernetes/resource_builder'

module Aictl
  module CLI
    module Commands
      # Persona management commands
      class Persona < Thor
        include Helpers::ClusterValidator

        desc 'list', 'List all personas in current cluster'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def list
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          personas = k8s.list_resources('LanguagePersona', namespace: cluster_config[:namespace])

          if personas.empty?
            Formatters::ProgressFormatter.info("No personas found in cluster '#{cluster}'")
            puts
            puts 'Personas define the personality and capabilities of agents.'
            puts
            puts 'Create a persona with:'
            puts '  aictl persona create <name>'
            return
          end

          # Get agents to count usage
          agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])

          table_data = personas.map do |persona|
            name = persona.dig('metadata', 'name')
            used_by = agents.count { |a| a.dig('spec', 'persona') == name }

            {
              name: name,
              tone: persona.dig('spec', 'tone') || 'neutral',
              used_by: used_by,
              description: persona.dig('spec', 'description') || ''
            }
          end

          Formatters::TableFormatter.personas(table_data)
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to list personas: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'show NAME', 'Display full persona details'
        option :cluster, type: :string, desc: 'Override current cluster context'
        def show(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          begin
            persona = k8s.get_resource('LanguagePersona', name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Persona '#{name}' not found in cluster '#{cluster}'")
            exit 1
          end

          puts "Persona: #{name}"
          puts '=' * 80
          puts
          puts YAML.dump(persona)
          puts
          puts '=' * 80
          puts
          puts 'Use this persona when creating agents:'
          puts "  aictl agent create \"description\" --persona #{name}"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to show persona: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'delete NAME', 'Delete a persona'
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :force, type: :boolean, default: false, desc: 'Skip confirmation'
        def delete(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get persona
          begin
            persona = k8s.get_resource('LanguagePersona', name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Persona '#{name}' not found in cluster '#{cluster}'")
            exit 1
          end

          # Check for agents using this persona
          agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])
          agents_using = agents.select { |a| a.dig('spec', 'persona') == name }

          if agents_using.any? && !options[:force]
            Formatters::ProgressFormatter.warn("Persona '#{name}' is in use by #{agents_using.count} agent(s)")
            puts
            puts 'Agents using this persona:'
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
            puts "This will delete persona '#{name}' from cluster '#{cluster}':"
            puts "  Tone:        #{persona.dig('spec', 'tone')}"
            puts "  Description: #{persona.dig('spec', 'description')}"
            puts
            print 'Are you sure? (y/N): '
            confirmation = $stdin.gets.chomp
            unless confirmation.downcase == 'y'
              puts 'Deletion cancelled'
              return
            end
          end

          # Delete persona
          Formatters::ProgressFormatter.with_spinner("Deleting persona '#{name}'") do
            k8s.delete_resource('LanguagePersona', name, cluster_config[:namespace])
          end

          Formatters::ProgressFormatter.success("Persona '#{name}' deleted successfully")
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to delete persona: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end
      end
    end
  end
end
