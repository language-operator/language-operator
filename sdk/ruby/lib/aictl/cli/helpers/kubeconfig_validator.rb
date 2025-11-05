# frozen_string_literal: true

require_relative '../formatters/progress_formatter'
require_relative '../../kubernetes/client'

module Aictl
  module CLI
    module Helpers
      # Validates kubeconfig and cluster connectivity
      class KubeconfigValidator
        class << self
          # Validate kubeconfig exists and cluster is accessible
          # Returns [valid, error_message]
          def validate
            # Check if kubeconfig file exists
            kubeconfig_path = detect_kubeconfig
            unless kubeconfig_path && File.exist?(kubeconfig_path)
              return [false, kubeconfig_missing_message(kubeconfig_path)]
            end

            # Try to connect to cluster
            begin
              k8s = Kubernetes::Client.new(kubeconfig: kubeconfig_path)

              # Test connectivity by listing namespaces
              k8s.client.api('v1').resource('namespaces').list

              # Check if operator is installed
              unless k8s.operator_installed?
                return [false, operator_missing_message]
              end

              [true, nil]
            rescue Errno::ECONNREFUSED, Errno::ETIMEDOUT => e
              [false, connection_failed_message(e)]
            rescue K8s::Error::Unauthorized => e
              [false, auth_failed_message(e)]
            rescue StandardError => e
              [false, generic_error_message(e)]
            end
          end

          # Validate and exit with error if invalid
          def validate!
            valid, error_message = validate
            unless valid
              Formatters::ProgressFormatter.error(error_message)
              exit 1
            end
          end

          # Detect kubeconfig path
          def detect_kubeconfig
            ENV.fetch('KUBECONFIG', nil) || default_kubeconfig_path
          end

          # Check if kubeconfig exists
          def kubeconfig_exists?
            path = detect_kubeconfig
            path && File.exist?(path)
          end

          private

          def default_kubeconfig_path
            File.expand_path('~/.kube/config')
          end

          def kubeconfig_missing_message(path)
            <<~MSG
              Kubeconfig file not found

              Expected location: #{path || default_kubeconfig_path}

              To fix this issue:
              1. Ensure you have a Kubernetes cluster configured
              2. Set KUBECONFIG environment variable to point to your kubeconfig file:
                 export KUBECONFIG=/path/to/your/kubeconfig

              Or place your kubeconfig at: ~/.kube/config

              For local development, you can use:
              - kind (https://kind.sigs.k8s.io/)
              - k3d (https://k3d.io/)
              - minikube (https://minikube.sigs.k8s.io/)
            MSG
          end

          def connection_failed_message(error)
            <<~MSG
              Failed to connect to Kubernetes cluster

              Error: #{error.message}

              To fix this issue:
              1. Check if your cluster is running:
                 kubectl cluster-info

              2. Verify your kubeconfig is correct:
                 kubectl config view

              3. Check your cluster context:
                 kubectl config current-context

              4. Test basic connectivity:
                 kubectl get namespaces
            MSG
          end

          def auth_failed_message(error)
            <<~MSG
              Kubernetes authentication failed

              Error: #{error.message}

              To fix this issue:
              1. Verify your credentials are valid:
                 kubectl config view

              2. Check if your authentication token/certificate is expired

              3. Re-authenticate with your cluster provider

              4. Test authentication:
                 kubectl get namespaces
            MSG
          end

          def operator_missing_message
            <<~MSG
              Language Operator is not installed in the cluster

              The Language Operator CRDs were not found in the cluster.

              To install the operator:

              1. Using Helm:
                 helm repo add langop https://charts.langop.io
                 helm install language-operator langop/language-operator \\
                   --namespace kube-system

              2. Or from OCI registry:
                 helm install language-operator oci://git.theryans.io/langop/charts/language-operator \\
                   --namespace kube-system

              3. Verify installation:
                 kubectl get deployment -n kube-system language-operator

              For more information, visit: https://github.com/langop/language-operator
            MSG
          end

          def generic_error_message(error)
            <<~MSG
              Unexpected error validating cluster connection

              Error: #{error.class}: #{error.message}

              Please check:
              1. Your kubeconfig file is valid
              2. Your cluster is accessible
              3. You have appropriate permissions

              For debugging, run with DEBUG=1:
                DEBUG=1 aictl <command>
            MSG
          end
        end
      end
    end
  end
end
