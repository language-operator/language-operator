# frozen_string_literal: true

require_relative '../formatters/progress_formatter'
require_relative 'kubeconfig_validator'
require_relative '../../config/cluster_config'

module LanguageOperator
  module CLI
    module Helpers
      # Validates that a cluster is selected before executing commands
      module ClusterValidator
        class << self
          # Ensure a cluster is selected, exit with helpful message if not
          def ensure_cluster_selected!
            return current_cluster if current_cluster

            Formatters::ProgressFormatter.error('No cluster selected')
            puts "\nYou must select a cluster before managing agents."
            puts
            puts 'Create a new cluster:'
            puts '  aictl cluster create <name>'
            puts
            puts 'Or select an existing cluster:'
            puts '  aictl use <cluster>'
            puts
            puts 'List available clusters:'
            puts '  aictl cluster list'
            exit 1
          end

          # Get current cluster, or allow override via --cluster flag
          def get_cluster(cluster_override = nil)
            if cluster_override
              validate_cluster_exists!(cluster_override)
              cluster_override
            else
              ensure_cluster_selected!
            end
          end

          # Validate that a specific cluster exists
          def validate_cluster_exists!(name)
            return if Config::ClusterConfig.cluster_exists?(name)

            Formatters::ProgressFormatter.error("Cluster '#{name}' not found")
            puts "\nAvailable clusters:"
            clusters = Config::ClusterConfig.list_clusters
            if clusters.empty?
              puts '  (none)'
              puts
              puts 'Create a cluster first:'
              puts '  aictl cluster create <name>'
            else
              clusters.each do |cluster|
                puts "  - #{cluster[:name]}"
              end
            end
            exit 1
          end

          # Get current cluster name
          def current_cluster
            Config::ClusterConfig.current_cluster
          end

          # Get current cluster config
          def current_cluster_config
            cluster_name = ensure_cluster_selected!
            Config::ClusterConfig.get_cluster(cluster_name)
          end

          # Get cluster config by name (with validation)
          def get_cluster_config(name)
            validate_cluster_exists!(name)
            config = Config::ClusterConfig.get_cluster(name)

            # Validate kubeconfig exists and is accessible
            validate_kubeconfig!(config)

            config
          end

          # Validate kubeconfig for the given cluster config
          def validate_kubeconfig!(cluster_config)
            # Check if kubeconfig file exists
            kubeconfig_path = cluster_config[:kubeconfig]
            unless kubeconfig_path && File.exist?(kubeconfig_path)
              Formatters::ProgressFormatter.error("Kubeconfig not found: #{kubeconfig_path}")
              puts
              puts 'The kubeconfig file for this cluster does not exist.'
              puts
              puts 'To fix this issue:'
              puts '  1. Verify the kubeconfig path in ~/.aictl/config.yaml'
              puts '  2. Re-create the cluster configuration:'
              puts "     aictl cluster create #{cluster_config[:name]}"
              exit 1
            end
          end
        end
      end
    end
  end
end
