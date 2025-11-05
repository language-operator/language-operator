# frozen_string_literal: true

require 'yaml'
require 'fileutils'

module Aictl
  module Config
    # Manages cluster configuration in ~/.aictl/config.yaml
    class ClusterConfig
      CONFIG_DIR = File.expand_path('~/.aictl')
      CONFIG_PATH = File.join(CONFIG_DIR, 'config.yaml')

      class << self
        def load
          return default_config unless File.exist?(CONFIG_PATH)

          YAML.load_file(CONFIG_PATH) || default_config
        rescue StandardError => e
          warn "Warning: Failed to load config from #{CONFIG_PATH}: #{e.message}"
          default_config
        end

        def save(config)
          FileUtils.mkdir_p(CONFIG_DIR)
          File.write(CONFIG_PATH, YAML.dump(config))
        end

        def current_cluster
          config = load
          config['current-cluster']
        end

        def set_current_cluster(name)
          config = load
          unless cluster_exists?(name)
            raise ArgumentError, "Cluster '#{name}' does not exist"
          end

          config['current-cluster'] = name
          save(config)
        end

        def add_cluster(name, namespace, kubeconfig, context)
          config = load
          config['clusters'] ||= []

          # Remove existing cluster with same name
          config['clusters'].reject! { |c| c['name'] == name }

          # Add new cluster
          config['clusters'] << {
            'name' => name,
            'namespace' => namespace,
            'kubeconfig' => kubeconfig,
            'context' => context,
            'created' => Time.now.utc.iso8601
          }

          save(config)
        end

        def remove_cluster(name)
          config = load
          config['clusters']&.reject! { |c| c['name'] == name }

          # Clear current-cluster if it was the removed one
          config['current-cluster'] = nil if config['current-cluster'] == name

          save(config)
        end

        def get_cluster(name)
          config = load
          config['clusters']&.find { |c| c['name'] == name }
        end

        def list_clusters
          config = load
          config['clusters'] || []
        end

        def cluster_exists?(name)
          !get_cluster(name).nil?
        end

        private

        def default_config
          {
            'apiVersion' => 'aictl.langop.io/v1',
            'kind' => 'Config',
            'current-cluster' => nil,
            'clusters' => []
          }
        end
      end
    end
  end
end
