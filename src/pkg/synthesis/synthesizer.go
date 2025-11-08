package synthesis

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-logr/logr"
)

//go:embed agent_synthesis.tmpl
var agentSynthesisTemplate string

//go:embed persona_distillation.tmpl
var personaDistillationTemplate string

// TemporalIntent represents the detected execution pattern from user instructions
type TemporalIntent int

const (
	// Continuous indicates agent should run indefinitely
	Continuous TemporalIntent = iota
	// Scheduled indicates agent should run on a schedule
	Scheduled
	// OneShot indicates agent should run once or a limited number of times
	OneShot
)

func (t TemporalIntent) String() string {
	switch t {
	case OneShot:
		return "One-shot"
	case Scheduled:
		return "Scheduled"
	case Continuous:
		return "Continuous"
	default:
		return "Unknown"
	}
}

// AgentSynthesizer is the interface for synthesizing agent code
type AgentSynthesizer interface {
	SynthesizeAgent(ctx context.Context, req AgentSynthesisRequest) (*AgentSynthesisResponse, error)
	DistillPersona(ctx context.Context, persona PersonaInfo, agentContext AgentContext) (string, error)
}

// ChatModel is the interface for LLM chat models (eino)
type ChatModel interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
}

// Synthesizer generates agent DSL code from natural language instructions
type Synthesizer struct {
	chatModel ChatModel
	log       logr.Logger
}

// AgentSynthesisRequest contains all information needed to synthesize an agent
type AgentSynthesisRequest struct {
	Instructions string
	Tools        []string
	Models       []string
	PersonaText  string // Distilled persona
	AgentName    string
	Namespace    string
}

// AgentSynthesisResponse contains the synthesized DSL code
type AgentSynthesisResponse struct {
	DSLCode          string
	Error            string
	DurationSeconds  float64
	ValidationErrors []string
}

// PersonaInfo contains persona details for distillation
type PersonaInfo struct {
	Name         string
	Description  string
	SystemPrompt string
	Tone         string
	Language     string
}

// AgentContext provides context for persona distillation
type AgentContext struct {
	AgentName    string
	Instructions string
	Tools        string
}

// NewSynthesizer creates a new synthesizer instance using eino ChatModel
func NewSynthesizer(chatModel ChatModel, log logr.Logger) *Synthesizer {
	return &Synthesizer{
		chatModel: chatModel,
		log:       log,
	}
}

// SynthesizeAgent generates Ruby DSL code from natural language instructions
func (s *Synthesizer) SynthesizeAgent(ctx context.Context, req AgentSynthesisRequest) (*AgentSynthesisResponse, error) {
	startTime := time.Now()

	s.log.Info("Synthesizing agent code",
		"agent", req.AgentName,
		"namespace", req.Namespace,
		"tools", len(req.Tools),
		"models", len(req.Models))

	// Build the synthesis prompt
	prompt := s.buildSynthesisPrompt(req)

	// Call LLM using eino ChatModel
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	response, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		duration := time.Since(startTime).Seconds()
		return &AgentSynthesisResponse{
			Error:           err.Error(),
			DurationSeconds: duration,
		}, err
	}

	dslCode := response.Content

	// Extract code from markdown blocks if present
	dslCode = extractCodeFromMarkdown(dslCode)

	// Validate the synthesized code (basic syntax check)
	validationErrors := []string{}
	if err := s.validateDSL(dslCode); err != nil {
		validationErrors = append(validationErrors, err.Error())
		duration := time.Since(startTime).Seconds()
		return &AgentSynthesisResponse{
			DSLCode:          dslCode,
			Error:            fmt.Sprintf("Validation failed: %v", err),
			DurationSeconds:  duration,
			ValidationErrors: validationErrors,
		}, err
	}

	duration := time.Since(startTime).Seconds()

	s.log.Info("Agent code synthesized successfully",
		"agent", req.AgentName,
		"codeLength", len(dslCode),
		"duration", duration)

	return &AgentSynthesisResponse{
		DSLCode:         dslCode,
		DurationSeconds: duration,
	}, nil
}

// DistillPersona converts a detailed persona into a concise system message
func (s *Synthesizer) DistillPersona(ctx context.Context, persona PersonaInfo, agentContext AgentContext) (string, error) {
	s.log.Info("Distilling persona",
		"persona", persona.Name,
		"agent", agentContext.AgentName)

	prompt := s.buildPersonaDistillationPrompt(persona, agentContext)

	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	response, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	distilled := response.Content

	s.log.Info("Persona distilled successfully",
		"persona", persona.Name,
		"length", len(distilled))

	return strings.TrimSpace(distilled), nil
}

