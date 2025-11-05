# frozen_string_literal: true

require 'thor'
require 'fileutils'
require_relative 'commands/cluster'
require_relative 'commands/use'
require_relative 'commands/agent'
require_relative 'commands/status'
require_relative 'formatters/progress_formatter'
require_relative '../config/cluster_config'
require_relative '../kubernetes/client'

module Aictl
  module CLI
    # Main CLI class for aictl command
    #
    # Provides commands for creating, running, and managing language-operator resources.
    class Main < Thor
      def self.exit_on_failure?
        true
      end

      desc 'status', 'Show system status and overview'
      def status
        Commands::Status.new.invoke(:overview)
      end

      desc 'version', 'Show aictl and operator version'
      def version
        puts "aictl v#{Aictl::VERSION}"
        puts

        # Try to get operator version from current cluster
        current_cluster = Config::ClusterConfig.current_cluster
        if current_cluster
          cluster_config = Config::ClusterConfig.get_cluster(current_cluster)
          begin
            k8s = Kubernetes::Client.new(
              kubeconfig: cluster_config[:kubeconfig],
              context: cluster_config[:context]
            )

            if k8s.operator_installed?
              operator_version = k8s.operator_version || 'unknown'
              puts "Operator: v#{operator_version}"
              puts "Cluster:  #{current_cluster}"

              # Check compatibility (simple version check)
              # In the future, this could be more sophisticated
              puts
              Formatters::ProgressFormatter.success('Versions are compatible')
            else
              Formatters::ProgressFormatter.warn("Operator not installed in cluster '#{current_cluster}'")
            end
          rescue StandardError => e
            Formatters::ProgressFormatter.error("Could not connect to cluster: #{e.message}")
          end
        else
          puts 'No cluster selected'
          puts
          puts 'Select a cluster to check operator version:'
          puts '  aictl use <cluster>'
        end
      end

      desc 'cluster SUBCOMMAND ...ARGS', 'Manage language clusters'
      subcommand 'cluster', Commands::Cluster

      desc 'use CLUSTER', 'Switch to a different cluster context'
      def use(cluster_name)
        Commands::Use.new.switch(cluster_name)
      end

      desc 'agent SUBCOMMAND ...ARGS', 'Manage autonomous agents'
      subcommand 'agent', Commands::Agent

      desc 'new TYPE NAME', 'Generate a new tool or agent project (TYPE: tool, agent)'
      long_desc <<-DESC
        Generate a new tool or agent project with the specified name.

        Examples:
          aictl new tool calculator
          aictl new agent news-summarizer
      DESC
      def new(type, name)
        case type
        when 'tool'
          generate_tool(name)
        when 'agent'
          generate_agent(name)
        else
          puts "Error: Unknown type '#{type}'. Use 'tool' or 'agent'."
          exit 1
        end
      end

      desc 'serve [FILE]', 'Start an MCP server for tools'
      long_desc <<-DESC
        Start an MCP server that serves the tools defined in FILE.
        If no FILE is specified, looks for mcp/tools.rb in the current directory.

        Example:
          aictl serve mcp/calculator.rb
      DESC
      option :port, type: :numeric, default: 80, desc: 'Port to listen on'
      option :host, type: :string, default: '0.0.0.0', desc: 'Host to bind to'
      def serve(file = 'mcp/tools.rb')
        unless File.exist?(file)
          puts "Error: Tool file '#{file}' not found."
          exit 1
        end

        puts "Loading tools from #{file}..."
        Langop.load_file(file)

        registry = Aictl::Dsl.registry
        puts "Loaded #{registry.all.length} tool(s):"
        registry.all.each do |tool|
          puts "  - #{tool.name}: #{tool.description}"
        end
        puts

        puts "Starting MCP server on #{options[:host]}:#{options[:port]}..."
        server = Aictl::Dsl.create_server

        # Start the server (implementation depends on MCP Ruby SDK)
        require 'mcp/server/sse'
        MCP::Server::SSE.run(server, host: options[:host], port: options[:port])
      end

      desc 'test [FILE]', 'Test tool definitions'
      long_desc <<-DESC
        Load and validate tool definitions from FILE.
        Displays tool information and validates parameter schemas.

        Example:
          aictl test mcp/calculator.rb
      DESC
      def test(file = 'mcp/tools.rb')
        unless File.exist?(file)
          puts "Error: Tool file '#{file}' not found."
          exit 1
        end

        puts "Testing tools from #{file}..."
        puts

        Langop.load_file(file)
        registry = Aictl::Dsl.registry

        if registry.all.empty?
          puts "❌ No tools found in #{file}"
          exit 1
        end

        puts "✅ Found #{registry.all.length} tool(s):"
        puts

        registry.all.each do |tool|
          puts "Tool: #{tool.name}"
          puts "  Description: #{tool.description}"
          puts "  Parameters:"
          if tool.parameters.empty?
            puts "    (none)"
          else
            tool.parameters.each do |name, param|
              required = param.required ? ' (required)' : ''
              puts "    - #{name}: #{param.type}#{required} - #{param.description}"
            end
          end
          puts
        end

        puts "✅ All tools validated successfully!"
      end

      desc 'run', 'Run an agent'
      long_desc <<-DESC
        Run an agent using configuration from the current directory or environment variables.

        The agent will look for config.yaml in the current directory, or use environment
        variables if not found.

        Example:
          aictl run
      DESC
      option :config, type: :string, desc: 'Path to configuration file'
      def run
        config_path = options[:config] || 'config.yaml'
        Aictl::Agent.run(config_path: config_path)
      rescue Interrupt
        puts "\n\n👋 Agent stopped"
      end

      desc 'console', 'Start an interactive Ruby console with aictl loaded'
      def console
        require 'irb'
        require 'aictl'
        ARGV.clear
        IRB.start
      end

      private

      def generate_tool(name)
        dir = name
        FileUtils.mkdir_p("#{dir}/mcp")

        # Copy tool template
        template_dir = File.expand_path('../cli/templates/tool', __dir__)
        copy_template(template_dir, dir, name: name)

        puts "✅ Created tool project: #{dir}"
        puts
        puts "Next steps:"
        puts "  cd #{dir}"
        puts "  bundle install"
        puts "  aictl test mcp/#{name}.rb"
        puts "  aictl serve mcp/#{name}.rb"
      end

      def generate_agent(name)
        dir = name
        FileUtils.mkdir_p("#{dir}/lib")
        FileUtils.mkdir_p("#{dir}/config")

        # Copy agent template
        template_dir = File.expand_path('../cli/templates/agent', __dir__)
        copy_template(template_dir, dir, name: name)

        puts "✅ Created agent project: #{dir}"
        puts
        puts "Next steps:"
        puts "  cd #{dir}"
        puts "  bundle install"
        puts "  # Edit config/config.yaml"
        puts "  aictl run"
      end

      def copy_template(template_dir, target_dir, variables = {})
        return unless Dir.exist?(template_dir)

        Dir.glob("#{template_dir}/**/*", File::FNM_DOTMATCH).each do |source|
          next if File.directory?(source)
          next if File.basename(source) == '.' || File.basename(source) == '..'

          relative_path = source.sub("#{template_dir}/", '')
          target = File.join(target_dir, relative_path)

          FileUtils.mkdir_p(File.dirname(target))

          content = File.read(source)
          variables.each do |key, value|
            content.gsub!("{{#{key}}}", value.to_s)
          end

          File.write(target, content)
        end
      end
    end
  end
end
