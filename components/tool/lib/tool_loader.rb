require_relative 'based/dsl'

# Loads tool definitions from Ruby files
class ToolLoader
  def initialize(registry, tools_dir = '/mcp')
    @registry = registry
    @tools_dir = tools_dir
  end

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

  def load_tool_file(file)
    puts "Loading tools from: #{file}"

    begin
      context = Based::Dsl::Context.new(@registry)
      code = File.read(file)
      context.instance_eval(code, file)
    rescue StandardError => e
      warn "Error loading tool file #{file}: #{e.message}"
      warn e.backtrace.join("\n")
    end
  end

  def reload
    puts "Reloading tools..."
    load_tools
  end
end
