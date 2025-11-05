# frozen_string_literal: true

require 'tty-table'
require 'pastel'

module Aictl
  module CLI
    module Formatters
      # Table output for CLI list commands
      class TableFormatter
        class << self
          def clusters(clusters)
            return ProgressFormatter.info('No clusters found') if clusters.empty?

            headers = ['NAME', 'NAMESPACE', 'AGENTS', 'TOOLS', 'MODELS', 'STATUS']
            rows = clusters.map do |cluster|
              [
                cluster[:name],
                cluster[:namespace],
                cluster[:agents] || 0,
                cluster[:tools] || 0,
                cluster[:models] || 0,
                status_indicator(cluster[:status])
              ]
            end

            table = TTY::Table.new(headers, rows)
            puts table.render(:unicode, padding: [0, 1])
          end

          def agents(agents)
            return ProgressFormatter.info('No agents found') if agents.empty?

            headers = ['NAME', 'MODE', 'STATUS', 'NEXT RUN', 'EXECUTIONS']
            rows = agents.map do |agent|
              [
                agent[:name],
                agent[:mode],
                status_indicator(agent[:status]),
                agent[:next_run] || 'N/A',
                agent[:executions] || 0
              ]
            end

            table = TTY::Table.new(headers, rows)
            puts table.render(:unicode, padding: [0, 1])
          end

          def tools(tools)
            return ProgressFormatter.info('No tools found') if tools.empty?

            headers = ['NAME', 'TYPE', 'STATUS', 'AGENTS USING']
            rows = tools.map do |tool|
              [
                tool[:name],
                tool[:type],
                status_indicator(tool[:status]),
                tool[:agents_using] || 0
              ]
            end

            table = TTY::Table.new(headers, rows)
            puts table.render(:unicode, padding: [0, 1])
          end

          def personas(personas)
            return ProgressFormatter.info('No personas found') if personas.empty?

            headers = ['NAME', 'TONE', 'USED BY', 'DESCRIPTION']
            rows = personas.map do |persona|
              [
                persona[:name],
                persona[:tone],
                persona[:used_by] || 0,
                truncate(persona[:description], 50)
              ]
            end

            table = TTY::Table.new(headers, rows)
            puts table.render(:unicode, padding: [0, 1])
          end

          def models(models)
            return ProgressFormatter.info('No models found') if models.empty?

            headers = ['NAME', 'PROVIDER', 'MODEL', 'STATUS']
            rows = models.map do |model|
              [
                model[:name],
                model[:provider],
                model[:model],
                status_indicator(model[:status])
              ]
            end

            table = TTY::Table.new(headers, rows)
            puts table.render(:unicode, padding: [0, 1])
          end

          private

          def status_indicator(status)
            case status&.downcase
            when 'ready', 'running', 'active'
              "#{pastel.green('●')} #{status}"
            when 'pending', 'creating', 'synthesizing'
              "#{pastel.yellow('●')} #{status}"
            when 'failed', 'error'
              "#{pastel.red('●')} #{status}"
            when 'paused', 'stopped'
              "#{pastel.dim('●')} #{status}"
            else
              "#{pastel.dim('●')} #{status || 'Unknown'}"
            end
          end

          def truncate(text, length)
            return text if text.nil? || text.length <= length

            "#{text[0...length - 3]}..."
          end

          def pastel
            @pastel ||= Pastel.new
          end
        end
      end
    end
  end
end
