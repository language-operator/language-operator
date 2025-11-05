# fish completion for aictl

# Helper functions for dynamic completion
function __aictl_clusters
    aictl cluster list 2>/dev/null | tail -n +2 | awk '{print $1}' | grep -v '^─' | grep -v '^NAME'
end

function __aictl_agents
    aictl agent list 2>/dev/null | tail -n +2 | awk '{print $1}' | grep -v '^─' | grep -v '^NAME'
end

function __aictl_personas
    aictl persona list 2>/dev/null | tail -n +2 | awk '{print $1}' | grep -v '^─' | grep -v '^NAME'
end

function __aictl_tools
    aictl tool list 2>/dev/null | tail -n +2 | awk '{print $1}' | grep -v '^─' | grep -v '^NAME'
end

# Disable file completion by default
complete -c aictl -f

# Top-level commands
complete -c aictl -n "__fish_use_subcommand" -a "cluster" -d "Manage language clusters"
complete -c aictl -n "__fish_use_subcommand" -a "use" -d "Switch to a different cluster context"
complete -c aictl -n "__fish_use_subcommand" -a "agent" -d "Manage autonomous agents"
complete -c aictl -n "__fish_use_subcommand" -a "persona" -d "Manage agent personas"
complete -c aictl -n "__fish_use_subcommand" -a "tool" -d "Manage MCP tools"
complete -c aictl -n "__fish_use_subcommand" -a "status" -d "Show system status and overview"
complete -c aictl -n "__fish_use_subcommand" -a "version" -d "Show aictl and operator version"
complete -c aictl -n "__fish_use_subcommand" -a "new" -d "Generate a new tool or agent project"
complete -c aictl -n "__fish_use_subcommand" -a "serve" -d "Start an MCP server for tools"
complete -c aictl -n "__fish_use_subcommand" -a "test" -d "Test tool definitions"
complete -c aictl -n "__fish_use_subcommand" -a "run" -d "Run an agent"
complete -c aictl -n "__fish_use_subcommand" -a "console" -d "Start an interactive Ruby console"
complete -c aictl -n "__fish_use_subcommand" -a "help" -d "Show help"

# cluster subcommands
complete -c aictl -n "__fish_seen_subcommand_from cluster" -a "create" -d "Create a new language cluster"
complete -c aictl -n "__fish_seen_subcommand_from cluster" -a "list" -d "List all language clusters"
complete -c aictl -n "__fish_seen_subcommand_from cluster" -a "current" -d "Show current cluster context"
complete -c aictl -n "__fish_seen_subcommand_from cluster" -a "inspect" -d "Show detailed cluster information"
complete -c aictl -n "__fish_seen_subcommand_from cluster" -a "delete" -d "Delete a language cluster"

# cluster inspect/delete - complete with cluster names
complete -c aictl -n "__fish_seen_subcommand_from cluster; and __fish_seen_subcommand_from inspect delete" -a "(__aictl_clusters)"

# use command - complete with cluster names
complete -c aictl -n "__fish_seen_subcommand_from use" -a "(__aictl_clusters)"

# agent subcommands
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "create" -d "Create a new autonomous agent"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "list" -d "List agents in current cluster"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "inspect" -d "Show detailed agent information"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "delete" -d "Delete an agent"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "logs" -d "View agent execution logs"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "code" -d "Display synthesized agent code"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "edit" -d "Edit agent instructions"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "pause" -d "Pause scheduled agent execution"
complete -c aictl -n "__fish_seen_subcommand_from agent" -a "resume" -d "Resume paused agent"

# agent create options
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from create" -l cluster -d "Override current cluster context" -a "(__aictl_clusters)"
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from create" -l create-cluster -d "Create cluster inline"
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from create" -l persona -d "Use specific persona" -a "(__aictl_personas)"
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from create" -l dry-run -d "Preview without creating"

# agent list options
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from list" -l all-clusters -d "Show agents from all clusters"
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from list" -l cluster -d "Show agents from specific cluster" -a "(__aictl_clusters)"

# agent inspect/delete/logs/code/edit/pause/resume - complete with agent names
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from inspect delete logs code edit pause resume" -a "(__aictl_agents)"

# agent logs options
complete -c aictl -n "__fish_seen_subcommand_from agent; and __fish_seen_subcommand_from logs" -s f -l follow -d "Follow log output"

# persona subcommands
complete -c aictl -n "__fish_seen_subcommand_from persona" -a "list" -d "List available personas"
complete -c aictl -n "__fish_seen_subcommand_from persona" -a "show" -d "Display full persona details"
complete -c aictl -n "__fish_seen_subcommand_from persona" -a "create" -d "Create a new custom persona"
complete -c aictl -n "__fish_seen_subcommand_from persona" -a "edit" -d "Edit an existing persona"
complete -c aictl -n "__fish_seen_subcommand_from persona" -a "delete" -d "Delete a persona"

# persona create options
complete -c aictl -n "__fish_seen_subcommand_from persona; and __fish_seen_subcommand_from create" -l from -d "Inherit from existing persona" -a "(__aictl_personas)"
complete -c aictl -n "__fish_seen_subcommand_from persona; and __fish_seen_subcommand_from create" -l cluster -d "Override current cluster" -a "(__aictl_clusters)"

# persona show/edit/delete - complete with persona names
complete -c aictl -n "__fish_seen_subcommand_from persona; and __fish_seen_subcommand_from show edit delete" -a "(__aictl_personas)"

# tool subcommands
complete -c aictl -n "__fish_seen_subcommand_from tool" -a "list" -d "List tools in current cluster"
complete -c aictl -n "__fish_seen_subcommand_from tool" -a "install" -d "Install a new MCP tool"
complete -c aictl -n "__fish_seen_subcommand_from tool" -a "auth" -d "Configure tool authentication"
complete -c aictl -n "__fish_seen_subcommand_from tool" -a "test" -d "Test tool connectivity"
complete -c aictl -n "__fish_seen_subcommand_from tool" -a "delete" -d "Delete a tool"

# tool list options
complete -c aictl -n "__fish_seen_subcommand_from tool; and __fish_seen_subcommand_from list" -l cluster -d "Override current cluster" -a "(__aictl_clusters)"

# tool auth/test/delete - complete with tool names
complete -c aictl -n "__fish_seen_subcommand_from tool; and __fish_seen_subcommand_from auth test delete" -a "(__aictl_tools)"

# new command
complete -c aictl -n "__fish_seen_subcommand_from new" -a "tool" -d "Generate a new tool project"
complete -c aictl -n "__fish_seen_subcommand_from new" -a "agent" -d "Generate a new agent project"

# serve options
complete -c aictl -n "__fish_seen_subcommand_from serve" -l port -d "Port to listen on"
complete -c aictl -n "__fish_seen_subcommand_from serve" -l host -d "Host to bind to"

# run options
complete -c aictl -n "__fish_seen_subcommand_from run" -l config -d "Path to configuration file" -r