// buildSynthesisPrompt creates the prompt for agent code synthesis
func (s *Synthesizer) buildSynthesisPrompt(req AgentSynthesisRequest) string {
	toolsList := "None"
	if len(req.Tools) > 0 {
		toolsList = ""
		for _, t := range req.Tools {
			toolsList += fmt.Sprintf("  - %s\n", t)
		}
	}

	modelsList := "None"
	if len(req.Models) > 0 {
		modelsList = ""
		for _, m := range req.Models {
			modelsList += fmt.Sprintf("  - %s\n", m)
		}
	}

	personaSection := ""
	if req.PersonaText != "" {
		personaSection = fmt.Sprintf(`
  # Persona/system prompt
  persona <<~PERSONA
%s
  PERSONA
`, indentText(req.PersonaText, "    "))
	}

	// Detect temporal intent from instructions
	intent := detectTemporalIntent(req.Instructions)

	// Build constraints and schedule sections based on intent
	constraintsSection := ""
	scheduleSection := ""
	scheduleRules := ""

	switch intent {
	case OneShot:
		constraintsSection = `  # One-shot execution detected from instructions
  constraints do
    max_iterations 10
    timeout "10m"
  end`
		scheduleRules = `2. One-shot execution detected - agent will run a limited number of times
3. Do NOT include a schedule block for one-shot agents`

	case Scheduled:
		constraintsSection = `  # Scheduled execution - high iteration limit for continuous operation
  constraints do
    max_iterations 999999
    timeout "10m"
  end`
		scheduleSection = `
  # Extract schedule from instructions (e.g., "daily at noon" -> "0 12 * * *")
  schedule "CRON_EXPRESSION"`
		scheduleRules = `2. Schedule detected - extract cron expression from instructions
3. Set schedule block with appropriate cron expression
4. Use high max_iterations for continuous scheduled operation`

	case Continuous:
		constraintsSection = `  # Continuous execution - no specific schedule or one-shot indicator found
  constraints do
    max_iterations 999999
    timeout "10m"
  end`
		scheduleRules = `2. No temporal intent detected - defaulting to continuous execution
3. Do NOT include a schedule block unless explicitly mentioned
4. Use high max_iterations for continuous operation`
	}

	// Execute template
	tmpl, err := template.New("agent_synthesis").Parse(agentSynthesisTemplate)
	if err != nil {
		s.log.Error(err, "Failed to parse agent synthesis template")
		// Fallback to inline template if parsing fails
		return s.buildSynthesisPromptFallback(req, toolsList, modelsList, personaSection, intent, scheduleSection, constraintsSection, scheduleRules)
	}

	data := map[string]interface{}{
		"Instructions":       req.Instructions,
		"ToolsList":          toolsList,
		"ModelsList":         modelsList,
		"AgentName":          req.AgentName,
		"TemporalIntent":     intent.String(),
		"PersonaSection":     personaSection,
		"ScheduleSection":    scheduleSection,
		"ConstraintsSection": constraintsSection,
		"ScheduleRules":      scheduleRules,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		s.log.Error(err, "Failed to execute agent synthesis template")
		return s.buildSynthesisPromptFallback(req, toolsList, modelsList, personaSection, intent, scheduleSection, constraintsSection, scheduleRules)
	}

	return buf.String()
}

