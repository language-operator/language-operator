package learning

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsCollector_EstimateCostSavings(t *testing.T) {
	tests := []struct {
		name              string
		neuralCost        float64
		symbolicCost      float64
		executionFreq     int64
		expectedSavings   float64
	}{
		{
			name:              "significant savings",
			neuralCost:        0.01,   // $0.01 per neural execution
			symbolicCost:      0.0001, // $0.0001 per symbolic execution
			executionFreq:     10,     // 10 executions per day
			expectedSavings:   0.099,  // (0.01 - 0.0001) * 10 = $0.099 per day
		},
		{
			name:              "minimal savings",
			neuralCost:        0.001,
			symbolicCost:      0.0005,
			executionFreq:     5,
			expectedSavings:   0.0025, // (0.001 - 0.0005) * 5 = $0.0025 per day
		},
		{
			name:              "no savings - costs equal",
			neuralCost:        0.01,
			symbolicCost:      0.01,
			executionFreq:     10,
			expectedSavings:   0.0, // No savings when costs are equal
		},
		{
			name:              "negative savings",
			neuralCost:        0.001,
			symbolicCost:      0.01,
			executionFreq:     10,
			expectedSavings:   0.0, // Max with 0 prevents negative savings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewMetricsCollector(testr.New(t))
			
			savings := collector.EstimateCostSavings(
				context.Background(),
				"test-namespace",
				"test-agent", 
				"test-task",
				tt.neuralCost,
				tt.symbolicCost,
				tt.executionFreq,
			)
			
			assert.InDelta(t, tt.expectedSavings, savings, 0.0001)
		})
	}
}

func TestLearningSuccessRateAggregator(t *testing.T) {
	aggregator := NewLearningSuccessRateAggregator("test-namespace", "test-agent", 24)
	
	// Initially should have 0% success rate
	assert.Equal(t, 0.0, aggregator.CalculateSuccessRate())
	
	// Add some attempts
	aggregator.AddAttempt(true)  // success
	aggregator.AddAttempt(false) // failure  
	aggregator.AddAttempt(true)  // success
	aggregator.AddAttempt(true)  // success
	
	// Should be 75% success rate (3 out of 4)
	expectedRate := 0.75
	assert.InDelta(t, expectedRate, aggregator.CalculateSuccessRate(), 0.001)
	
	// Check reset functionality
	assert.False(t, aggregator.ShouldReset()) // Should not reset immediately
	
	// Simulate window expiration
	aggregator.windowStart = time.Now().Add(-25 * time.Hour) // 25 hours ago
	assert.True(t, aggregator.ShouldReset())
	
	aggregator.Reset()
	assert.Equal(t, int64(0), aggregator.totalAttempts)
	assert.Equal(t, int64(0), aggregator.successfulAttempts)
	assert.Equal(t, 0.0, aggregator.CalculateSuccessRate())
}

func TestTriggerReasonCategorizer(t *testing.T) {
	categorizer := &TriggerReasonCategorizer{}
	
	tests := []struct {
		originalReason     string
		expectedCategory   string
	}{
		{"traces_accumulated", "pattern_detection"},
		{"error_threshold", "error_rate_high"},
		{"consecutive_failures", "error_recovery"},
		{"manual_trigger", "manual"},
		{"scheduled_learning", "scheduled"},
		{"unknown_reason", "other"},
		{"", "other"},
	}
	
	for _, tt := range tests {
		t.Run(tt.originalReason, func(t *testing.T) {
			category := categorizer.CategorizeReason(tt.originalReason)
			assert.Equal(t, tt.expectedCategory, category)
		})
	}
}

func TestPatternConfidenceTracker(t *testing.T) {
	tracker := NewPatternConfidenceTracker("test-ns", "test-agent", "test-task", 0.85)
	
	// Test confidence thresholds
	assert.True(t, tracker.IsConfidenceHigh(0.8))
	assert.False(t, tracker.IsConfidenceLow(0.8))
	assert.False(t, tracker.IsConfidenceHigh(0.9))
	assert.True(t, tracker.IsConfidenceLow(0.9))
	
	// Test confidence categories
	tests := []struct {
		confidence float64
		category   string
	}{
		{0.95, "very_high"},
		{0.85, "high"},
		{0.75, "medium"},    // 0.75 >= 0.6, so "medium"
		{0.55, "low"},       // 0.55 < 0.6 but >= 0.4, so "low"
		{0.45, "low"},       // 0.45 >= 0.4 and < 0.6, so "low"
		{0.35, "very_low"},  // 0.35 < 0.4, so "very_low"
		{0.25, "very_low"},
	}
	
	for _, tt := range tests {
		tracker.confidence = tt.confidence
		assert.Equal(t, tt.category, tracker.GetConfidenceCategory())
	}
}

