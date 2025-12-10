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

  # Find the task by name (tasks is a hash with symbol or string keys)
  task = agent.tasks[task_name.to_sym] || agent.tasks[task_name]

  if task.nil?
    STDERR.puts "Task '#{task_name}' not found in agent. Available tasks: #{agent.tasks.keys.join(', ')}"
    exit 1
  end

  # Determine task type
  task_type = if task.neural? && task.symbolic?
                "hybrid"
              elsif task.neural?
                "neural"
              elsif task.symbolic?
                "symbolic"
              else
                "unknown"
              end

  # Extract task information
  task_info = {
    name: task.name.to_s,
    instructions: task.instructions_text || "",
    inputs: task.inputs_schema || {},
    outputs: task.outputs_schema || {},
    has_code: task.symbolic?,
    type: task_type
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
