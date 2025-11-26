package synthesis

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/validation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed agent_synthesis.tmpl
var agentSynthesisTemplate string

//go:embed persona_distillation.tmpl
var personaDistillationTemplate string

// Package-level tracer for OpenTelemetry instrumentation
var tracer trace.Tracer = otel.Tracer("language-operator/synthesizer")

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
	chatModel     ChatModel
	log           logr.Logger
	costTracker   *CostTracker
	modelName     string
	schemaVersion string // DSL schema version for telemetry tracking
}

// AgentSynthesisRequest contains all information needed to synthesize an agent
type AgentSynthesisRequest struct {
	Instructions string
	Tools        []string                    // Deprecated: use ToolSchemas instead
	ToolSchemas  []langopv1alpha1.ToolSchema // Complete tool schemas with parameters/types
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

// NewSynthesizerFromLanguageModel creates a synthesizer from a LanguageModel CRD
func NewSynthesizerFromLanguageModel(ctx context.Context, k8sClient client.Client, model *langopv1alpha1.LanguageModel, log logr.Logger) (*Synthesizer, error) {
	// Get API key from secret
	apiKey := ""
	if model.Spec.APIKeySecretRef != nil {
		secret := &corev1.Secret{}
		secretNamespace := model.Spec.APIKeySecretRef.Namespace
		if secretNamespace == "" {
			secretNamespace = model.Namespace
		}
		secretKey := model.Spec.APIKeySecretRef.Key
		if secretKey == "" {
			secretKey = "api-key"
		}

		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      model.Spec.APIKeySecretRef.Name,
			Namespace: secretNamespace,
		}, secret); err != nil {
			return nil, fmt.Errorf("failed to get API key secret: %w", err)
		}

		apiKey = string(secret.Data[secretKey])
		if apiKey == "" {
			return nil, fmt.Errorf("API key not found in secret %s/%s at key %s", secretNamespace, model.Spec.APIKeySecretRef.Name, secretKey)
		}
	}

	// Build eino OpenAI ChatModel config
	config := &openai.ChatModelConfig{
		Model:  model.Spec.ModelName,
		APIKey: apiKey,
	}

	// Set endpoint for openai-compatible providers
	if model.Spec.Endpoint != "" {
		endpoint := model.Spec.Endpoint
		// Normalize endpoint: add /v1 suffix if not present (required for OpenAI-compatible APIs)
		if !strings.HasSuffix(endpoint, "/v1") && !strings.HasSuffix(endpoint, "/v1/") {
			endpoint = strings.TrimSuffix(endpoint, "/") + "/v1"
		}
		config.BaseURL = endpoint
	}

	// Apply configuration options
	if model.Spec.Configuration != nil {
		if model.Spec.Configuration.Temperature != nil {
			temp := float32(*model.Spec.Configuration.Temperature)
			config.Temperature = &temp
		}
		if model.Spec.Configuration.MaxTokens != nil {
			maxTokens := int(*model.Spec.Configuration.MaxTokens)
			config.MaxTokens = &maxTokens
		}
	} else {
		// Default settings for synthesis
		temp := float32(0.3)
		config.Temperature = &temp
		maxTokens := 8192
		config.MaxTokens = &maxTokens
	}

	// Create ChatModel
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChatModel: %w", err)
	}

	synth := NewSynthesizer(chatModel, log)
	synth.modelName = model.Spec.ModelName
	return synth, nil
}

// NewSynthesizer creates a new synthesizer instance using eino ChatModel
func NewSynthesizer(chatModel ChatModel, log logr.Logger) *Synthesizer {
	// Fetch DSL schema version for telemetry tracking
	// Use context with timeout to avoid blocking on schema fetch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	schemaVersion, err := GetSchemaVersion(ctx)
	if err != nil {
		// Log warning but continue - don't block synthesizer creation
		log.Info("Warning: Failed to fetch DSL schema version, continuing without it",
			"error", err.Error())
		schemaVersion = "" // Empty string indicates version unavailable
	} else {
		// Log schema version at INFO level during initialization
		log.Info("DSL schema version loaded", "version", schemaVersion)
	}

	return &Synthesizer{
		chatModel:     chatModel,
		log:           log,
		costTracker:   nil, // Will be set via SetCostTracker
		modelName:     "unknown",
		schemaVersion: schemaVersion,
	}
}

