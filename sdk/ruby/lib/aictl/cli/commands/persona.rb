# frozen_string_literal: true

require 'thor'
require 'yaml'
require 'pastel'
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

          puts
          puts "Persona: #{pastel.cyan.bold(name)}"
          puts '═' * 80
          puts

          # Format and display key persona details
          spec = persona.dig('spec') || {}
          puts "#{pastel.bold('Display Name:')} #{spec['displayName']}"
          puts "#{pastel.bold('Tone:')} #{pastel.yellow(spec['tone'])}" if spec['tone']
          puts
          puts "#{pastel.bold('Description:')}"
          puts "  #{spec['description']}"
          puts

          if spec['systemPrompt']
            puts "#{pastel.bold('System Prompt:')}"
            puts "  #{spec['systemPrompt']}"
            puts
          end

          if spec['capabilities']&.any?
            puts "#{pastel.bold('Capabilities:')}"
            spec['capabilities'].each do |cap|
              puts "  #{pastel.green('•')} #{cap}"
            end
            puts
          end

          if spec['toolPreferences']&.any?
            puts "#{pastel.bold('Tool Preferences:')}"
            spec['toolPreferences'].each do |pref|
              puts "  #{pastel.green('•')} #{pref}"
            end
            puts
          end

          if spec['responseFormat']
            puts "#{pastel.bold('Response Format:')} #{spec['responseFormat']}"
            puts
          end

          puts '═' * 80
          puts
          Formatters::ProgressFormatter.info("Use this persona when creating agents:")
          puts "  aictl agent create \"description\" --persona #{name}"
          puts
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to show persona: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'create NAME', 'Create a new persona'
        long_desc <<-DESC
          Create a new persona with the specified name using an interactive wizard.

          Examples:
            aictl persona create helpful-assistant
            aictl persona create code-reviewer --from helpful-assistant
        DESC
        option :cluster, type: :string, desc: 'Override current cluster context'
        option :from, type: :string, desc: 'Copy from existing persona as starting point'
        def create(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Check if persona already exists
          begin
            k8s.get_resource('LanguagePersona', name, cluster_config[:namespace])
            Formatters::ProgressFormatter.error("Persona '#{name}' already exists in cluster '#{cluster}'")
            puts
            puts 'Use a different name or delete the existing persona first:'
            puts "  aictl persona delete #{name}"
            exit 1
          rescue K8s::Error::NotFound
            # Good - persona doesn't exist yet
          end

          # If --from flag provided, copy from existing persona
          base_persona = nil
          if options[:from]
            begin
              base_persona = k8s.get_resource('LanguagePersona', options[:from], cluster_config[:namespace])
              Formatters::ProgressFormatter.info("Copying from persona '#{options[:from]}'")
              puts
            rescue K8s::Error::NotFound
              Formatters::ProgressFormatter.error("Source persona '#{options[:from]}' not found")
              exit 1
            end
          end

          # Interactive prompts
          require 'tty-prompt'
          prompt = TTY::Prompt.new

          puts
          puts '=' * 80
          puts '  Create New Persona'
          puts '=' * 80
          puts

          # Get display name
          default_display_name = base_persona&.dig('spec', 'displayName') || name.split('-').map(&:capitalize).join(' ')
          display_name = prompt.ask('Display Name:', default: default_display_name)

          # Get description
          default_description = base_persona&.dig('spec', 'description') || ''
          description = prompt.ask('Description:', default: default_description) do |q|
            q.required true
          end

          # Get tone
          default_tone = base_persona&.dig('spec', 'tone') || 'neutral'
          tone = prompt.select('Tone:', %w[neutral friendly professional technical creative], default: default_tone)

          # Get system prompt
          puts
          puts 'System Prompt (press Enter to open editor):'
          prompt.keypress('Press any key to continue...')

          default_system_prompt = base_persona&.dig('spec', 'systemPrompt') || ''
          system_prompt = edit_in_editor(default_system_prompt, 'persona-system-prompt')

          if system_prompt.strip.empty?
            Formatters::ProgressFormatter.error('System prompt cannot be empty')
            exit 1
          end

          # Get capabilities
          puts
          puts 'Capabilities (optional, press Enter to open editor, or leave empty to skip):'
          puts 'Describe what this persona can do, one per line.'
          prompt.keypress('Press any key to continue...')

          default_capabilities = base_persona&.dig('spec', 'capabilities')&.join("\n") || ''
          capabilities_text = edit_in_editor(default_capabilities, 'persona-capabilities')
          capabilities = capabilities_text.strip.empty? ? [] : capabilities_text.strip.split("\n").map(&:strip).reject(&:empty?)

          # Build persona resource
          persona_spec = {
            'displayName' => display_name,
            'description' => description,
            'tone' => tone,
            'systemPrompt' => system_prompt.strip
          }
          persona_spec['capabilities'] = capabilities unless capabilities.empty?

          persona_resource = Kubernetes::ResourceBuilder.build_persona(
            name: name,
            spec: persona_spec,
            namespace: cluster_config[:namespace]
          )

          # Show preview
          puts
          puts '=' * 80
          puts 'Preview:'
          puts '=' * 80
          puts YAML.dump(persona_resource)
          puts '=' * 80
          puts

          # Confirm creation
          unless prompt.yes?('Create this persona?')
            puts 'Creation cancelled'
            return
          end

          # Create persona
          Formatters::ProgressFormatter.with_spinner("Creating persona '#{name}'") do
            k8s.create_resource(persona_resource)
          end

          Formatters::ProgressFormatter.success("Persona '#{name}' created successfully")
          puts
          puts 'Use this persona when creating agents:'
          puts "  aictl agent create \"description\" --persona #{name}"
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to create persona: #{e.message}")
          raise if ENV['DEBUG']
          exit 1
        end

        desc 'edit NAME', 'Edit an existing persona'
        long_desc <<-DESC
          Edit an existing persona by opening the YAML definition in your editor.

          When you save and close the editor, the persona will be updated.
          Any agents using this persona will be automatically re-synthesized.

          Example:
            aictl persona edit helpful-assistant
        DESC
        option :cluster, type: :string, desc: 'Override current cluster context'
        def edit(name)
          cluster = ClusterValidator.get_cluster(options[:cluster])
          cluster_config = ClusterValidator.get_cluster_config(cluster)

          k8s = Kubernetes::Client.new(
            kubeconfig: cluster_config[:kubeconfig],
            context: cluster_config[:context]
          )

          # Get current persona
          begin
            persona = k8s.get_resource('LanguagePersona', name, cluster_config[:namespace])
          rescue K8s::Error::NotFound
            Formatters::ProgressFormatter.error("Persona '#{name}' not found in cluster '#{cluster}'")
            exit 1
          end

          # Open in editor
          original_yaml = YAML.dump(persona)
          edited_yaml = edit_in_editor(original_yaml, "persona-#{name}")

          # Check if changed
          if edited_yaml.strip == original_yaml.strip
            Formatters::ProgressFormatter.info('No changes made')
            return
          end

          # Parse edited YAML
          begin
            edited_persona = YAML.safe_load(edited_yaml)
          rescue Psych::SyntaxError => e
            Formatters::ProgressFormatter.error("Invalid YAML: #{e.message}")
            exit 1
          end

          # Validate structure
          unless edited_persona.is_a?(Hash) && edited_persona['spec']
            Formatters::ProgressFormatter.error('Invalid persona structure: missing spec')
            exit 1
          end

          # Update persona
          Formatters::ProgressFormatter.with_spinner("Updating persona '#{name}'") do
            k8s.update_resource(edited_persona)
          end

          Formatters::ProgressFormatter.success("Persona '#{name}' updated successfully")

          # Check for agents using this persona
          agents = k8s.list_resources('LanguageAgent', namespace: cluster_config[:namespace])
          agents_using = agents.select { |a| a.dig('spec', 'persona') == name }

          if agents_using.any?
            puts
            Formatters::ProgressFormatter.info("#{agents_using.count} agent(s) will be re-synthesized automatically:")
            agents_using.each do |agent|
              puts "  - #{agent.dig('metadata', 'name')}"
            end
          end
        rescue StandardError => e
          Formatters::ProgressFormatter.error("Failed to edit persona: #{e.message}")
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

        private

        def edit_in_editor(content, filename_prefix)
          require 'tempfile'

          editor = ENV['EDITOR'] || 'vi'
          tempfile = Tempfile.new([filename_prefix, '.txt'])

          begin
            tempfile.write(content)
            tempfile.flush
            tempfile.close

            system("#{editor} #{tempfile.path}")

            File.read(tempfile.path)
          ensure
            tempfile.unlink
          end
        end

        def pastel
          @pastel ||= Pastel.new
        end
      end
    end
  end
end
