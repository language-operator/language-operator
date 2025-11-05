# frozen_string_literal: true

require 'thor'
require_relative '../formatters/progress_formatter'
require_relative '../../config/cluster_config'

module LanguageOperator
  module CLI
    module Commands
      # Switch cluster context command
      class Use < Thor
        desc 'use CLUSTER', 'Switch to a different cluster context'
        def self.exit_on_failure?
          true
        end

        def switch(cluster_name)
          unless Config::ClusterConfig.cluster_exists?(cluster_name)
            Formatters::ProgressFormatter.error("Cluster '#{cluster_name}' not found")
            puts "\nAvailable clusters:"
            Config::ClusterConfig.list_clusters.each do |cluster|
              puts "  - #{cluster[:name]}"
            end
            exit 1
          end

          Config::ClusterConfig.set_current_cluster(cluster_name)
          cluster = Config::ClusterConfig.get_cluster(cluster_name)

          Formatters::ProgressFormatter.success("Switched to cluster '#{cluster_name}'")
          puts "\nCluster Details:"
          puts "  Name:      #{cluster[:name]}"
          puts "  Namespace: #{cluster[:namespace]}"
          puts "  Context:   #{cluster[:context] || 'default'}"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to switch cluster: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end
      end
    end
  end
end
