# Man page tools for MCP

tool "man" do
  description "Retrieve and display a manual page"

  parameter "page" do
    type :string
    required true
    description "The name of the manual page to retrieve (e.g., 'ls', 'grep', 'bash')"
  end

  parameter "section" do
    type :number
    required false
    description "The manual section number (1-8). If not specified, shows the first matching page."
  end

  execute do |params|
    page = params["page"]
    section = params["section"]

    # Build the man command
    cmd = if section
            "man #{section.to_i} #{page} 2>&1"
          else
            "man #{page} 2>&1"
          end

    # Execute and capture output
    output = `#{cmd}`

    if $?.success?
      output
    else
      "Error: No manual entry found for '#{page}'" + (section ? " in section #{section}" : "")
    end
  end
end

tool "man_search" do
  description "Search for manual pages containing a keyword in their name or description"

  parameter "keyword" do
    type :string
    required true
    description "The keyword to search for in man page names and descriptions"
  end

  execute do |params|
    keyword = params["keyword"]

    # Use apropos to search
    output = `apropos #{keyword} 2>&1`

    if $?.success? && !output.strip.empty?
      output
    else
      "No manual pages found matching '#{keyword}'"
    end
  end
end

tool "whatis" do
  description "Display a brief description of a command from its man page"

  parameter "command" do
    type :string
    required true
    description "The command name to look up"
  end

  execute do |params|
    command = params["command"]

    output = `whatis #{command} 2>&1`

    if $?.success?
      output
    else
      "No description found for '#{command}'"
    end
  end
end

tool "man_sections" do
  description "List all available manual sections and their descriptions"

  execute do |params|
    <<~SECTIONS
      Manual Page Sections:

      1 - User Commands: Executable programs or shell commands
      2 - System Calls: Functions provided by the kernel
      3 - Library Calls: Functions within program libraries
      4 - Special Files: Usually devices found in /dev
      5 - File Formats: Configuration files and structures
      6 - Games: Games and entertainment programs
      7 - Miscellaneous: Macro packages, conventions, protocols
      8 - System Administration: Commands usually run by root

      Usage: Use the 'man' tool with the 'section' parameter to view a specific section.
      Example: man(page="printf", section=3) for the C library function printf
    SECTIONS
  end
end