// buildSynthesisPromptFallback provides a fallback when template loading fails
func (s *Synthesizer) buildSynthesisPromptFallback(req AgentSynthesisRequest, toolsList, modelsList, personaSection string, intent TemporalIntent, scheduleSection, constraintsSection, scheduleRules string) string {
	// Use a heredoc-style string to avoid backtick issues
	return fmt.Sprintf(`You are generating Ruby DSL code for an autonomous agent in a Kubernetes operator.

**User Instructions:**
%s

**Available Tools:**
%s

**Available Models:**
%s

**Agent Name:** %s

**Detected Temporal Intent:** %s

Generate Ruby DSL code using this exact format (wrapped in triple-backticks with ruby):

`+"```ruby"+`
require 'language_operator'

agent "%s" do
  description "Brief description extracted from instructions"
%s%s
  # Extract objectives from instructions
  objectives [
    "First objective",
    "Second objective"
  ]

  # Define workflow if instructions mention specific steps
  # workflow do
  #   step :step_name, tool: "tool_name", params: {key: "value"}
  #   step :another_step, depends_on: :step_name
  # end

%s

  # Output configuration (if workspace enabled)
  output do
    workspace "results/output.txt"
  end
end
`+"```"+`

**Rules:**
1. Generate ONLY the Ruby code within triple-backticks, no explanations before or after
%s
5. Break down instructions into clear, actionable objectives
6. Create workflow steps if instructions describe a process
7. Use available tools in workflow steps
8. Use the agent name: "%s"

Generate the code now:`,
		req.Instructions,
		toolsList,
		modelsList,
		req.AgentName,
		intent.String(),
		req.AgentName,
		personaSection,
		scheduleSection,
		constraintsSection,
		scheduleRules,
		req.AgentName)
}

// buildPersonaDistillationPrompt creates the prompt for persona distillation
func (s *Synthesizer) buildPersonaDistillationPrompt(persona PersonaInfo, agentCtx AgentContext) string {
	// Execute template
	tmpl, err := template.New("persona_distillation").Parse(personaDistillationTemplate)
	if err != nil {
		s.log.Error(err, "Failed to parse persona distillation template")
		// Fallback to inline template if parsing fails
		return s.buildPersonaDistillationPromptFallback(persona, agentCtx)
	}

	data := map[string]interface{}{
		"PersonaName":         persona.Name,
		"PersonaDescription":  persona.Description,
		"PersonaSystemPrompt": persona.SystemPrompt,
		"PersonaTone":         persona.Tone,
		"PersonaLanguage":     persona.Language,
		"AgentInstructions":   agentCtx.Instructions,
		"AgentTools":          agentCtx.Tools,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		s.log.Error(err, "Failed to execute persona distillation template")
		return s.buildPersonaDistillationPromptFallback(persona, agentCtx)
	}

	return buf.String()
}

// buildPersonaDistillationPromptFallback provides a fallback when template loading fails
func (s *Synthesizer) buildPersonaDistillationPromptFallback(persona PersonaInfo, agentCtx AgentContext) string {
	return fmt.Sprintf(`Distill this persona into a single concise paragraph for an AI agent.

**Persona Details:**
Name: %s
Description: %s
System Prompt: %s
Tone: %s
Language: %s

**Agent Context:**
Goal: %s
Available Tools: %s

Generate a single paragraph (2-4 sentences) that captures the essence of this persona
in the context of the agent's goal. Focus on tone, expertise, and key behaviors.

Output ONLY the distilled persona paragraph, nothing else.

Distilled persona:`,
		persona.Name,
		persona.Description,
		persona.SystemPrompt,
		persona.Tone,
		persona.Language,
		agentCtx.Instructions,
		agentCtx.Tools)
}

// validateDSL performs comprehensive validation on the synthesized DSL code
func (s *Synthesizer) validateDSL(code string) error {
	// Basic checks
	if code == "" {
		return fmt.Errorf("empty code generated")
	}

	if !strings.Contains(code, "agent ") {
		return fmt.Errorf("code does not contain 'agent' definition")
	}

	if !strings.Contains(code, "require 'language_operator'") && !strings.Contains(code, `require "language_operator"`) {
		return fmt.Errorf("code does not require language_operator")
	}

	// Check for basic Ruby syntax issues
	if strings.Count(code, " do") != strings.Count(code, "end") {
		s.log.Info("Warning: mismatched do/end blocks", "code", code[:min(200, len(code))])
		// Don't fail on this, just warn
	}

	// Security validation: detect dangerous Ruby patterns
	if err := s.validateSecurity(code); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	return nil
}

