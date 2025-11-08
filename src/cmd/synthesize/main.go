package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/based/language-operator/pkg/synthesis"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

const usage = `synthesize - Test agent synthesis locally

Usage:
  synthesize [flags] [instructions]
  synthesize [flags] --file=instructions.txt

Note: Flags must come before positional arguments.

Examples:
  # Basic usage
  synthesize "Provides fun facts about Ruby"

  # With options
  synthesize -agent-name=monitor -tools=web-tool,email-tool "Monitors system health"

  # From file
  synthesize -file=instructions.txt

  # JSON output
  synthesize -json "Agent instructions"

  # Validate only (no LLM call)
  synthesize -validate-only "Test agent"

Flags:
`

type Config struct {
	Instructions string
	File         string
	AgentName    string
	Tools        string
	Models       string
	JSON         bool
	ValidateOnly bool
	Endpoint     string
	APIKey       string
	Model        string
}

type Output struct {
	DSLCode          string   `json:"dsl_code"`
	TemporalIntent   string   `json:"temporal_intent"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
	Error            string   `json:"error,omitempty"`
	DurationSeconds  float64  `json:"duration_seconds"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()

	// Determine instructions source
	instructions := cfg.Instructions
	if cfg.File != "" {
		content, err := os.ReadFile(cfg.File)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		instructions = string(content)
	}

	if instructions == "" {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
		return fmt.Errorf("instructions required (provide as argument or --file)")
	}

	// Parse tools and models
	tools := parseList(cfg.Tools)
	models := parseList(cfg.Models)

	// Validate-only mode - exit early without LLM setup
	if cfg.ValidateOnly {
		return validateOnly(instructions, cfg.AgentName, tools, models)
	}

	// Set up logger for actual synthesis
	zapLog, _ := zap.NewDevelopment()
	log := zapr.NewLogger(zapLog)

	// Create LLM client
	chatModel, err := createChatModel(cfg, log)
	if err != nil {
		return err
	}

	// Create synthesizer
	synth := synthesis.NewSynthesizer(chatModel, log)

	// Build synthesis request
	req := synthesis.AgentSynthesisRequest{
		Instructions: instructions,
		Tools:        tools,
		Models:       models,
		AgentName:    cfg.AgentName,
	}

	// Synthesize
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := synth.SynthesizeAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("synthesis failed: %w", err)
	}

	// Output results
	if cfg.JSON {
		return outputJSON(resp)
	}

	return outputText(resp)
}

func parseFlags() Config {
	cfg := Config{}

	flag.StringVar(&cfg.File, "file", "", "Read instructions from file")
	flag.StringVar(&cfg.AgentName, "agent-name", "test-agent", "Agent name")
	flag.StringVar(&cfg.Tools, "tools", "", "Comma-separated list of tools")
	flag.StringVar(&cfg.Models, "models", "", "Comma-separated list of models")
	flag.BoolVar(&cfg.JSON, "json", false, "Output in JSON format")
	flag.BoolVar(&cfg.ValidateOnly, "validate-only", false, "Only validate, don't call LLM")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}

	flag.Parse()

	// Get instructions from positional argument if not from file
	if cfg.File == "" && flag.NArg() > 0 {
		cfg.Instructions = strings.Join(flag.Args(), " ")
	}

	// Get LLM configuration from environment
	cfg.Endpoint = os.Getenv("SYNTHESIS_ENDPOINT")
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.anthropic.com/v1"
	}

	cfg.APIKey = os.Getenv("SYNTHESIS_API_KEY")
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	cfg.Model = os.Getenv("SYNTHESIS_MODEL")
	if cfg.Model == "" {
		cfg.Model = "claude-3-5-sonnet-20241022"
	}

	return cfg
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func createChatModel(cfg Config, log logr.Logger) (synthesis.ChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("SYNTHESIS_API_KEY or ANTHROPIC_API_KEY environment variable required")
	}

	// Create OpenAI-compatible client (works with Anthropic API)
	client, err := openai.NewChatModel(
		context.Background(),
		&openai.ChatModelConfig{
			BaseURL: cfg.Endpoint,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &chatModelAdapter{client: client}, nil
}

// chatModelAdapter adapts eino ChatModel to synthesis.ChatModel interface
type chatModelAdapter struct {
	client model.ChatModel
}

func (a *chatModelAdapter) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return a.client.Generate(ctx, input, opts...)
}

func validateOnly(instructions, agentName string, tools, models []string) error {
	fmt.Println("Validation mode - no LLM call")
	fmt.Printf("Instructions: %s\n", instructions)
	fmt.Printf("Agent Name: %s\n", agentName)
	if len(tools) > 0 {
		fmt.Printf("Tools: %s\n", strings.Join(tools, ", "))
	}
	if len(models) > 0 {
		fmt.Printf("Models: %s\n", strings.Join(models, ", "))
	}
	return nil
}

func outputJSON(resp *synthesis.AgentSynthesisResponse) error {
	out := Output{
		DSLCode:          resp.DSLCode,
		ValidationErrors: resp.ValidationErrors,
		Error:            resp.Error,
		DurationSeconds:  resp.DurationSeconds,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func outputText(resp *synthesis.AgentSynthesisResponse) error {
	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		if len(resp.ValidationErrors) > 0 {
			fmt.Fprintf(os.Stderr, "Validation errors:\n")
			for _, e := range resp.ValidationErrors {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
		}
		return fmt.Errorf("synthesis failed")
	}

	fmt.Printf("# Synthesis completed in %.2fs\n\n", resp.DurationSeconds)
	fmt.Println(resp.DSLCode)

	return nil
}
