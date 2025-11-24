package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MockLLMService provides a mock LLM API server for testing
type MockLLMService struct {
	server      *httptest.Server
	requests    []string
	shouldError bool
	errorMsg    string
}

// MockChatModel implements the ChatModel interface for testing
type MockChatModel struct {
	mockLLM *MockLLMService
}

// LLMRequest represents an incoming LLM request
type LLMRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents an LLM API response
type LLMResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// NewMockLLMService creates a new mock LLM HTTP service
func NewMockLLMService(t *testing.T) *MockLLMService {
	m := &MockLLMService{
		requests:    make([]string, 0),
		shouldError: false,
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.handleRequest(w, r, t)
	}))

	return m
}

// NewMockLLMServiceWithError creates a mock that returns errors
func NewMockLLMServiceWithError(t *testing.T, errorMsg string) *MockLLMService {
	m := &MockLLMService{
		requests:    make([]string, 0),
		shouldError: true,
		errorMsg:    errorMsg,
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.handleRequest(w, r, t)
	}))

	return m
}

// NewMockChatModel creates a ChatModel that uses the mock LLM service
func NewMockChatModel(mockLLM *MockLLMService) *MockChatModel {
	return &MockChatModel{
		mockLLM: mockLLM,
	}
}

// Generate implements the ChatModel interface (eino)
func (m *MockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// Extract the last user message content (full synthesis prompt)
	var promptContent string
	for i := len(input) - 1; i >= 0; i-- {
		if input[i].Role == schema.User && input[i].Content != "" {
			promptContent = input[i].Content
			break
		}
	}

	// Extract just the user instructions from the synthesis prompt
	// The synthesizer wraps instructions in a section like:
	// **User Instructions:**
	// <instructions>
	instructions := extractUserInstructions(promptContent)

	// Generate mock synthesis
	code := m.mockLLM.generateMockSynthesis(instructions)

	if m.mockLLM.shouldError {
		return nil, fmt.Errorf(m.mockLLM.errorMsg)
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: code,
	}, nil
}

// extractUserInstructions extracts just the user instructions from the synthesis prompt
func extractUserInstructions(prompt string) string {
	// Look for "**User Instructions:**" section
	lines := strings.Split(prompt, "\n")
	var instructions strings.Builder
	inInstructionsSection := false

	for _, line := range lines {
		if strings.Contains(line, "**User Instructions:**") {
			inInstructionsSection = true
			continue
		}
		if inInstructionsSection {
			// Stop at next section marker
			if strings.HasPrefix(line, "**") {
				break
			}
			// Collect instruction lines
			if len(strings.TrimSpace(line)) > 0 {
				if instructions.Len() > 0 {
					instructions.WriteString("\n")
				}
				instructions.WriteString(line)
			}
		}
	}

	return instructions.String()
}

// URL returns the mock service URL
func (m *MockLLMService) URL() string {
	return m.server.URL
}

// Close shuts down the mock service
func (m *MockLLMService) Close() {
	m.server.Close()
}

// Requests returns all received requests
func (m *MockLLMService) Requests() []string {
	return m.requests
}

// handleRequest handles incoming API requests
func (m *MockLLMService) handleRequest(w http.ResponseWriter, r *http.Request, t *testing.T) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if m.shouldError {
		http.Error(w, m.errorMsg, http.StatusInternalServerError)
		return
	}

	var req LLMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Store request
	for _, msg := range req.Messages {
		m.requests = append(m.requests, msg.Content)
	}

	// Extract instructions
	var instructions string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			instructions = req.Messages[i].Content
			break
		}
	}

	// Generate code
	code := m.generateMockSynthesis(instructions)

	// Send response
	response := LLMResponse{
		ID:      "mock-response-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: code,
				},
				FinishReason: "stop",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateMockSynthesis generates predictable Ruby DSL code
func (m *MockLLMService) generateMockSynthesis(instructions string) string {
	lower := strings.ToLower(instructions)

	// Don't generate code for empty/whitespace instructions
	if strings.TrimSpace(instructions) == "" {
		return ""
	}

	schedule := m.extractSchedule(lower)

	// Generate agent code with require statement using DSL v1 task/main model
	// The synthesizer validates this is present, and the validator wrapper
	// strips it before security validation (see issue #41 in language-operator-gem)
	code := `require 'language_operator'

agent "test-agent" do
  description "Generated agent for testing"`

	if schedule != "" {
		code += fmt.Sprintf(`
  schedule "%s"`, schedule)
	}

	// Add task definition based on instructions
	taskName := m.extractTaskName(instructions)
	code += fmt.Sprintf(`

  task :%s,
    instructions: "%s",
    inputs: {},
    outputs: { result: 'string' }

  main do |inputs|
    result = execute_task(:%s)
    result
  end

  output do |outputs|
    puts outputs.inspect
  end
end
`, taskName, instructions, taskName)

	return code
}

// extractSchedule extracts cron schedule from instructions
func (m *MockLLMService) extractSchedule(instructions string) string {
	if strings.Contains(instructions, "5 minute") {
		return "*/5 * * * *"
	}
	if strings.Contains(instructions, "15 minute") {
		return "*/15 * * * *"
	}
	if strings.Contains(instructions, "hour") {
		return "0 * * * *"
	}
	if strings.Contains(instructions, "daily") || strings.Contains(instructions, "every day") {
		if strings.Contains(instructions, "4pm") || strings.Contains(instructions, "16:00") {
			return "0 16 * * *"
		}
		if strings.Contains(instructions, "9am") || strings.Contains(instructions, "09:00") {
			return "0 9 * * *"
		}
		return "0 0 * * *"
	}
	if strings.Contains(instructions, "weekly") || strings.Contains(instructions, "every week") {
		return "0 0 * * 0"
	}
	return ""
}

// extractTaskName generates a task name from instructions
func (m *MockLLMService) extractTaskName(instructions string) string {
	lower := strings.ToLower(instructions)
	
	// Generate task name based on keywords in instructions
	if strings.Contains(lower, "review") || strings.Contains(lower, "check") {
		return "review_task"
	}
	if strings.Contains(lower, "sync") || strings.Contains(lower, "data") {
		return "sync_task"
	}
	if strings.Contains(lower, "report") || strings.Contains(lower, "send") {
		return "report_task"
	}
	if strings.Contains(lower, "process") || strings.Contains(lower, "tasks") {
		return "process_task"
	}
	
	// Default task name
	return "main_task"
}

// extractTools identifies tools from instructions
func (m *MockLLMService) extractTools(instructions string) []string {
	tools := []string{}

	// Don't add default tools for empty instructions
	if strings.TrimSpace(instructions) == "" {
		return tools
	}

	toolKeywords := map[string]string{
		"spreadsheet":   "google-sheets",
		"google sheets": "google-sheets",
		"email":         "email",
		"send email":    "email",
		"send me":       "email",
		"report":        "email",
		"web":           "web-fetch",
		"http":          "web-fetch",
		"https":         "web-fetch",
		"api":           "web-fetch",
		"slack":         "slack",
		"jira":          "jira",
		"github":        "github",
	}

	for keyword, tool := range toolKeywords {
		if strings.Contains(instructions, keyword) {
			// Avoid duplicates
			found := false
			for _, t := range tools {
				if t == tool {
					found = true
					break
				}
			}
			if !found {
				tools = append(tools, tool)
			}
		}
	}

	if len(tools) == 0 {
		tools = append(tools, "web-fetch")
	}

	return tools
}
