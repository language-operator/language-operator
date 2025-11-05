# frozen_string_literal: true

module Aictl
  module Kubernetes
    # Builds Kubernetes resource manifests for language-operator
    class ResourceBuilder
      class << self
        # Build a LanguageCluster resource
        def language_cluster(name, namespace: nil, labels: {})
          {
            'apiVersion' => 'langop.io/v1alpha1',
            'kind' => 'LanguageCluster',
            'metadata' => {
              'name' => name,
              'namespace' => namespace || 'default',
              'labels' => default_labels.merge(labels)
            },
            'spec' => {
              'namespace' => namespace || name,
              'resourceQuota' => default_resource_quota,
              'networkPolicy' => default_network_policy
            }
          }
        end

        # Build a LanguageAgent resource
        def language_agent(name, instructions:, cluster: nil, schedule: nil, persona: nil, tools: [], models: [], labels: {})
          spec = {
            'instructions' => instructions,
            'mode' => schedule ? 'scheduled' : 'autonomous'
          }

          spec['schedule'] = schedule if schedule
          spec['persona'] = persona if persona
          spec['tools'] = tools unless tools.empty?
          spec['models'] = models unless models.empty?

          {
            'apiVersion' => 'langop.io/v1alpha1',
            'kind' => 'LanguageAgent',
            'metadata' => {
              'name' => name,
              'namespace' => cluster || 'default',
              'labels' => default_labels.merge(labels)
            },
            'spec' => spec
          }
        end

        # Build a LanguageTool resource
        def language_tool(name, type:, config: {}, cluster: nil, labels: {})
          {
            'apiVersion' => 'langop.io/v1alpha1',
            'kind' => 'LanguageTool',
            'metadata' => {
              'name' => name,
              'namespace' => cluster || 'default',
              'labels' => default_labels.merge(labels)
            },
            'spec' => {
              'type' => type,
              'config' => config
            }
          }
        end

        # Build a LanguageModel resource
        def language_model(name, provider:, model:, endpoint: nil, cluster: nil, labels: {})
          spec = {
            'provider' => provider,
            'model' => model
          }
          spec['endpoint'] = endpoint if endpoint

          {
            'apiVersion' => 'langop.io/v1alpha1',
            'kind' => 'LanguageModel',
            'metadata' => {
              'name' => name,
              'namespace' => cluster || 'default',
              'labels' => default_labels.merge(labels)
            },
            'spec' => spec
          }
        end

        # Build a LanguagePersona resource
        def language_persona(name, description:, tone:, system_prompt:, cluster: nil, labels: {})
          {
            'apiVersion' => 'langop.io/v1alpha1',
            'kind' => 'LanguagePersona',
            'metadata' => {
              'name' => name,
              'namespace' => cluster || 'default',
              'labels' => default_labels.merge(labels)
            },
            'spec' => {
              'displayName' => name.split('-').map(&:capitalize).join(' '),
              'description' => description,
              'tone' => tone,
              'systemPrompt' => system_prompt
            }
          }
        end

        private

        def default_labels
          {
            'app.kubernetes.io/managed-by' => 'aictl',
            'app.kubernetes.io/part-of' => 'language-operator'
          }
        end

        def default_resource_quota
          {
            'hard' => {
              'requests.cpu' => '4',
              'requests.memory' => '8Gi',
              'limits.cpu' => '8',
              'limits.memory' => '16Gi'
            }
          }
        end

        def default_network_policy
          {
            'egress' => {
              'allowDNS' => ['8.8.8.8/32', '8.8.4.4/32'],
              'allowHTTPS' => ['0.0.0.0/0']
            }
          }
        end
      end
    end
  end
end
