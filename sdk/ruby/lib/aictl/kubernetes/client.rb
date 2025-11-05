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

      # Get the current namespace from the context
      def current_namespace
        config = K8s::Config.load_file(@kubeconfig)
        config = config.context(@context) if @context
        config.context&.namespace
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
            # Merge existing metadata (especially resourceVersion) with new resource
            merged_resource = if resource.is_a?(Hash)
                                resource.dup
                              else
                                resource.to_h
                              end
            merged_resource['metadata'] ||= {}
            merged_resource['metadata']['resourceVersion'] = existing.metadata.resourceVersion
            merged_resource['metadata']['uid'] = existing.metadata.uid if existing.metadata.uid

            # Update existing resource
            update_resource(kind, name, namespace, merged_resource, api_version)
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
        resource_client = resource_client_for_resource(resource)
        # Convert hash to K8s::Resource if needed
        k8s_resource = resource.is_a?(K8s::Resource) ? resource : K8s::Resource.new(resource)
        resource_client.create_resource(k8s_resource)
      end

      # Update a resource
      def update_resource(kind, name, namespace, resource, api_version)
        resource_client = resource_client_for(kind, namespace, api_version)
        # Convert hash to K8s::Resource if needed
        k8s_resource = resource.is_a?(K8s::Resource) ? resource : K8s::Resource.new(resource)
        resource_client.update_resource(k8s_resource)
      end

      # Get a resource
      def get_resource(kind, name, namespace = nil, api_version = nil)
        resource_client = resource_client_for(kind, namespace, api_version || default_api_version(kind))
        resource_client.get(name)
      end

      # List resources
      def list_resources(kind, namespace: nil, api_version: nil, label_selector: nil)
        resource_client = resource_client_for(kind, namespace, api_version || default_api_version(kind))
        opts = {}
        opts[:labelSelector] = label_selector if label_selector

        resource_client.list(**opts)
      end

      # Delete a resource
      def delete_resource(kind, name, namespace = nil, api_version = nil)
        resource_client = resource_client_for(kind, namespace, api_version || default_api_version(kind))
        resource_client.delete(name)
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
                .find { |api| api.api_version == 'langop.io/v1alpha1' }
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
        K8s::Client.config(config)
      end

      def resource_client_for_resource(resource)
        kind = resource['kind']
        namespace = resource.dig('metadata', 'namespace')
        api_version = resource['apiVersion']
        resource_client_for(kind, namespace, api_version)
      end

      def resource_client_for(kind, namespace, api_version)
        api_client = api_for_version(api_version)
        resource_name = kind_to_resource_name(kind)
        if namespace
          api_client.resource(resource_name, namespace: namespace)
        else
          api_client.resource(resource_name)
        end
      end

      def api_for_version(api_version)
        if api_version.include?('/')
          group, version = api_version.split('/', 2)
          @client.api("#{group}/#{version}")
        else
          @client.api(api_version)
        end
      end

      def kind_to_resource_name(kind)
        # Convert Kind (singular, capitalized) to resource name (plural, lowercase)
        case kind.downcase
        when 'languagecluster'
          'languageclusters'
        when 'languageagent'
          'languageagents'
        when 'languagetool'
          'languagetools'
        when 'languagemodel'
          'languagemodels'
        when 'languageclient'
          'languageclients'
        when 'languagepersona'
          'languagepersonas'
        when 'namespace'
          'namespaces'
        when 'configmap'
          'configmaps'
        when 'secret'
          'secrets'
        when 'service'
          'services'
        when 'deployment'
          'deployments'
        when 'statefulset'
          'statefulsets'
        when 'cronjob'
          'cronjobs'
        else
          # Generic pluralization - add 's'
          "#{kind.downcase}s"
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