// validateSecurity checks for dangerous Ruby patterns that could be security risks
func (s *Synthesizer) validateSecurity(code string) error {
	// Dangerous method calls that allow arbitrary code execution
	dangerousMethods := []string{
		"system(",
		"exec(",
		"spawn(",
		"eval(",
		"instance_eval(",
		"class_eval(",
		"module_eval(",
		"`",  // Backticks for shell execution
		"%x", // Alternative shell execution syntax
	}

	for _, method := range dangerousMethods {
		if strings.Contains(code, method) {
			return fmt.Errorf("dangerous method call detected: %s", method)
		}
	}

	// File system operations outside of DSL context
	// Allow File.read and File.write in workflow context, but block others
	dangerousFileOps := []string{
		"File.delete",
		"File.unlink",
		"FileUtils.rm",
		"FileUtils.rm_rf",
		"Dir.delete",
		"Dir.rmdir",
	}

	for _, op := range dangerousFileOps {
		if strings.Contains(code, op) {
			return fmt.Errorf("dangerous file operation detected: %s", op)
		}
	}

	// Network operations outside of DSL (tools should handle this)
	dangerousNetOps := []string{
		"Net::HTTP",
		"TCPSocket",
		"UDPSocket",
		"Socket.",
	}

	for _, op := range dangerousNetOps {
		if strings.Contains(code, op) {
			return fmt.Errorf("direct network operation detected: %s (use tools instead)", op)
		}
	}

	// Process and environment manipulation
	dangerousProcess := []string{
		"Process.kill",
		"Process.spawn",
		"fork(",
		"fork {",
		"exit!",
		"abort(",
	}

	for _, proc := range dangerousProcess {
		if strings.Contains(code, proc) {
			return fmt.Errorf("dangerous process operation detected: %s", proc)
		}
	}

	// Kernel methods that can be dangerous
	dangerousKernel := []string{
		"Kernel.system",
		"Kernel.exec",
		"Kernel.spawn",
		"Kernel.eval",
		"Kernel.load",
		"Kernel.require_relative", // Could load arbitrary code
	}

	for _, kern := range dangerousKernel {
		if strings.Contains(code, kern) {
			return fmt.Errorf("dangerous Kernel method detected: %s", kern)
		}
	}

	// Load and require operations (except safe ones)
	if strings.Contains(code, "load(") {
		return fmt.Errorf("dangerous code loading detected: load()")
	}

	// Check for require statements that aren't language_operator
	requirePattern := `require`
	if strings.Contains(code, requirePattern) {
		// Split by lines and check each require statement
		lines := strings.Split(code, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "require") {
				// Allow only language_operator
				if !strings.Contains(trimmed, "language_operator") {
					return fmt.Errorf("unauthorized require statement detected: %s", trimmed)
				}
			}
		}
	}

	// Check for send/public_send/__send__ which can bypass access controls
	dangerousSend := []string{
		".send(",
		".public_send(",
		".__send__(",
		".method(",
		".instance_method(",
	}

	for _, send := range dangerousSend {
		if strings.Contains(code, send) {
			return fmt.Errorf("dangerous reflective method detected: %s", send)
		}
	}

	// Check for constant manipulation
	dangerousConst := []string{
		"const_set",
		"remove_const",
		"const_missing",
	}

	for _, const_op := range dangerousConst {
		if strings.Contains(code, const_op) {
			return fmt.Errorf("dangerous constant manipulation detected: %s", const_op)
		}
	}

	return nil
}

// Helper functions

func indentText(text string, indent string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, indent+line)
	}
	return strings.Join(result, "\n")
}

func extractCodeFromMarkdown(content string) string {
	// Remove markdown code fences if present
	content = strings.TrimSpace(content)

	// Try ```ruby first
	if idx := strings.Index(content, "```ruby"); idx != -1 {
		content = content[idx+7:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	} else if idx := strings.Index(content, "```"); idx != -1 {
		// Try generic ``` blocks
		content = content[idx+3:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	}

	return strings.TrimSpace(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// detectTemporalIntent analyzes user instructions to determine execution pattern
func detectTemporalIntent(instructions string) TemporalIntent {
	lower := strings.ToLower(instructions)

	// One-shot indicators (highest priority)
	oneShotKeywords := []string{
		"run once",
		"one time",
		"single time",
		"execute once",
		"just once",
		"do once",
		"perform once",
	}
	for _, keyword := range oneShotKeywords {
		if strings.Contains(lower, keyword) {
			return OneShot
		}
	}

	// Schedule indicators (medium priority)
	scheduleKeywords := []string{
		"every",
		"daily",
		"hourly",
		"weekly",
		"monthly",
		"cron",
		"at midnight",
		"at noon",
		"schedule",
		"periodically",
		"regularly",
		"each day",
		"each hour",
		"each week",
		"each month",
	}
	for _, keyword := range scheduleKeywords {
		if strings.Contains(lower, keyword) {
			return Scheduled
		}
	}

	// Default to continuous execution (lowest priority)
	// This is for agents like "provides fun facts" that should run continuously
	return Continuous
}
