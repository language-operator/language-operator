#!/usr/bin/env ruby
# Schema validation script for generated DSL code
# This script validates Ruby DSL code against the language-operator schema
# Used by the Go synthesizer to ensure generated code conforms to the DSL

require 'bundler/setup'
require 'language_operator'
require 'json'
require 'tempfile'

# Suppress parser warnings
$VERBOSE = nil

# Helper method to extract line number from error messages or backtrace
def extract_line_number(text)
  return 1 unless text

  # Try to extract line number from various formats
  match = text.match(/line (\d+)/) || text.match(/:(\d+):/)
  match ? match[1].to_i : 1
end

# Helper method to clean error messages by removing temp file paths
def clean_error_message(message)
  message.gsub(/\/tmp\/agent-validation\d+-\d+-\w+\.rb/, '<generated code>')
end

# Read code from STDIN
code = STDIN.read

violations = []

begin
  # Write code to a temporary file
  # This is necessary because load_agent_file expects a file path
  tempfile = Tempfile.new(['agent-validation', '.rb'])
  tempfile.write(code)
  tempfile.close

  # Redirect stderr to capture any error output
  stderr_original = $stderr.dup
  stderr_capture = StringIO.new

  begin
    $stderr = stderr_capture

    # Use the gem's built-in loader which validates the DSL
    # This will raise errors for schema violations
    LanguageOperator::Dsl.load_agent_file(tempfile.path)

  ensure
    $stderr = stderr_original
    tempfile.unlink
  end

  # If we got here, the code is valid
  # No violations to report

rescue SyntaxError => e
  violations << {
    type: 'syntax_error',
    location: extract_line_number(e.message),
    message: "Syntax error: #{clean_error_message(e.message)}"
  }

rescue NoMethodError => e
  # NoMethodError typically means invalid DSL property or method
  violations << {
    type: 'schema_violation',
    location: extract_line_number(e.backtrace&.first),
    message: "Invalid DSL property or method: #{clean_error_message(e.message)}"
  }

rescue ArgumentError => e
  violations << {
    type: 'schema_violation',
    location: extract_line_number(e.backtrace&.first),
    message: "Schema violation: #{clean_error_message(e.message)}"
  }

rescue StandardError => e
  violations << {
    type: 'validation_error',
    location: extract_line_number(e.backtrace&.first),
    message: "Validation error: #{clean_error_message(e.message)}"
  }
ensure
  # Clean up tempfile if it still exists
  if tempfile && tempfile.path && File.exist?(tempfile.path)
    tempfile.unlink
  end
end

# Output violations as JSON
puts JSON.generate(violations)

# Exit with appropriate code
exit(violations.empty? ? 0 : 1)
