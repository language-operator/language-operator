package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MockChatModel adapts MockLLMService to synthesis.ChatModel interface
type MockChatModel struct {
	mockLLM *MockLLMService
}

// NewMockChatModel creates a ChatModel that uses the mock LLM service
func NewMockChatModel(mockLLM *MockLLMService) *MockChatModel {
	return &MockChatModel{
		mockLLM: mockLLM,
	}
}

// Generate implements the ChatModel interface
func (m *MockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// Convert eino messages to LLM API request format
	messages := make([]Message, len(input))
	for i, msg := range input {
		messages[i] = Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	req := LLMRequest{
		Model:    "test-model",
		Messages: messages,
	}

	// Send request to mock LLM service
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(m.mockLLM.URL(), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call mock LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mock LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Convert to eino message
	return &schema.Message{
		Role:    schema.RoleType(llmResp.Choices[0].Message.Role),
		Content: llmResp.Choices[0].Message.Content,
	}, nil
}
