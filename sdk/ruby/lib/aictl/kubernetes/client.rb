# frozen_string_literal: true

require 'k8s-ruby'
require 'yaml'

module Aictl
  module Kubernetes
    # Kubernetes client wrapper for interacting with language-operator resources
    class Client
      attr_reader :client

      def initialize(kubeconfig: nil, context: nil)
        @kubeconfig = kubeconfig || ENV.fetch('KUBECONFIG', File.expand_path('~/.kube/config'))
        @context = context
        @client = build_client
      end

      # Create or update a Kubernetes resource
      def apply_resource(resource)
        namespace = resource.dig('metadata', 'namespace')
        name = resource.dig('metadata', 'name')
        kind = resource['kind']
        api_version = resource['apiVersion']

        begin
          # Try to get existing resource
          existing = get_resource(kind, name, namespace, api_version)
          if existing
            # Update existing resource
            update_resource(kind, name, namespace, resource, api_version)
          else
            # Create new resource
            create_resource(resource)
          end
        rescue K8s::Error::NotFound
          # Resource doesn't exist, create it
          create_resource(resource)
        end
      end

      # Create a resource
      def create_resource(resource)
        api_client = api_for_resource(resource)
        api_client.create_resource(resource)
      end

      # Update a resource
      def update_resource(kind, name, namespace, resource, api_version)
        api_client = api_for_version(api_version)
        api_client.update_resource(resource)
      end

      # Get a resource
      def get_resource(kind, name, namespace = nil, api_version = nil)
        api_client = api_for_version(api_version || default_api_version(kind))
        if namespace
          api_client.get_resource(kind, name, namespace)
        else
          api_client.get_resource(kind, name)
        end
      end

      # List resources
      def list_resources(kind, namespace: nil, api_version: nil, label_selector: nil)
        api_client = api_for_version(api_version || default_api_version(kind))
        opts = {}
        opts[:labelSelector] = label_selector if label_selector

        if namespace
          api_client.list_resources(kind, namespace, **opts)
        else
          api_client.list_resources(kind, **opts)
        end
      end

      # Delete a resource
      def delete_resource(kind, name, namespace = nil, api_version = nil)
        api_client = api_for_version(api_version || default_api_version(kind))
        if namespace
          api_client.delete_resource(kind, name, namespace)
        else
          api_client.delete_resource(kind, name)
        end
      end

      # Check if namespace exists
      def namespace_exists?(name)
        @client.api('v1').resource('namespaces').get(name)
        true
      rescue K8s::Error::NotFound
        false
      end

      # Create namespace
      def create_namespace(name, labels: {})
        resource = {
          'apiVersion' => 'v1',
          'kind' => 'Namespace',
          'metadata' => {
            'name' => name,
            'labels' => labels
          }
        }
        create_resource(resource)
      end

      # Check if operator is installed
      def operator_installed?
        # Check if LanguageCluster CRD exists
        @client.apis(prefetch_resources: true)
                .find { |api| api.group_version == 'langop.io/v1alpha1' }
      rescue StandardError
        false
      end

      # Get operator version
      def operator_version
        deployment = @client.api('apps/v1')
                           .resource('deployments', namespace: 'kube-system')
                           .get('language-operator')
        deployment.dig('metadata', 'labels', 'app.kubernetes.io/version') || 'unknown'
      rescue K8s::Error::NotFound
        nil
      end

      private

      def build_client
        config = K8s::Config.load_file(@kubeconfig)
        config = config.context(@context) if @context
        K8s::Client.new(config)
      end

      def api_for_resource(resource)
        api_version = resource['apiVersion']
        api_for_version(api_version)
      end

      def api_for_version(api_version)
        if api_version.include?('/')
          group, version = api_version.split('/', 2)
          @client.api("#{group}/#{version}")
        else
          @client.api(api_version)
        end
      end

      def default_api_version(kind)
        case kind.downcase
        when 'languagecluster', 'languageagent', 'languagetool', 'languagemodel', 'languageclient', 'languagepersona'
          'langop.io/v1alpha1'
        when 'namespace', 'configmap', 'secret', 'service'
          'v1'
        when 'deployment', 'statefulset'
          'apps/v1'
        when 'cronjob'
          'batch/v1'
        else
          'v1'
        end
      end
    end
  end
end
