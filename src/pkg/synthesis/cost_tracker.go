package synthesis

import (
	"fmt"
	"math"
	"time"

	v1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CostTracker tracks token usage and costs for LLM API calls
type CostTracker struct {
	inputTokenCost  float64 // cost per 1K input tokens
	outputTokenCost float64 // cost per 1K output tokens
	currency        string
}

// NewCostTracker creates a new cost tracker from model pricing
func NewCostTracker(model *v1alpha1.LanguageModel) *CostTracker {
	if model == nil || model.Spec.CostTracking == nil || !model.Spec.CostTracking.Enabled {
		// Return tracker with zero costs if cost tracking disabled
		return &CostTracker{
			inputTokenCost:  0,
			outputTokenCost: 0,
			currency:        "USD",
		}
	}

	inputCost := 0.0
	if model.Spec.CostTracking.InputTokenCost != nil {
		inputCost = *model.Spec.CostTracking.InputTokenCost
	}

	outputCost := 0.0
	if model.Spec.CostTracking.OutputTokenCost != nil {
		outputCost = *model.Spec.CostTracking.OutputTokenCost
	}

	return &CostTracker{
		inputTokenCost:  inputCost,
		outputTokenCost: outputCost,
		currency:        model.Spec.CostTracking.Currency,
	}
}

// SynthesisCost represents the cost of a single synthesis operation
type SynthesisCost struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	InputCost    float64
	OutputCost   float64
	TotalCost    float64
	Currency     string
	Timestamp    time.Time
	ModelName    string
}

// CalculateCost computes the cost from token counts
func (ct *CostTracker) CalculateCost(inputTokens, outputTokens int64, modelName string) *SynthesisCost {
	inputCost := (float64(inputTokens) / 1000.0) * ct.inputTokenCost
	outputCost := (float64(outputTokens) / 1000.0) * ct.outputTokenCost

	return &SynthesisCost{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    inputCost + outputCost,
		Currency:     ct.currency,
		Timestamp:    time.Now(),
		ModelName:    modelName,
	}
}

// EstimateTokens estimates token count from text (approximate: 1 token ~= 4 chars)
// This is a rough estimate for cost prediction before API calls
func EstimateTokens(text string) int64 {
	// Average: 1 token = 4 characters for English text
	// Add 10% buffer for safety
	estimated := int64(math.Ceil(float64(len(text)) / 4.0 * 1.1))
	return estimated
}

// EstimateCost estimates the cost of a synthesis operation before making the API call
func (ct *CostTracker) EstimateCost(promptText string, expectedOutputTokens int64, modelName string) *SynthesisCost {
	inputTokens := EstimateTokens(promptText)
	return ct.CalculateCost(inputTokens, expectedOutputTokens, modelName)
}

// ToAgentCostMetrics converts SynthesisCost to CRD cost metrics format
func (sc *SynthesisCost) ToAgentCostMetrics() *v1alpha1.AgentCostMetrics {
	now := metav1.Now()
	return &v1alpha1.AgentCostMetrics{
		TotalCost: &sc.TotalCost,
		Currency:  sc.Currency,
		ModelCosts: []v1alpha1.ModelCostSpec{
			{
				ModelName: sc.ModelName,
				Cost:      sc.TotalCost,
			},
		},
		LastReset: &now,
	}
}

// String returns a human-readable cost summary
func (sc *SynthesisCost) String() string {
	return fmt.Sprintf("Synthesis cost: %.4f %s (input: %d tokens / %.4f %s, output: %d tokens / %.4f %s)",
		sc.TotalCost, sc.Currency,
		sc.InputTokens, sc.InputCost, sc.Currency,
		sc.OutputTokens, sc.OutputCost, sc.Currency)
}

// ExceedsBudget checks if the cost exceeds the given budget
func (sc *SynthesisCost) ExceedsBudget(maxCost float64) bool {
	return sc.TotalCost > maxCost
}