// SetCostTracker sets the cost tracker for this synthesizer
func (s *Synthesizer) SetCostTracker(tracker *CostTracker, modelName string) {
	s.costTracker = tracker
	s.modelName = modelName
}

// SynthesizeAgent generates Ruby DSL code from natural language instructions
func (s *Synthesizer) SynthesizeAgent(ctx context.Context, req AgentSynthesisRequest) (*AgentSynthesisResponse, error) {
	// Start synthesis span
	ctx, span := tracer.Start(ctx, "synthesis.agent.generate")
	defer span.End()

	// Add initial span attributes
	spanAttrs := []attribute.KeyValue{
		attribute.String("synthesis.agent_name", req.AgentName),
		attribute.String("synthesis.namespace", req.Namespace),
		attribute.Int("synthesis.tools_count", len(req.Tools)),
		attribute.Int("synthesis.models_count", len(req.Models)),
		attribute.Bool("synthesis.is_retry", req.IsRetry),
		attribute.Int("synthesis.attempt_number", int(req.AttemptNumber)),
	}

	// Add DSL schema version if available
	if s.schemaVersion != "" {
		spanAttrs = append(spanAttrs, attribute.String("dsl.schema.version", s.schemaVersion))
	}

	span.SetAttributes(spanAttrs...)

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
		// Record error in span
		span.RecordError(err)
		span.SetStatus(codes.Error, "LLM call failed")
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

		// Add token/cost attributes to span
		span.SetAttributes(
			attribute.Int64("synthesis.input_tokens", synthesisCost.InputTokens),
			attribute.Int64("synthesis.output_tokens", synthesisCost.OutputTokens),
			attribute.Float64("synthesis.cost_usd", synthesisCost.TotalCost),
			attribute.String("synthesis.model", s.modelName),
		)

		s.log.Info("Synthesis cost tracked",
			"agent", req.AgentName,
			"inputTokens", synthesisCost.InputTokens,
			"outputTokens", synthesisCost.OutputTokens,
			"totalCost", synthesisCost.TotalCost,
			"currency", synthesisCost.Currency)
	}

	// Extract code from markdown blocks if present
	dslCode = extractCodeFromMarkdown(dslCode)

	// Validate against DSL schema first
	validationErrors := []string{}
	schemaViolations, err := ValidateGeneratedCodeAgainstSchema(ctx, dslCode)
	if err != nil {
		s.log.Error(err, "Schema validation execution failed", "agent", req.AgentName)

		duration := time.Since(startTime).Seconds()
		span.RecordError(err)
		span.SetStatus(codes.Error, "Schema validation execution failed")

		return &AgentSynthesisResponse{
			DSLCode:          dslCode,
			Error:            fmt.Sprintf("Schema validation execution failed: %v", err),
			DurationSeconds:  duration,
			ValidationErrors: []string{err.Error()},
			Cost:             synthesisCost,
		}, fmt.Errorf("schema validation execution failed: %w", err)
	} else if len(schemaViolations) > 0 {
		// Convert violations to error messages
		for _, violation := range schemaViolations {
			errMsg := fmt.Sprintf("Line %d: %s (%s)", violation.Location, violation.Message, violation.Type)
			validationErrors = append(validationErrors, errMsg)
		}

		// Add telemetry event for schema validation failure
		span.AddEvent("schema_validation_failed", trace.WithAttributes(
			attribute.Int("violation_count", len(schemaViolations)),
			attribute.String("schema_version", s.schemaVersion),
		))

		s.log.Info("Schema validation failed",
			"agent", req.AgentName,
			"violations", len(schemaViolations),
			"schemaVersion", s.schemaVersion)

		duration := time.Since(startTime).Seconds()
		span.RecordError(fmt.Errorf("schema validation failed with %d violations", len(schemaViolations)))
		span.SetStatus(codes.Error, "Schema validation failed")

		return &AgentSynthesisResponse{
			DSLCode:          dslCode,
			Error:            fmt.Sprintf("Schema validation failed: %d violations found", len(schemaViolations)),
			DurationSeconds:  duration,
			ValidationErrors: validationErrors,
			Cost:             synthesisCost,
		}, fmt.Errorf("schema validation failed with %d violations", len(schemaViolations))
	} else {
		// Schema validation passed - add telemetry event
		span.AddEvent("schema_validation_passed", trace.WithAttributes(
			attribute.String("schema_version", s.schemaVersion),
		))
	}

	// Validate the synthesized code (basic syntax and security checks)
	if err := s.validateDSL(ctx, dslCode); err != nil {
		validationErrors = append(validationErrors, err.Error())
		duration := time.Since(startTime).Seconds()
		// Record error in span
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed")
		return &AgentSynthesisResponse{
			DSLCode:          dslCode,
			Error:            fmt.Sprintf("Validation failed: %v", err),
			DurationSeconds:  duration,
			ValidationErrors: validationErrors,
		}, err
	}

	duration := time.Since(startTime).Seconds()

	// Add success attributes to span
	span.SetAttributes(
		attribute.Int("synthesis.code_length", len(dslCode)),
		attribute.Float64("synthesis.duration_seconds", duration),
	)
	span.SetStatus(codes.Ok, "Synthesis successful")

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

