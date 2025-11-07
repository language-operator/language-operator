package synthesis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-logr/logr"
)

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

	// Use a heredoc-style string to avoid backtick issues
	return fmt.Sprintf(`You are generating Ruby DSL code for an autonomous agent in a Kubernetes operator.

**User Instructions:**
%s

**Available Tools:**
%s

**Available Models:**
%s

**Agent Name:** %s

Generate Ruby DSL code using this exact format (wrapped in triple-backticks with ruby):

`+"```ruby"+`
require 'language_operator'

agent "%s" do
  description "Brief description extracted from instructions"
%s
  # Extract schedule from instructions (e.g., "daily at noon" -> "0 12 * * *")
  # Only include schedule if instructions mention timing/frequency
  # schedule "CRON_EXPRESSION"

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

  # Infer reasonable constraints
  constraints do
    max_iterations 50
    timeout "10m"
  end

  # Output configuration (if workspace enabled)
  output do
    workspace "results/output.txt"
  end
end
`+"```"+`

**Rules:**
1. Generate ONLY the Ruby code within triple-backticks, no explanations before or after
2. Extract schedule ONLY if timing is mentioned (daily, hourly, cron, etc.)
3. If scheduled, set schedule and remove it from objectives
4. Break down instructions into clear, actionable objectives
5. Create workflow steps if instructions describe a process
6. Use available tools in workflow steps
7. Keep constraints reasonable (max_iterations: 50, timeout: "10m")
8. Use the agent name: "%s"

Generate the code now:`,
		req.Instructions,
		toolsList,
		modelsList,
		req.AgentName,
		req.AgentName,
		personaSection,
		req.AgentName)
}

// buildPersonaDistillationPrompt creates the prompt for persona distillation
func (s *Synthesizer) buildPersonaDistillationPrompt(persona PersonaInfo, agentCtx AgentContext) string {
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

// validateDSL performs basic validation on the synthesized DSL code
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
