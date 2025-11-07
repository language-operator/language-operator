# Synthesis Prompt Templates

This directory contains the prompt templates used by the synthesis system to generate agent DSL code.

## Template Files

- **agent_synthesis.tmpl** - Main template for generating agent Ruby DSL code from natural language instructions
- **persona_distillation.tmpl** - Template for distilling detailed persona specifications into concise system messages

## Usage

The templates are embedded into the Go binary using `//go:embed` directives. The actual embedded files are located in `src/pkg/synthesis/` (same directory as the synthesizer code) due to Go embed path requirements.

**Note**: When modifying templates, update BOTH locations:
1. `src/config/synthesis/*.tmpl` (canonical source, version controlled)
2. `src/pkg/synthesis/*.tmpl` (embedded into binary)

## Template Variables

### agent_synthesis.tmpl

- `Instructions` - User's natural language instructions for the agent
- `ToolsList` - Formatted list of available MCP tools
- `ModelsList` - Formatted list of available language models
- `AgentName` - Name of the agent being synthesized
- `TemporalIntent` - Detected execution pattern (One-shot/Scheduled/Continuous)
- `PersonaSection` - Optional persona/system prompt block
- `ScheduleSection` - Optional schedule block (for scheduled agents)
- `ConstraintsSection` - Constraints block with max_iterations based on intent
- `ScheduleRules` - Contextual rules for the LLM based on detected intent

### persona_distillation.tmpl

- `PersonaName` - Name of the persona
- `PersonaDescription` - Detailed persona description
- `PersonaSystemPrompt` - Full system prompt specification
- `PersonaTone` - Desired communication tone
- `PersonaLanguage` - Language for responses
- `AgentInstructions` - The agent's goal/purpose
- `AgentTools` - Available tools for context

## Temporal Intent Detection

The synthesis system automatically detects execution patterns from instructions:

- **One-shot**: "run once", "one time", "single time" → `max_iterations 10`
- **Scheduled**: "every", "daily", "hourly", "cron" → Schedule + `max_iterations 999999`
- **Continuous** (default): No temporal keywords → `max_iterations 999999`

This ensures agents run appropriately based on user intent without requiring explicit configuration.