// buildToolsList creates formatted tool information for synthesis prompts
func (s *Synthesizer) buildToolsList(req AgentSynthesisRequest) string {
	// Use ToolSchemas if available, otherwise fall back to Tools for backward compatibility
	if len(req.ToolSchemas) > 0 {
		return s.formatToolSchemas(req.ToolSchemas)
	}

	if len(req.Tools) > 0 {
		toolsList := ""
		for _, t := range req.Tools {
			toolsList += fmt.Sprintf("  - %s\n", t)
		}
		return toolsList
	}

	return "None"
}

// formatToolSchemas converts ToolSchemas to human-readable format for LLM synthesis
func (s *Synthesizer) formatToolSchemas(schemas []langopv1alpha1.ToolSchema) string {
	if len(schemas) == 0 {
		return "None"
	}

	var builder strings.Builder
	for _, schema := range schemas {
		builder.WriteString(fmt.Sprintf("### %s\n", schema.Name))

		if schema.Description != "" {
			builder.WriteString(fmt.Sprintf("%s\n", schema.Description))
		}

		// Format input parameters
		if schema.InputSchema != nil && len(schema.InputSchema.Properties) > 0 {
			builder.WriteString("**Parameters:**\n")
			for paramName, prop := range schema.InputSchema.Properties {
				required := ""
				if containsString(schema.InputSchema.Required, paramName) {
					required = " (required)"
				}

				description := ""
				if prop.Description != "" {
					description = fmt.Sprintf(" - %s", prop.Description)
				}

				example := ""
				if prop.Example != "" {
					example = fmt.Sprintf(" (e.g., %s)", prop.Example)
				}

				builder.WriteString(fmt.Sprintf("- `%s`: %s%s%s%s\n",
					paramName, prop.Type, required, description, example))
			}
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// containsString checks if a string slice contains a given string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// buildSynthesisPrompt creates the prompt for agent code synthesis
func (s *Synthesizer) buildSynthesisPrompt(req AgentSynthesisRequest) string {
	toolsList := s.buildToolsList(req)

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

Generate Ruby DSL v1 code using the task/main model (wrapped in triple-backticks with ruby):

`+"```ruby"+`
require 'language_operator'

agent "%s" do
  description "Brief description extracted from instructions"
%s%s
  # Define tasks with type schemas (organic functions)
  task :example_task,
    instructions: "specific task description from instructions",
    inputs: { param: 'string' },
    outputs: { result: 'string' }

  # Add more tasks as needed based on instructions

  # Main execution flow
  main do |inputs|
    result = execute_task(:example_task, inputs: { param: "value" })
    result
  end

%s

  # Output configuration
  output do |outputs|
    puts outputs.inspect
  end
end
`+"```"+`

**Rules:**
1. Generate ONLY the Ruby code within triple-backticks, no explanations before or after
%s
5. Break down instructions into specific tasks with type schemas
6. Use task/main model, not workflow/steps
7. Each task must have inputs/outputs type contracts
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
func (s *Synthesizer) validateDSL(ctx context.Context, code string) error {
	// Start validation span
	ctx, span := tracer.Start(ctx, "synthesis.validate")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("validation.language", "ruby"),
		attribute.Int("validation.code_length", len(code)),
	)

	// Basic checks
	if code == "" {
		span.SetAttributes(attribute.String("validation.error_type", "empty_code"))
		span.RecordError(fmt.Errorf("empty code generated"))
		span.SetStatus(codes.Error, "Validation failed: empty code")
		return fmt.Errorf("empty code generated")
	}

	if !strings.Contains(code, "agent ") {
		span.SetAttributes(attribute.String("validation.error_type", "missing_agent"))
		err := fmt.Errorf("code does not contain 'agent' definition")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed: missing agent definition")
		return err
	}

	if !strings.Contains(code, "require 'language_operator'") && !strings.Contains(code, `require "language_operator"`) {
		span.SetAttributes(attribute.String("validation.error_type", "missing_require"))
		err := fmt.Errorf("code does not require language_operator")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed: missing require")
		return err
	}

	// Check for basic Ruby syntax issues
	if strings.Count(code, " do") != strings.Count(code, "end") {
		s.log.Info("Warning: mismatched do/end blocks", "code", code[:min(200, len(code))])
		// Don't fail on this, just warn
	}

	// Security validation: use AST-based validator
	if err := validation.ValidateRubyCode(code); err != nil {
		span.SetAttributes(attribute.String("validation.error_type", "security_violation"))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed: security violation")
		return fmt.Errorf("security validation failed: %w", err)
	}

	// Task validation: validate DSL v1 task/main structure
	taskValidator := NewTaskValidator(s.log)
	taskErrors, err := taskValidator.ValidateTaskAgent(ctx, code)
	if err != nil {
		span.SetAttributes(attribute.String("validation.error_type", "task_validation_execution_failed"))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Task validation execution failed")
		s.log.Info("Task validation execution failed", "error", err.Error())
		// Don't fail synthesis if validation execution fails - continue
	} else if len(taskErrors) > 0 {
		// Filter out warnings and count only errors
		errorCount := 0
		var errorMessages []string
		for _, taskErr := range taskErrors {
			if taskErr.Severity == "error" {
				errorCount++
				if taskErr.Task != "" {
					errorMessages = append(errorMessages, fmt.Sprintf("Task '%s': %s", taskErr.Task, taskErr.Message))
				} else {
					errorMessages = append(errorMessages, taskErr.Message)
				}
			}
		}

		if errorCount > 0 {
			span.SetAttributes(
				attribute.String("validation.error_type", "task_validation_failed"),
				attribute.Int("validation.task_error_count", errorCount),
			)
			err := fmt.Errorf("task validation failed: %s", strings.Join(errorMessages, "; "))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Task validation failed")
			return err
		}

		// Log warnings but don't fail
		warningCount := len(taskErrors) - errorCount
		if warningCount > 0 {
			span.SetAttributes(attribute.Int("validation.task_warning_count", warningCount))
			s.log.Info("Task validation warnings", "warningCount", warningCount)
		}
	} else {
		// Task validation passed
		span.AddEvent("task_validation_passed")
	}

	// Validation successful
	span.SetAttributes(attribute.String("validation.result", "success"))
	span.SetStatus(codes.Ok, "Validation successful")

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
