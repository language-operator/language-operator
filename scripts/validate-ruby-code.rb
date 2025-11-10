#!/usr/bin/env ruby
# Wrapper script that calls the AST validator from the language-operator gem
# This script is used by the Go synthesizer for security validation

require 'bundler/setup'
require 'language_operator/agent/safety/ast_validator'

# Suppress parser warnings to STDERR so they don't interfere with JSON output
$VERBOSE = nil

# Read code from STDIN
code = STDIN.read

# WORKAROUND: Strip the require 'language_operator' line before validation
# The synthesizer requires this line but the validator blocks all requires
# See: https://git.theryans.io/language-operator/language-operator-gem/issues/41
code_to_validate = code.gsub(/^require\s+['"]language_operator['"]\s*$/, '')

# Create validator and validate code
# Redirect STDERR to /dev/null to suppress parser version warnings
validator = LanguageOperator::Agent::Safety::ASTValidator.new
violations = nil
stderr_original = $stderr.dup
begin
  $stderr.reopen('/dev/null', 'w')
  violations = validator.validate(code_to_validate)
ensure
  $stderr.reopen(stderr_original)
  stderr_original.close
end

# Output violations as JSON
require 'json'
puts JSON.generate(violations)

# Exit with appropriate code
exit(violations.empty? ? 0 : 1)
