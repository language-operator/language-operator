#!/usr/bin/env ruby
# Wrapper script that formats Ruby code using RuboCop auto-correct
# This script is used by the Go synthesizer for code formatting

require 'tempfile'
require 'json'

# Read code from STDIN
code = STDIN.read

# Create temporary file for RuboCop processing
Tempfile.create(['agent', '.rb']) do |file|
  file.write(code)
  file.flush

  # Run RuboCop with auto-correct, focusing on layout issues
  # Add gem bin to PATH
  gem_bin_path = File.join(ENV['HOME'] || '~', '.local/share/gem/ruby/3.4.0/bin')
  system_path = ENV['PATH'] || ''
  ENV['PATH'] = "#{gem_bin_path}:#{system_path}"
  
  # --autocorrect: fix formatting issues automatically
  # --only: focus only on key layout cops that fix the issues we see
  rubocop_cmd = [
    'rubocop',
    '--autocorrect',
    '--only', 'Layout/ArgumentAlignment,Layout/MultilineMethodCallIndentation,Layout/HashAlignment,Layout/FirstHashElementIndentation,Layout/FirstParameterIndentation',
    file.path
  ]

  # Execute RuboCop
  result = system(rubocop_cmd.join(' '))
  exit_code = $?.exitstatus

  # Read the corrected file content
  formatted_code = File.read(file.path)
  
  # Output formatted code to stdout
  puts formatted_code
end

# Exit with success (we don't fail on RuboCop warnings)
exit 0