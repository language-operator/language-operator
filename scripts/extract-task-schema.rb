#!/usr/bin/env ruby
# Task schema extraction script
# Extracts task definitions (inputs, outputs, code) from agent DSL code
# Used by the learning controller to provide context for optimization

require 'language_operator'
require 'json'
require 'tempfile'

# Suppress parser warnings
$VERBOSE = nil

# Read task name and code from arguments
if ARGV.length < 2
  STDERR.puts "Usage: extract-task-schema.rb <task_name> <agent_code_file>"
  exit 1
end

task_name = ARGV[0]
agent_code_file = ARGV[1]

begin
  # Load the agent DSL - returns an AgentRegistry
  registry = LanguageOperator::Dsl.load_agent_file(agent_code_file)

  # Get the first (and should be only) agent from the registry
  agent = registry.all.first

  if agent.nil?
    STDERR.puts "No agent found in file"
    exit 1
  end

  # Find the task by name
  task = agent.tasks.find { |t| t.name == task_name || t.name.to_s == task_name }

  if task.nil?
    STDERR.puts "Task '#{task_name}' not found in agent"
    exit 1
  end

  # Extract task information
  task_info = {
    name: task.name,
    instructions: task.instructions || "",
    inputs: task.inputs || {},
    outputs: task.outputs || {},
    code: task.code || "",
    type: task.type || "neural" # neural or symbolic
  }

  # Output as JSON
  puts JSON.generate(task_info)
  exit 0

rescue SyntaxError => e
  STDERR.puts "Syntax error in agent code: #{e.message}"
  exit 1

rescue StandardError => e
  STDERR.puts "Error extracting task schema: #{e.message}"
  exit 1
end
