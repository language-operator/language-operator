package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockLLMService provides a mock LLM API for testing
type MockLLMService struct {
	server      *httptest.Server
	requests    []string
	shouldError bool
	errorMsg    string
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

// NewMockLLMService creates a new mock LLM service
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

// NewMockLLMServiceWithError creates a new mock LLM service that returns errors
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

	// If configured to return errors, return error
	if m.shouldError {
		http.Error(w, m.errorMsg, http.StatusInternalServerError)
		return
	}

	// Parse request
	var req LLMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Store request for inspection
	for _, msg := range req.Messages {
		m.requests = append(m.requests, msg.Content)
	}

	// Extract instructions from the last user message
	var instructions string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			instructions = req.Messages[i].Content
			break
		}
	}

	// Generate mock synthesis based on instructions
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

// generateMockSynthesis generates predictable Ruby code based on instructions
func (m *MockLLMService) generateMockSynthesis(instructions string) string {
	lower := strings.ToLower(instructions)

	// Extract schedule pattern
	schedule := m.extractSchedule(lower)

	// Extract tools
	tools := m.extractTools(lower)

	// Generate agent code
	code := `require 'language_operator'

agent "test-agent" do
`

	// Add schedule if found
	if schedule != "" {
		code += fmt.Sprintf(`  schedule "%s"
`, schedule)
	}

	// Add workflow
	code += `  workflow do
`

	// Add tool usage steps
	for i, tool := range tools {
		code += fmt.Sprintf(`    step :step_%d, execute: -> {
      use_tool "%s"
    }
`, i+1, tool)
	}

	// Close workflow
	code += `  end
end
`

	return code
}

// extractSchedule extracts cron schedule from instructions
func (m *MockLLMService) extractSchedule(instructions string) string {
	// Check for minute intervals first (more specific)
	if strings.Contains(instructions, "5 minute") {
		return "*/5 * * * *"
	}
	if strings.Contains(instructions, "15 minute") {
		return "*/15 * * * *"
	}

	// Hourly patterns
	if strings.Contains(instructions, "hour") {
		return "0 * * * *"
	}

	// Daily patterns
	if strings.Contains(instructions, "daily") || strings.Contains(instructions, "every day") {
		if strings.Contains(instructions, "4pm") || strings.Contains(instructions, "16:00") {
			return "0 16 * * *"
		}
		if strings.Contains(instructions, "9am") || strings.Contains(instructions, "09:00") {
			return "0 9 * * *"
		}
		return "0 0 * * *" // midnight by default
	}

	// Weekly patterns
	if strings.Contains(instructions, "weekly") || strings.Contains(instructions, "every week") {
		return "0 0 * * 0" // Sunday midnight
	}

	return ""
}

// extractTools identifies tools from instructions
func (m *MockLLMService) extractTools(instructions string) []string {
	tools := []string{}

	toolKeywords := map[string]string{
		"spreadsheet": "google-sheets",
		"google sheets": "google-sheets",
		"email": "email",
		"send email": "email",
		"web": "web-fetch",
		"http": "web-fetch",
		"https": "web-fetch",
		"api": "web-fetch",
		"slack": "slack",
		"jira": "jira",
		"github": "github",
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

	// Default to at least one tool if none detected
	if len(tools) == 0 {
		tools = append(tools, "web-fetch")
	}

	return tools
}
