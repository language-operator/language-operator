# frozen_string_literal: true

require_relative 'dsl'

module LanguageOperator
  # Loads tool definitions from Ruby files
  #
  # Scans a directory for Ruby files containing tool definitions and loads them
  # into a registry. Provides hot-reloading capability for development.
  #
  # @example Basic usage
  #   registry = LanguageOperator::Dsl::Registry.new
  #   loader = LanguageOperator::ToolLoader.new(registry, '/mcp/tools')
  #   loader.load_tools
  #   puts "Loaded #{registry.all.length} tools"
  #
  # @example With custom context
  #   loader = LanguageOperator::ToolLoader.new(registry)
  #   loader.load_tools
  #   loader.reload  # Hot reload tools
  class ToolLoader
    # Initialize tool loader
    #
    # @param registry [LanguageOperator::Dsl::Registry] Tool registry
    # @param tools_dir [String] Directory containing tool definition files
    def initialize(registry, tools_dir = '/mcp')
      @registry = registry
      @tools_dir = tools_dir
    end

    # Load all tool files from the tools directory
    #
    # @return [void]
    def load_tools
      @registry.clear

      unless Dir.exist?(@tools_dir)
        puts "Tools directory #{@tools_dir} does not exist. Skipping tool loading."
        return
      end

      tool_files = Dir.glob(File.join(@tools_dir, '**', '*.rb'))

      if tool_files.empty?
        puts "No tool files found in #{@tools_dir}"
        return
      end

      tool_files.each do |file|
        load_tool_file(file)
      end

      puts "Loaded #{@registry.all.length} tools from #{tool_files.length} files"
    end

    # Load a single tool file
    #
    # @param file [String] Path to tool definition file
    # @return [void]
    def load_tool_file(file)
      puts "Loading tools from: #{file}"

      begin
        context = LanguageOperator::Dsl::Context.new(@registry)
        code = File.read(file)
        context.instance_eval(code, file)
      rescue StandardError => e
        warn "Error loading tool file #{file}: #{e.message}"
        warn e.backtrace.join("\n")
      end
    end

    # Reload all tools (hot reload)
    #
    # @return [void]
    def reload
      puts "Reloading tools..."
      load_tools
    end
  end
end
