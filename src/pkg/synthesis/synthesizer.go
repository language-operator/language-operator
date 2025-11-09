package synthesis

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/based/language-operator/pkg/validation"
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
	chatModel   ChatModel
	log         logr.Logger
	costTracker *CostTracker
	modelName   string
}

// AgentSynthesisRequest contains all information needed to synthesize an agent
type AgentSynthesisRequest struct {
	Instructions string
	Tools        []string
	Models       []string
	PersonaText  string // Distilled persona
	AgentName    string
	Namespace    string

	// Self-Healing Context (NEW)
	ErrorContext      *ErrorContext `json:"errorContext,omitempty"`
	IsRetry           bool          `json:"isRetry"`
	AttemptNumber     int32         `json:"attemptNumber"`
	LastKnownGoodCode string        `json:"lastKnownGoodCode,omitempty"`
}

// AgentSynthesisResponse contains the synthesized DSL code
type AgentSynthesisResponse struct {
	DSLCode          string
	Error            string
	DurationSeconds  float64
	ValidationErrors []string
	Cost             *SynthesisCost // Cost tracking for this synthesis
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

// ErrorContext provides error information for self-healing synthesis
type ErrorContext struct {
	RuntimeErrors       []RuntimeError `json:"runtimeErrors"`
	ValidationErrors    []string       `json:"validationErrors"`
	LastCrashLog        string         `json:"lastCrashLog"`
	ConsecutiveFailures int32          `json:"consecutiveFailures"`
	PreviousAttempts    int32          `json:"previousAttempts"`
}

// RuntimeError captures runtime failure information
type RuntimeError struct {
	Timestamp         string   `json:"timestamp"`
	ErrorType         string   `json:"errorType"`
	ErrorMessage      string   `json:"errorMessage"`
	StackTrace        []string `json:"stackTrace"`
	ContainerExitCode int32    `json:"exitCode"`
	SynthesisAttempt  int32    `json:"synthesisAttempt"`
}

// NewSynthesizer creates a new synthesizer instance using eino ChatModel
func NewSynthesizer(chatModel ChatModel, log logr.Logger) *Synthesizer {
	return &Synthesizer{
		chatModel:   chatModel,
		log:         log,
		costTracker: nil, // Will be set via SetCostTracker
		modelName:   "unknown",
	}
}

// SetCostTracker sets the cost tracker for this synthesizer
func (s *Synthesizer) SetCostTracker(tracker *CostTracker, modelName string) {
	s.costTracker = tracker
	s.modelName = modelName
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

	// Call the chat model (returns *schema.Message, not *schema.ChatCompletionResponse)
	responseMsg, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		duration := time.Since(startTime).Seconds()
		return &AgentSynthesisResponse{
			Error:           err.Error(),
			DurationSeconds: duration,
		}, err
	}

	dslCode := responseMsg.Content

	// Track cost if cost tracker is configured
	var synthesisCost *SynthesisCost
	if s.costTracker != nil {
		// Note: eino's Generate returns *schema.Message, not *schema.ChatCompletionResponse
		// We need to access the response metadata to get token usage
		// For now, estimate tokens as we don't have direct access to usage data
		inputTokens := EstimateTokens(prompt)
		outputTokens := EstimateTokens(dslCode)
		synthesisCost = s.costTracker.CalculateCost(inputTokens, outputTokens, s.modelName)

		s.log.Info("Synthesis cost tracked",
			"agent", req.AgentName,
			"inputTokens", synthesisCost.InputTokens,
			"outputTokens", synthesisCost.OutputTokens,
			"totalCost", synthesisCost.TotalCost,
			"currency", synthesisCost.Currency)
	}

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
		Cost:            synthesisCost,
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
		"ErrorContext":       req.ErrorContext,
		"AttemptNumber":      req.AttemptNumber,
		"MaxAttempts":        5, // TODO: Make this configurable
		"LastKnownGoodCode":  req.LastKnownGoodCode,
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

	// Security validation: use AST-based validator
	if err := validation.ValidateRubyCode(code); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
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
