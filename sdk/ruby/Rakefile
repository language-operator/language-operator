# frozen_string_literal: true

require 'bundler/gem_tasks'
require 'rspec/core/rake_task'
require 'rubocop/rake_task'

RSpec::Core::RakeTask.new(:spec) do |t|
  t.exclude_pattern = 'spec/e2e/**/*_spec.rb'
end

RSpec::Core::RakeTask.new(:e2e) do |t|
  t.pattern = 'spec/e2e/**/*_spec.rb'
  t.rspec_opts = '--tag e2e'
end

RuboCop::RakeTask.new

desc 'Run all tests (unit + e2e)'
task test: %i[spec e2e]

task default: %i[spec rubocop]

desc 'Generate YARD documentation'
task :docs do
  sh 'yard doc'
end

desc 'Open console with gem loaded'
task :console do
  require 'irb'
  require 'language_operator'
  ARGV.clear
  IRB.start
end
