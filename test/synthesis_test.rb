#!/usr/bin/env ruby
# frozen_string_literal: true

require 'yaml'
require 'fileutils'
require 'json'
require 'tempfile'

# Test harness for bulk synthesis testing
#
# Usage:
#   ruby test/synthesis_test.rb                    # Run all tests
#   ruby test/synthesis_test.rb financial          # Run specific category
#   ruby test/synthesis_test.rb --validate-only    # Validate metadata only
class SynthesisTestHarness
  INSTRUCTIONS_DIR = File.join(__dir__, 'instructions')
  RESULTS_FILE = File.join(__dir__, 'synthesis_results.json')

  def initialize(filter: nil, validate_only: false)
    @filter = filter
    @validate_only = validate_only
    @results = {
      total: 0,
      passed: 0,
      failed: 0,
      skipped: 0,
      errors: []
    }
  end

  def run
    puts "Synthesis Test Harness"
    puts "=" * 80
    puts

    instruction_files.each do |file|
      process_instruction_file(file)
    end

    print_summary
    save_results
    exit(@results[:failed].positive? ? 1 : 0)
  end

  private

  def instruction_files
    pattern = if @filter
                File.join(INSTRUCTIONS_DIR, @filter, '*.txt')
              else
                File.join(INSTRUCTIONS_DIR, '**', '*.txt')
              end

    Dir.glob(pattern).sort
  end

  def process_instruction_file(file)
    relative_path = file.sub("#{INSTRUCTIONS_DIR}/", '')
    puts "Testing: #{relative_path}"

    begin
      metadata, instructions = parse_instruction_file(file)
      @results[:total] += 1

      # Validate metadata
      validate_metadata(metadata, file)

      if @validate_only
        puts "  ✓ Metadata valid"
        @results[:passed] += 1
        return
      end

      # Run synthesis
      output = run_synthesis(instructions, metadata, file)

      # Validate output
      validate_synthesis_output(output, metadata, file)

      puts "  ✓ Synthesis successful"
      @results[:passed] += 1

    rescue StandardError => e
      puts "  ✗ FAILED: #{e.message}"
      @results[:failed] += 1
      @results[:errors] << {
        file: relative_path,
        error: e.message,
        backtrace: e.backtrace&.first(5)
      }
    end

    puts
  end

  def parse_instruction_file(file)
    content = File.read(file)

    # Split YAML frontmatter and instructions
    if content.start_with?('---')
      parts = content.split(/^---\s*$/, 3)
      metadata = YAML.safe_load(parts[1])
      instructions = parts[2].strip
    else
      raise "Missing YAML frontmatter in #{file}"
    end

    [metadata, instructions]
  end

  def validate_metadata(metadata, file)
    required_fields = %w[category persona execution_mode difficulty description]
    required_fields.each do |field|
      raise "Missing required field: #{field}" unless metadata.key?(field)
    end

    # Validate execution_mode
    valid_modes = %w[scheduled event autonomous]
    unless valid_modes.include?(metadata['execution_mode'])
      raise "Invalid execution_mode: #{metadata['execution_mode']}"
    end

    # Validate scheduled mode has schedule
    if metadata['execution_mode'] == 'scheduled' && !metadata['expected_schedule']
      raise "scheduled execution_mode requires expected_schedule"
    end

    # Validate difficulty
    valid_difficulties = %w[easy medium hard]
    unless valid_difficulties.include?(metadata['difficulty'])
      raise "Invalid difficulty: #{metadata['difficulty']}"
    end
  end

  def run_synthesis(instructions, metadata, source_file)
    # Generate output file alongside input file (meeting-notes.txt -> meeting-notes.rb)
    output_file = source_file.sub(/\.txt$/, '.rb')

    # Call the synthesis engine binary
    synthesize_bin = File.join(__dir__, '..', 'src', 'bin', 'synthesize')

    unless File.exist?(synthesize_bin)
      raise "Synthesis binary not found: #{synthesize_bin}. Run 'make build' first."
    end

    # Create temp file for instructions
    temp_instructions = Tempfile.new(['instructions', '.txt'])
    temp_instructions.write(instructions)
    temp_instructions.close

    # Build tools list if specified
    tools_flag = metadata['expected_tools'] ? "-tools=#{metadata['expected_tools'].join(',')}" : ""

    # Run synthesis
    cmd = "#{synthesize_bin} -file=#{temp_instructions.path} #{tools_flag} 2>&1"
    result = `#{cmd}`
    exit_status = $?.exitstatus

    temp_instructions.unlink

    if exit_status != 0
      raise "Synthesis failed (exit #{exit_status}): #{result}"
    end

    # Strip logging output - find the first line starting with "require"
    lines = result.lines
    code_start = lines.index { |line| line.strip.start_with?('require ') || line.strip.start_with?('agent ') }

    if code_start
      result = lines[code_start..-1].join
    end

    # Write output
    File.write(output_file, result)
    result
  end

  def generate_mock_synthesis(instructions, metadata)
    # Generate a mock Ruby DSL output for testing
    <<~RUBY
      # Generated from: #{metadata['description']}
      # Persona: #{metadata['persona']}
      # Execution mode: #{metadata['execution_mode']}

      agent do
        persona "#{metadata['persona']}"

        #{execution_mode_code(metadata)}

        task do
          description "#{metadata['description']}"

          # TODO: Actual synthesis would generate steps from instructions
          # Instructions: #{instructions}

          #{mock_tool_usage(metadata)}
        end
      end
    RUBY
  end

  def execution_mode_code(metadata)
    case metadata['execution_mode']
    when 'scheduled'
      "schedule \"#{metadata['expected_schedule']}\""
    when 'event'
      "on_event :trigger_name"
    when 'autonomous'
      "autonomous iterations: 100"
    end
  end

  def mock_tool_usage(metadata)
    return "" unless metadata['expected_tools']

    metadata['expected_tools'].map do |tool|
      "use_tool :#{tool.tr('-', '_')}"
    end.join("\n    ")
  end

  def validate_synthesis_output(output, metadata, file)
    # Validate that output is valid Ruby
    begin
      # Basic syntax check
      raise "Empty output" if output.strip.empty?
      raise "Output doesn't start with 'require' or 'agent'" unless output.strip.start_with?('require', 'agent')
      raise "Output doesn't contain agent definition" unless output.match?(/agent\s+["'][\w-]+["']\s+do/)
    rescue StandardError => e
      raise "Invalid Ruby output: #{e.message}"
    end

    # Validate execution mode is present (scheduled mode requires schedule)
    if metadata['execution_mode'] == 'scheduled'
      unless output.match?(/schedule\s+["']/)
        raise "Scheduled mode requires schedule definition"
      end
    end

    # Note: We don't validate persona or tools strictly since the synthesis engine
    # might use different naming or might not expose persona in the DSL
  end

  def print_summary
    puts "=" * 80
    puts "Summary"
    puts "=" * 80
    puts "Total:   #{@results[:total]}"
    puts "Passed:  #{@results[:passed]} (#{percentage(@results[:passed])}%)"
    puts "Failed:  #{@results[:failed]} (#{percentage(@results[:failed])}%)"
    puts "Skipped: #{@results[:skipped]} (#{percentage(@results[:skipped])}%)"
    puts

    if @results[:errors].any?
      puts "Errors:"
      @results[:errors].each do |error|
        puts "  - #{error[:file]}: #{error[:error]}"
      end
      puts
    end
  end

  def percentage(count)
    return 0 if @results[:total].zero?

    ((count.to_f / @results[:total]) * 100).round(1)
  end

  def save_results
    File.write(RESULTS_FILE, JSON.pretty_generate(@results))
    puts "Results saved to: #{RESULTS_FILE}"
  end
end

# Parse command line arguments
filter = ARGV.find { |arg| !arg.start_with?('--') }
validate_only = ARGV.include?('--validate-only')

harness = SynthesisTestHarness.new(filter: filter, validate_only: validate_only)
harness.run
