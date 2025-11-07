# Natural Language Agent Creation

## Vision

Enable users to create autonomous agents using natural language descriptions, inspired by how accountants, lawyers, analysts, and other professionals describe their repetitive tasks.

## User Experience

### Target Workflow

```bash
# User describes their workflow in natural language
langop create agent "review my recent changes in https://docs.google.com/spreadsheets/d/xyz around 4pm every day, and let me know if I've made any mistakes before I sign off for the day"

# System responds with:
Creating agent from natural language...
✓ Parsed task: Review spreadsheet changes
✓ Detected persona: Financial Analyst (detail-oriented, error-checking)
✓ Configured schedule: Daily at 4:00 PM
✓ Connected data source: Google Sheets (xyz)
✓ Output destination: Email notification

Agent 'spreadsheet-reviewer' created successfully!

Next execution: Today at 4:00 PM
View logs: langop logs spreadsheet-reviewer
Edit agent: langop edit spreadsheet-reviewer
```

### Additional Examples

```bash
# Legal assistant
langop create agent "every Monday morning, scan my inbox for new client emails and draft initial response templates based on my previous responses"

# Customer support
langop create agent "when someone mentions 'refund' in our #support channel, check if they're a premium customer and if so, auto-approve refunds under $50"

# DevOps
langop create agent "check our prod error rate every 15 minutes and page me if it spikes above 1%"

# Executive assistant
langop create agent "review my calendar each evening and email me a summary of tomorrow's meetings with relevant background docs"
```

## Architecture

### Phase 1: Natural Language Parsing (LLM-Based)

The `langop create agent` command uses an LLM to parse natural language into structured agent configuration.

#### Input Processing Flow

```
User Input (Natural Language)
    ↓
LLM Parser (with schema constraints)
    ↓
Structured Agent Spec
    ↓
LanguagePersona creation (if custom persona needed)
    ↓
LanguageAgent creation
    ↓
Kubernetes deployment
```

#### LLM Parser Schema

The parser extracts:

1. **Task Description** - What the agent does
2. **Persona Requirements** - Role, tone, expertise needed
3. **Data Sources** - URLs, files, APIs, databases
4. **Schedule/Triggers** - When it runs (cron, event-based, continuous)
5. **Output Destinations** - Where results go (email, slack, file, webhook)
6. **Tools Required** - MCP tools needed (web, email, sheets, etc.)
7. **Constraints** - Limits, permissions, approval requirements

#### Parser Prompt Template

```yaml
system_prompt: |
  You are a langop agent configuration parser. Parse natural language agent descriptions
  into structured YAML configuration for Kubernetes LanguageAgent and LanguagePersona resources.

  Extract the following:
  - task: The core task the agent performs
  - persona: The professional role and communication style
  - schedule: When the agent runs (cron expression or trigger)
  - dataSources: External data the agent needs access to
  - outputs: Where the agent sends results
  - tools: MCP tools required
  - constraints: Limits, approval requirements, permissions

user_prompt: |
  Parse this agent request:

  "{{user_input}}"

  Respond with YAML following the langop agent schema.
```

### Phase 2: Persona Library

Pre-built personas for common job functions:

```bash
langop persona list
# Shows:
# - financial-analyst: Detail-oriented number cruncher
# - legal-assistant: Formal, precise, citation-focused
# - customer-support: Empathetic, solution-oriented
# - devops-engineer: Alert-focused, action-oriented
# - executive-assistant: Organized, proactive, diplomatic

langop persona show financial-analyst
# Shows full YAML spec

langop persona create my-accountant --from financial-analyst
# Creates custom persona you can edit
```

Pre-built personas ship with the operator as ConfigMaps:

```
chart/persona-library/
├── financial-analyst.yaml
├── legal-assistant.yaml
├── customer-support.yaml
├── devops-engineer.yaml
└── executive-assistant.yaml
```

### Phase 3: Smart Tool Discovery

Automatically detect required tools from natural language:

| User Says | Tools Detected |
|-----------|---------------|
| "review my spreadsheet" | `google-sheets`, `mcp-gdrive` |
| "scan my inbox" | `email`, `mcp-gmail` |
| "check our Slack" | `slack`, `mcp-slack` |
| "page me" | `pagerduty`, `email` |
| "when someone tweets" | `twitter`, `mcp-twitter` |

```bash
# Tool detection during creation
langop create agent "review my spreadsheet..."

# If tools not installed:
⚠ Required tools not found:
  - google-sheets (for spreadsheet access)

Install now? [Y/n]: y

Installing google-sheets...
✓ google-sheets installed
✓ Credentials required: Run 'aictl tools auth google-sheets'
```

### Phase 4: Interactive Refinement

If the natural language is ambiguous, ask clarifying questions:

```bash
langop create agent "review my documents daily"

? Which documents should I review?
  1. All files in Google Drive
  2. Specific folder (specify path)
  3. Files matching pattern
  4. Documents you specify manually

> 2

? What folder path?
> /Accounting/Q4-2025

? What should I look for when reviewing?
> (free text) Check for missing receipts and flag expenses over $500

? How should I notify you?
  1. Email
  2. Slack DM
  3. Slack channel
  4. Write to file

> 1

Creating agent...
✓ Agent 'document-reviewer' created
```

## Implementation Plan

### Stage 1: Basic NLP Parser (MVP)

**Components:**
- New CLI command: `langop create agent <description>`
- LLM-based parser (using Anthropic API)
- YAML template generator
- kubectl apply wrapper

**File Structure:**
```
sdk/ruby/lib/langop/cli/
├── commands/
│   └── create_agent.rb      # Main command handler
├── parsers/
│   └── nlp_parser.rb         # LLM-based natural language parser
├── generators/
│   └── agent_generator.rb   # Generates YAML from parsed spec
└── templates/
    ├── agent.yaml.erb        # LanguageAgent template
    └── persona.yaml.erb      # LanguagePersona template
```

