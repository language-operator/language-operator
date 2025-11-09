#!/usr/bin/env ruby
# frozen_string_literal: true

# Wrapper script for AST-based Ruby code validation
# Called by Go operator to validate synthesized code
#
# Usage: ruby scripts/validate-ruby-code.rb < code.rb
# Output: JSON array of violations
# Exit: 0 if valid, 1 if violations found, 2 if error

require 'json'

begin
  require 'language_operator/agent/safety/ast_validator'
rescue LoadError => e
  STDERR.puts "Error: language_operator gem not found"
  STDERR.puts e.message
  exit 2
end

# Read code from STDIN
code = STDIN.read

# Validate using AST validator
validator = LanguageOperator::Agent::Safety::ASTValidator.new
violations = validator.validate(code, '(synthesized)')

# Output violations as JSON
puts violations.to_json

# Exit with appropriate code
exit(violations.empty? ? 0 : 1)
