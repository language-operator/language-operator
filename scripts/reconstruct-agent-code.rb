#!/usr/bin/env ruby
# Code reconstruction script that combines optimized task code with original agent structure
# This script uses string-based parsing to properly rebuild complete agent DSL files

require 'json'

class AgentCodeReconstructor
  class ReconstructionError < StandardError; end

  # Input structure expected from Go controller
  class Input
    attr_reader :original_code, :optimized_tasks

    def initialize(json_input)
      data = JSON.parse(json_input)
      @original_code = data['original_code']
      @optimized_tasks = data['optimized_tasks'] || {}
    rescue JSON::ParserError => e
      raise ReconstructionError, "Invalid JSON input: #{e.message}"
    end
  end

  def reconstruct(input)
    result_code = input.original_code.dup
    
    # For each optimized task, find and replace its definition
    input.optimized_tasks.each do |task_name, optimized_code|
      result_code = replace_task_definition(result_code, task_name, optimized_code)
    end
    
    result_code
  end

  private

  # Replace a task definition in the agent code
  def replace_task_definition(code, task_name, optimized_code)
    # Pattern for neural tasks: task(:name, instructions: "...", inputs: {...}, outputs: {...})
    # This pattern matches the whole task definition as a single statement
    task_pattern = /^(\s*)task\s*\(\s*:#{Regexp.escape(task_name)}\s*,\s*(.*?)\)\s*$/m
    
    if match = code.match(task_pattern)
      indentation = match[1]
      original_params = match[2].strip
      
      # Build the new symbolic task definition to replace the neural one
      replacement = build_replacement_task(task_name, optimized_code, original_params, indentation)
      
      # Replace the entire task line
      return code.gsub(task_pattern, replacement)
    end
    
    # If we couldn't find the task, insert it as a new task
    insert_new_task(code, task_name, optimized_code)
  end


  # Build the replacement task with proper formatting
  def build_replacement_task(task_name, optimized_code, original_params, indentation)
    # Parse the original parameters to extract inputs and outputs
    inputs_match = original_params.match(/inputs:\s*\{([^}]*)\}/)
    outputs_match = original_params.match(/outputs:\s*\{([^}]*)\}/)
    
    inputs_str = inputs_match ? inputs_match[1].strip : ''
    outputs_str = outputs_match ? outputs_match[1].strip : ''
    
    # Indent the body code properly (relative to the task)
    body_lines = optimized_code.strip.lines
    indented_body = body_lines.map { |line| "#{indentation}  #{line.rstrip}" }.join("\n")
    indented_body += "\n" unless indented_body.end_with?("\n")

    <<~TASK.chomp
      #{indentation}task :#{task_name},
      #{indentation}     inputs: { #{inputs_str} },
      #{indentation}     outputs: { #{outputs_str} } do |inputs|
      #{indented_body}#{indentation}end
    TASK
  end

  # Insert a new task when not found in original code
  def insert_new_task(code, task_name, optimized_code)
    # Find insertion point before main block or final end
    if code =~ /(\n[ \t]*)main\s+do/
      # Insert before main block
      insertion_point = $~.begin(0)
      indentation = '  ' # Standard indentation
      # Use default empty inputs/outputs for new tasks
      new_task = "\n\n#{build_replacement_task(task_name, optimized_code, 'inputs: {}, outputs: {}', indentation)}"
      code[insertion_point, 0] = new_task
      code
    elsif code =~ /\n([ \t]*)end\s*\z/
      # Insert before final end
      insertion_point = $~.begin(0)
      indentation = $1 || '  '
      new_task = "\n\n#{build_replacement_task(task_name, optimized_code, 'inputs: {}, outputs: {}', indentation)}\n"
      code[insertion_point, 0] = new_task
      code
    else
      # Append at end
      indentation = '  '
      new_task = "\n\n#{build_replacement_task(task_name, optimized_code, 'inputs: {}, outputs: {}', indentation)}"
      code + new_task
    end
  end

  def self.reconstruct_agent_code(json_input)
    input = Input.new(json_input)
    reconstructor = new
    reconstructor.reconstruct(input)
  end
end

# Main execution when called from command line
if __FILE__ == $0
  begin
    # Read JSON input from STDIN
    input_json = STDIN.read
    
    # Reconstruct the code
    result = AgentCodeReconstructor.reconstruct_agent_code(input_json)
    
    # Output the reconstructed code
    puts result
    exit 0
    
  rescue AgentCodeReconstructor::ReconstructionError => e
    STDERR.puts "Reconstruction error: #{e.message}"
    exit 1
  rescue StandardError => e
    STDERR.puts "Unexpected error: #{e.message}"
    STDERR.puts e.backtrace.join("\n")
    exit 1
  end
end