func TestLearningEventProcessor_ProcessTaskLearned(t *testing.T) {
	collector := NewMetricsCollector(testr.New(t))
	processor := NewLearningEventProcessor(collector)
	
	ctx := context.Background()
	
	err := processor.ProcessTaskLearned(
		ctx,
		"test-namespace",
		"test-agent", 
		"test-task",
		"pattern_detection",
		0.85,
		0.50, // $0.50 cost savings
	)
	
	require.NoError(t, err)
}

func TestLearningEventProcessor_ProcessLearningFailure(t *testing.T) {
	collector := NewMetricsCollector(testr.New(t))
	processor := NewLearningEventProcessor(collector)
	
	ctx := context.Background()
	
	err := processor.ProcessLearningFailure(
		ctx,
		"test-namespace",
		"test-agent",
		"test-task",
		"synthesis failed",
	)
	
	require.NoError(t, err)
}

func TestLearningEventProcessor_ProcessErrorTriggeredResynthesis(t *testing.T) {
	collector := NewMetricsCollector(testr.New(t))
	processor := NewLearningEventProcessor(collector)
	
	ctx := context.Background()
	
	err := processor.ProcessErrorTriggeredResynthesis(
		ctx,
		"test-namespace",
		"test-agent",
		"test-task",
		"connection_timeout",
	)
	
	require.NoError(t, err)
}

func TestHealthMetrics_CalculateOverallLearningHealth(t *testing.T) {
	healthMetrics := NewHealthMetrics("test-namespace", "test-agent")
	
	tests := []struct {
		name         string
		successRate  float64
		avgConfidence float64
		errorRate    float64
		expectedMin  float64
		expectedMax  float64
		expectedCategory string
	}{
		{
			name:         "excellent health",
			successRate:  0.95,
			avgConfidence: 0.90,
			errorRate:    0.02,
			expectedMin:  0.8,
			expectedMax:  1.0,
			expectedCategory: "excellent",
		},
		{
			name:         "good health",
			successRate:  0.70,
			avgConfidence: 0.65,
			errorRate:    0.15,
			expectedMin:  0.6,
			expectedMax:  0.8,
			expectedCategory: "good",
		},
		{
			name:         "poor health", 
			successRate:  0.25,
			avgConfidence: 0.35,
			errorRate:    0.50,
			expectedMin:  0.2,
			expectedMax:  0.4,
			expectedCategory: "poor",
		},
		{
			name:         "critical health",
			successRate:  0.10,
			avgConfidence: 0.20,
			errorRate:    0.80,
			expectedMin:  0.0,
			expectedMax:  0.2,
			expectedCategory: "critical",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthScore := healthMetrics.CalculateOverallLearningHealth(
				tt.successRate,
				tt.avgConfidence,
				tt.errorRate,
			)
			
			assert.GreaterOrEqual(t, healthScore, tt.expectedMin)
			assert.LessOrEqual(t, healthScore, tt.expectedMax)
			assert.Equal(t, tt.expectedCategory, healthMetrics.GetHealthCategory(healthScore))
		})
	}
}

func TestCostSavingsCalculator_ProjectedSavings(t *testing.T) {
	calculator := &CostSavingsCalculator{
		neuralCostPerExecution:   0.01,
		symbolicCostPerExecution: 0.0001,
		executionsPerDay:         20,
	}
	
	dailySavings := calculator.CalculateDailySavings()
	expectedDaily := (0.01 - 0.0001) * 20 // $0.198 per day
	assert.InDelta(t, expectedDaily, dailySavings, 0.001)
	
	monthlySavings := calculator.CalculateProjectedMonthlySavings()
	expectedMonthly := expectedDaily * 30 // $5.94 per month
	assert.InDelta(t, expectedMonthly, monthlySavings, 0.001)
}