# Natural Language Agents - MVP Implementation Roadmap

## Vision

"I want to automate my job by describing what I do, not by writing YAML."

Transform langop from a Kubernetes operator toolkit into a natural language interface for autonomous work automation.

## MVP Scope

**Core User Story:**
```bash
langop create agent "review my spreadsheet at https://docs.google.com/... at 4pm daily and email me any errors"
```

**What This Must Do:**
1. Parse natural language into agent configuration
2. Create appropriate LanguagePersona (or use default)
3. Create LanguageAgent with schedule
4. Deploy to Kubernetes cluster
5. Return success confirmation with next execution time

## Implementation Phases

### Phase 0: Foundation (Week 1)

**Goal:** Set up LLM integration in SDK and CLI parser infrastructure

**Tasks:**

1. Add Anthropic SDK dependency to langop gem
2. Create `Langop::Parsers::NLPParser` class
3. Create structured output schema for agent specs
4. Build basic CLI command `langop create agent`

**Files to Create:**
```
sdk/ruby/lib/langop/parsers/
├── nlp_parser.rb              # LLM-based parser
└── agent_spec.rb              # Structured output schema

sdk/ruby/lib/langop/cli/commands/
└── create_agent.rb            # CLI command

sdk/ruby/spec/langop/parsers/
└── nlp_parser_spec.rb         # Tests
```

**Acceptance Criteria:**
- [ ] `langop create agent "test" --dry-run` prints parsed structure
- [ ] Parser extracts task, schedule, tools, output destination
- [ ] Test coverage > 80%

### Phase 1: Basic Agent Generation (Week 2)

**Goal:** Generate valid LanguageAgent YAML from parsed specs

**Tasks:**

1. Create YAML template generator
2. Map parsed specs to LanguageAgent CRD fields
3. Support cron schedule generation from natural language
4. Handle kubectl apply for deployment

**Files to Create:**
```
sdk/ruby/lib/langop/generators/
├── agent_generator.rb         # YAML generator
└── schedule_parser.rb         # Cron from NL

sdk/ruby/lib/langop/templates/
└── language_agent.yaml.erb    # Agent template

sdk/ruby/lib/langop/deployers/
└── kubectl_deployer.rb        # kubectl wrapper
```

**Acceptance Criteria:**
- [ ] Generated YAML passes kubectl validation
- [ ] Schedule correctly parsed ("4pm daily" → "0 16 * * *")
- [ ] Agent deploys to cluster successfully
- [ ] `langop create agent "run every hour" --apply` works end-to-end

### Phase 2: Persona Integration (Week 3)

**Goal:** Auto-generate or select personas based on task description

**Tasks:**

1. Create default persona library
2. Build persona selector (matches task to persona)
3. Generate custom personas when needed
4. Link LanguageAgent to LanguagePersona

**Files to Create:**
```
kubernetes/charts/language-operator/persona-library/
├── financial-analyst.yaml
├── general-assistant.yaml
├── devops-engineer.yaml
└── customer-support.yaml

sdk/ruby/lib/langop/generators/
├── persona_generator.rb       # Persona YAML generator
└── persona_selector.rb        # Match task to persona

sdk/ruby/lib/langop/templates/
└── language_persona.yaml.erb  # Persona template
```

**Acceptance Criteria:**
- [ ] "review spreadsheet" → financial-analyst persona
- [ ] "monitor errors" → devops-engineer persona
- [ ] Generic tasks → general-assistant persona
- [ ] Custom persona created for specialized tasks

### Phase 3: Tool Discovery (Week 4)

**Goal:** Automatically detect and connect required MCP tools

**Tasks:**

1. Build tool pattern matching system
2. Create tool registry mapping keywords to MCP servers
3. Add tool availability checking
4. Prompt for missing tools

**Files to Create:**
```
sdk/ruby/lib/langop/tools/
├── tool_registry.rb           # Tool pattern matching
├── tool_detector.rb           # Detect from NL
└── tool_installer.rb          # Install missing tools

sdk/ruby/config/
└── tool_patterns.yaml         # Keywords → tools mapping
```

**Tool Patterns (Initial):**
```yaml
patterns:
  - keywords: [spreadsheet, sheet, excel, csv, google sheets]
    tool: mcp-gdrive

  - keywords: [email, inbox, gmail, mail]
    tool: mcp-gmail

  - keywords: [slack, channel, message, dm]
    tool: mcp-slack

  - keywords: [web, http, fetch, scrape, url]
    tool: mcp-fetch

  - keywords: [file, read, write, save]
    tool: mcp-filesystem
```

