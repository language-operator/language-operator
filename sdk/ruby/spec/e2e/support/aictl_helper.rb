# frozen_string_literal: true

require 'open3'
require 'json'
require 'timeout'

module E2E
  # Helper module for running aictl commands in E2E tests
  #
  # Provides utilities for:
  # - Running aictl commands and capturing output
  # - Waiting for resources to reach desired states
  # - Cleaning up test resources
  # - Validating synthesis and deployment
  module LanguageOperatorHelper
    # Run an aictl command and return stdout, stderr, and exit status
    #
    # @param command [String] The aictl command to run (without 'aictl' prefix)
    # @param timeout [Integer] Maximum seconds to wait for command completion
    # @return [Hash] Hash with :stdout, :stderr, :status, :success keys
    #
    # @example
    #   result = run_aictl("cluster list")
    #   puts result[:stdout] if result[:success]
    def run_aictl(command, timeout: 30)
      full_command = "aictl #{command}"
      stdout, stderr, status = nil

      Timeout.timeout(timeout) do
        stdout, stderr, status = Open3.capture3(full_command)
      end

      {
        stdout: stdout,
        stderr: stderr,
        status: status.exitstatus,
        success: status.success?
      }
    rescue Timeout::Error
      {
        stdout: '',
        stderr: "Command timed out after #{timeout}s",
        status: 124,
        success: false
      }
    end

    # Wait for a condition to become true
    #
    # @param timeout [Integer] Maximum seconds to wait
    # @param interval [Float] Seconds between checks
    # @yield Block that returns true when condition is met
    # @return [Boolean] True if condition met, false if timeout
    #
    # @example
    #   wait_for_condition(timeout: 60) { agent_ready?("my-agent") }
    def wait_for_condition(timeout: 30, interval: 1.0)
      deadline = Time.now + timeout
      loop do
        return true if yield

        return false if Time.now >= deadline

        sleep interval
      end
    end

    # Check if an agent exists
    #
    # @param name [String] Agent name
    # @param cluster [String, nil] Cluster name (uses current if nil)
    # @return [Boolean] True if agent exists
    def agent_exists?(name, cluster: nil)
      cmd = cluster ? "agent list --cluster #{cluster}" : 'agent list'
      result = run_aictl(cmd)
      result[:success] && result[:stdout].include?(name)
    end

    # Check if an agent has been synthesized
    #
    # @param name [String] Agent name
    # @param cluster [String, nil] Cluster name (uses current if nil)
    # @return [Boolean] True if agent is synthesized
    def agent_synthesized?(name, cluster: nil)
      result = run_aictl("agent inspect #{name}")
      return false unless result[:success]

      # Check for "Synthesized: True" or similar in output
      result[:stdout].match?(/Synthesized.*True/i)
    end

    # Check if agent pods are running
    #
    # @param name [String] Agent name
    # @param namespace [String] Kubernetes namespace
    # @return [Boolean] True if agent pod is running
    def agent_running?(name, namespace: 'default')
      # Use kubectl directly for pod status
      stdout, _, status = Open3.capture3(
        "kubectl get pods -n #{namespace} -l app=#{name} -o json"
      )
      return false unless status.success?

      begin
        data = JSON.parse(stdout)
        return false if data['items'].empty?

        pod = data['items'].first
        pod.dig('status', 'phase') == 'Running' &&
          pod.dig('status', 'containerStatuses')&.all? { |c| c['ready'] }
      rescue JSON::ParserError
        false
      end
    end

    # Get agent logs
    #
    # @param name [String] Agent name
    # @param lines [Integer] Number of lines to retrieve
    # @return [String, nil] Log output or nil if failed
    def get_agent_logs(name, lines: 50)
      result = run_aictl("agent logs #{name} --tail #{lines}")
      result[:success] ? result[:stdout] : nil
    end

    # Check if a cluster exists
    #
    # @param name [String] Cluster name
    # @return [Boolean] True if cluster exists
    def cluster_exists?(name)
      result = run_aictl('cluster list')
      result[:success] && result[:stdout].include?(name)
    end

    # Delete an agent
    #
    # @param name [String] Agent name
    # @param cluster [String, nil] Cluster name (uses current if nil)
    # @return [Boolean] True if deletion succeeded
    def delete_agent(name, cluster: nil)
      cmd = "agent delete #{name}"
      cmd += " --cluster #{cluster}" if cluster
      result = run_aictl(cmd)
      result[:success]
    end

    # Delete a cluster
    #
    # @param name [String] Cluster name
    # @return [Boolean] True if deletion succeeded
    def delete_cluster(name)
      result = run_aictl("cluster delete #{name} --yes")
      result[:success]
    end

    # Edit agent instructions
    #
    # @param name [String] Agent name
    # @param new_instructions [String] New instruction text
    # @return [Boolean] True if edit succeeded
    def edit_agent(name, new_instructions)
      # Create a temporary file with the new instructions
      require 'tempfile'
      Tempfile.create(['agent-instructions', '.txt']) do |f|
        f.write(new_instructions)
        f.flush

        # Set EDITOR to cat the temp file content
        result = run_aictl("agent edit #{name}")
        return result[:success]
      end
    end

    # Clean up all test resources with a given prefix
    #
    # @param prefix [String] Prefix to match (e.g., "e2e-test")
    def cleanup_test_resources(prefix)
      # Delete agents with prefix
      result = run_aictl('agent list --all-clusters')
      if result[:success]
        result[:stdout].lines.each do |line|
          next unless line.include?(prefix)

          # Extract agent name (assumes format: NAME | CLUSTER | ...)
          parts = line.split('|').map(&:strip)
          next if parts.empty?

          agent_name = parts[0]
          delete_agent(agent_name) if agent_name.start_with?(prefix)
        end
      end

      # Delete clusters with prefix
      result = run_aictl('cluster list')
      return unless result[:success]

      result[:stdout].lines.each do |line|
        next unless line.include?(prefix)

        parts = line.split('|').map(&:strip)
        next if parts.empty?

        cluster_name = parts[0]
        delete_cluster(cluster_name) if cluster_name.start_with?(prefix)
      end
    end
  end
end
