# frozen_string_literal: true

require 'tty-spinner'
require 'pastel'

module LanguageOperator
  module CLI
    module Formatters
      # Beautiful progress output for CLI operations
      class ProgressFormatter
        class << self
          def with_spinner(message, &block)
            spinner = TTY::Spinner.new("[:spinner] #{message}...", format: :dots)
            spinner.auto_spin

            result = block.call
            spinner.success("(#{pastel.green('✓')})")
            result
          rescue StandardError => e
            spinner.error("(#{pastel.red('✗')})")
            raise
          end

          def success(message)
            puts "#{pastel.green('✓')} #{message}"
          end

          def error(message)
            puts "#{pastel.red('✗')} #{message}"
          end

          def info(message)
            puts "#{pastel.blue('ℹ')} #{message}"
          end

          def warn(message)
            puts "#{pastel.yellow('⚠')} #{message}"
          end

          private

          def pastel
            @pastel ||= Pastel.new
          end
        end
      end
    end
  end
end