**Acceptance Criteria:**
- [ ] "review spreadsheet" → detects need for mcp-gdrive
- [ ] "check slack" → detects need for mcp-slack
- [ ] Warns if tool not installed
- [ ] Links agent to available LanguageTools

### Phase 4: Output Routing (Week 5)

**Goal:** Route agent outputs to user-specified destinations

**Tasks:**

1. Parse output destinations from NL ("email me", "post to slack")
2. Configure notification tools
3. Template response formatting
4. Handle multiple output channels

**Files to Create:**
```
sdk/ruby/lib/langop/outputs/
├── output_router.rb           # Route outputs
├── email_formatter.rb         # Email templates
└── slack_formatter.rb         # Slack templates

sdk/ruby/lib/langop/templates/outputs/
├── email.erb
└── slack.erb
```

**Acceptance Criteria:**
- [ ] "email me" → configures email notification
- [ ] "post to #alerts" → configures slack channel
- [ ] Supports multiple outputs ("email me and post to slack")
- [ ] Templates render correctly

### Phase 5: End-to-End Polish (Week 6)

**Goal:** Complete MVP with excellent UX

**Tasks:**

1. Add beautiful CLI output with progress indicators
2. Implement `--dry-run` flag for preview
3. Add validation and error handling
4. Create comprehensive error messages
5. Write user documentation

**Files to Create:**
```
sdk/ruby/lib/langop/cli/
├── output_formatter.rb        # Pretty CLI output
└── validator.rb               # Pre-deployment validation

docs/
├── getting-started.md
├── examples/
│   ├── accountant-workflow.md
│   ├── devops-monitoring.md
│   └── email-triage.md
└── troubleshooting.md
```

**Acceptance Criteria:**
- [ ] Beautiful progress output during creation
- [ ] `--dry-run` shows what would be created
- [ ] Clear error messages for all failure modes
- [ ] 5+ documented real-world examples
- [ ] User can go from install to deployed agent in < 5 minutes

## MVP Test Cases

### Test Case 1: Accountant Spreadsheet Review
```bash
langop create agent "review my spreadsheet at https://docs.google.com/spreadsheets/d/abc123 at 4pm daily and email me if there are any errors"

Expected:
✓ Task: Review Google Sheets for errors
✓ Persona: Financial Analyst
✓ Schedule: Daily at 4:00 PM (0 16 * * *)
✓ Tools: mcp-gdrive
✓ Output: Email notification
✓ Agent 'spreadsheet-reviewer' created
```

### Test Case 2: DevOps Error Monitoring
```bash
langop create agent "check https://api.example.com/health every 5 minutes and page me if status isn't 200"

Expected:
✓ Task: HTTP health check monitoring
✓ Persona: DevOps Engineer
✓ Schedule: Every 5 minutes (*/5 * * * *)
✓ Tools: mcp-fetch
✓ Output: PagerDuty alert
✓ Agent 'health-monitor' created
```

### Test Case 3: Email Triage
```bash
langop create agent "scan my inbox every morning at 9am and categorize emails by urgency"

Expected:
✓ Task: Email categorization
✓ Persona: General Assistant
✓ Schedule: Daily at 9:00 AM (0 9 * * *)
✓ Tools: mcp-gmail
✓ Output: Categorized email summary
✓ Agent 'inbox-triage' created
```

### Test Case 4: Slack Monitoring
```bash
langop create agent "when someone mentions 'incident' in #engineering, create a ticket and notify #ops"

Expected:
✓ Task: Slack keyword monitoring
✓ Persona: DevOps Engineer
✓ Schedule: Event-triggered (webhook)
✓ Tools: mcp-slack, ticket-system
✓ Output: Slack notification to #ops
✓ Agent 'incident-responder' created
```

### Test Case 5: Calendar Management
```bash
langop create agent "email me a summary of tomorrow's meetings every evening at 6pm"

Expected:
✓ Task: Calendar summary generation
✓ Persona: Executive Assistant
✓ Schedule: Daily at 6:00 PM (0 18 * * *)
✓ Tools: mcp-google-calendar
✓ Output: Email notification
✓ Agent 'meeting-prep' created
```

## Technical Architecture

### Parser Flow