**Code Sketch:**

```ruby
module Langop
  module CLI
    module Commands
      class CreateAgent < Thor::Group
        argument :description

        def parse_description
          parser = Langop::Parsers::NLPParser.new
          @spec = parser.parse(description)
        end

        def generate_persona
          if @spec.needs_custom_persona?
            generator = Langop::Generators::PersonaGenerator.new(@spec)
            @persona_yaml = generator.generate
          end
        end

        def generate_agent
          generator = Langop::Generators::AgentGenerator.new(@spec)
          @agent_yaml = generator.generate
        end

        def deploy
          kubectl_apply(@persona_yaml) if @persona_yaml
          kubectl_apply(@agent_yaml)

          puts "✓ Agent '#{@spec.name}' created successfully!"
          puts "Next execution: #{@spec.next_run_time}"
        end
      end
    end
  end
end
```

### Stage 2: Persona Library

**Components:**
- Pre-built persona YAML files
- `langop persona` subcommands
- Persona inheritance system

**CLI Commands:**
```bash
langop persona list                    # List available personas
langop persona show <name>             # Show persona details
langop persona create <name>           # Create custom persona
langop persona edit <name>             # Edit existing persona
langop persona delete <name>           # Delete persona
```

### Stage 3: Tool Integration

**Components:**
- Tool discovery from natural language
- Tool installation prompts
- Tool authentication flows

**Tool Registry:**
```yaml
# sdk/ruby/lib/langop/tools/registry.yaml
tools:
  google-sheets:
    patterns: ["spreadsheet", "sheet", "excel", "csv"]
    mcp_server: "mcp-gdrive"
    requires_auth: true
    auth_type: "oauth2"

  email:
    patterns: ["email", "inbox", "mail"]
    mcp_server: "mcp-gmail"
    requires_auth: true
    auth_type: "oauth2"

  slack:
    patterns: ["slack", "channel", "dm", "message"]
    mcp_server: "mcp-slack"
    requires_auth: true
    auth_type: "api_key"
```

### Stage 4: Interactive Mode

**Components:**
- Wizard-style prompts for ambiguous requests
- TTY detection for interactive vs non-interactive
- Smart defaults based on user history

## Testing Strategy

### Unit Tests

```ruby
# sdk/ruby/spec/langop/parsers/nlp_parser_spec.rb
describe Langop::Parsers::NLPParser do
  it "parses accountant workflow" do
    input = "review my spreadsheet at 4pm daily"
    spec = described_class.new.parse(input)

    expect(spec.task).to eq("review spreadsheet")
    expect(spec.schedule).to eq("0 16 * * *")
    expect(spec.tools).to include("google-sheets")
  end
end
```

### Integration Tests

```bash
# Test full workflow
langop create agent "test agent that runs every minute" --dry-run
# Should output YAML without deploying

langop create agent "test agent that runs every minute" --apply
# Should create resources in cluster

kubectl get languageagent test-agent
# Should show created agent
```

### User Acceptance Tests

Real-world scenarios:
1. Accountant spreadsheet review
2. Legal document scanning
3. DevOps error monitoring
4. Executive calendar summary
5. Customer support automation

## Success Metrics

### MVP Success Criteria

1. **Parse accuracy**: 90%+ of common natural language patterns correctly parsed
2. **Time to deploy**: < 60 seconds from command to deployed agent
3. **User satisfaction**: Users can create agents without reading docs
4. **Tool coverage**: Support 10+ common tools (email, slack, sheets, calendar, etc.)

### Advanced Success Criteria

1. **Persona reuse**: 80%+ of agents use pre-built personas
2. **Tool auto-detection**: 95%+ accuracy on tool requirements
3. **Natural refinement**: Users prefer interactive wizard over manual YAML editing
4. **Production usage**: 100+ agents deployed in real workflows

## Examples Library

Ship with real-world examples users can copy:

```bash
langop examples list
# Shows:
# 1. daily-spreadsheet-review - Accountant workflow
# 2. inbox-triage - Email management
# 3. error-monitoring - DevOps alerts
# 4. meeting-prep - Executive assistant
# 5. support-ticket-routing - Customer support

langop examples show daily-spreadsheet-review
# Shows full description and YAML

langop examples create daily-spreadsheet-review --interactive
# Walks through customizing the example
```

## Future Enhancements

### Voice Input
```bash
langop create agent --voice
# Opens microphone, transcribes to text, creates agent
```

### Multi-Agent Workflows
```bash
langop create workflow "when sales team closes a deal, notify legal to draft contract, then notify accounting to set up billing"
# Creates orchestrated multi-agent workflow
```

### Agent Marketplace
```bash
langop marketplace search "accounting"
# Shows community-built accounting agents

langop marketplace install @acme/quarterly-report-generator
# Installs pre-built agent from marketplace
```

### Learning from Feedback
```bash
langop feedback agent-name "this should run at 5pm not 4pm"
# LLM learns from feedback, improves future parsing
```

## Open Questions

1. **Privacy**: How do we handle sensitive data in natural language descriptions?
2. **Multi-tenancy**: How do we isolate agents across different users/teams?
3. **Cost**: LLM parsing costs - cache common patterns?
4. **Validation**: How do we validate generated configs before deployment?
5. **Rollback**: Easy way to undo agent creation if it's wrong?

## Related Documents

- [requirements/tasks/optimize.md](../tasks/optimize.md) - Performance optimization requirements
- [requirements/ARCHITECTURE.md](../ARCHITECTURE.md) - Overall system architecture
- [chart/crds/](../../chart/crds/) - CRD specifications