```
User Input (NL)
    ↓
Anthropic Claude API (structured output)
    ↓
AgentSpec object
    ↓
┌─────────────────────────┐
│ PersonaSelector         │ → LanguagePersona YAML
│ ScheduleParser          │ → Cron expression
│ ToolDetector            │ → Required tools list
│ OutputRouter            │ → Notification config
└─────────────────────────┘
    ↓
AgentGenerator
    ↓
LanguageAgent YAML
    ↓
kubectl apply
    ↓
Running Agent in K8s
```

### Parser Prompt

```
System: You are a langop agent configuration parser. Parse natural language descriptions into structured agent specifications.

Output JSON with this schema:
{
  "name": "suggested agent name (kebab-case)",
  "task": "one sentence task description",
  "task_type": "review|monitor|notify|process|analyze",
  "schedule_type": "cron|event|continuous",
  "schedule": "cron expression or event trigger",
  "data_sources": [
    {"type": "url|file|api|spreadsheet", "location": "..."}
  ],
  "tools_required": ["tool-name", ...],
  "outputs": [
    {"type": "email|slack|file|webhook", "destination": "..."}
  ],
  "persona_role": "accountant|engineer|assistant|analyst",
  "constraints": {
    "max_execution_time": "duration",
    "require_approval": true|false
  }
}

User: <user natural language description>
```

### Data Models

```ruby
# sdk/ruby/lib/langop/parsers/agent_spec.rb
module Langop
  module Parsers
    class AgentSpec
      attr_accessor :name, :task, :task_type, :schedule_type,
                    :schedule, :data_sources, :tools_required,
                    :outputs, :persona_role, :constraints

      def initialize(json)
        # Parse from LLM output
      end

      def to_agent_yaml
        # Generate LanguageAgent YAML
      end

      def to_persona_yaml
        # Generate LanguagePersona YAML if needed
      end
    end
  end
end
```

## Dependencies

### Required Gems
```ruby
# sdk/ruby/langop.gemspec
spec.add_dependency "anthropic", "~> 0.1"     # Anthropic API client
spec.add_dependency "thor", "~> 1.3"          # CLI framework
spec.add_dependency "tty-prompt", "~> 0.23"   # Interactive prompts
spec.add_dependency "tty-spinner", "~> 0.9"   # Progress spinners
spec.add_dependency "tty-table", "~> 0.12"    # Pretty tables
spec.add_dependency "chronic", "~> 0.10"      # Natural language date/time
spec.add_dependency "k8s-ruby", "~> 0.13"     # Kubernetes client
```

### Required Environment Variables
```bash
ANTHROPIC_API_KEY=sk-ant-...    # For NL parsing
KUBECONFIG=~/.kube/config       # For kubectl access
LANGOP_NAMESPACE=default        # Default namespace for agents
```

## Success Metrics

### MVP Definition of Done

1. **Functional**: All 5 test cases pass
2. **Performance**: Agent creation < 30 seconds
3. **UX**: Non-technical user can create agent without reading docs
4. **Reliability**: Parse accuracy > 85% on common patterns
5. **Documentation**: Getting started guide + 5 examples

### Post-MVP Metrics

1. **Adoption**: 10+ real agents deployed by users
2. **Satisfaction**: User feedback score > 4/5
3. **Coverage**: 20+ tool integrations available
4. **Reliability**: Parse accuracy > 95%

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM parsing inaccurate | High | Extensive prompt engineering + test suite |
| Tool ecosystem too small | Medium | Start with 10 most common tools |
| Kubernetes complexity | Medium | Excellent error messages + docs |
| Cost of LLM calls | Low | Cache common patterns, use efficient models |
| Security (NL contains secrets) | High | Detect and warn about secrets in descriptions |

## Next Steps After MVP

1. **Interactive wizard** for ambiguous requests
2. **Persona marketplace** for sharing custom personas
3. **Agent templates** library with one-click deploy
4. **Voice input** for hands-free agent creation
5. **Multi-agent workflows** for complex automation
6. **Learning system** that improves parsing from user feedback

## Appendix: Cilium-Inspired UX Patterns

What makes Cilium's CLI excellent:
1. **Immediate feedback**: `cilium status` shows everything instantly
2. **Beautiful output**: Tables, colors, spinners
3. **Helpful errors**: Clear next steps when something fails
4. **Discoverability**: `cilium connectivity test` makes features obvious
5. **Debugging**: `cilium monitor` gives real-time visibility

Apply to langop:
```bash
langop status                    # Show all agents, personas, tools
langop logs agent-name -f        # Real-time agent execution logs
langop test agent-name           # Dry-run agent to test behavior
langop monitor                   # Watch all agent activity
langop debug agent-name          # Show full agent state + recent errors
```
