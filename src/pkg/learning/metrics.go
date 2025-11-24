package learning

import (
	"context"
	"math"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/language-operator/language-operator/pkg/synthesis"
)

// MetricsCollector handles learning-specific metrics collection and aggregation
type MetricsCollector struct {
	log logr.Logger
}

// NewMetricsCollector creates a new learning metrics collector
func NewMetricsCollector(log logr.Logger) *MetricsCollector {
	return &MetricsCollector{
		log: log,
	}
}

var learningMetricsTracer = otel.Tracer("language-operator/learning-metrics")

// CostSavingsCalculator calculates cost savings from neural to symbolic transitions
type CostSavingsCalculator struct {
	neuralCostPerExecution   float64
	symbolicCostPerExecution float64
	executionsPerDay         int64
}

// CalculateDailySavings calculates the daily cost savings from converting a task to symbolic
func (c *CostSavingsCalculator) CalculateDailySavings() float64 {
	neuralDailyCost := c.neuralCostPerExecution * float64(c.executionsPerDay)
	symbolicDailyCost := c.symbolicCostPerExecution * float64(c.executionsPerDay)
	return math.Max(0, neuralDailyCost-symbolicDailyCost)
}

// CalculateProjectedMonthlySavings projects monthly savings from the conversion
func (c *CostSavingsCalculator) CalculateProjectedMonthlySavings() float64 {
	dailySavings := c.CalculateDailySavings()
	return dailySavings * 30 // 30 days
}

// EstimateCostSavings estimates cost savings from a neural to symbolic conversion
func (mc *MetricsCollector) EstimateCostSavings(ctx context.Context, namespace, agent, taskName string, neuralCost, symbolicCost float64, executionFrequency int64) float64 {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.estimate_cost_savings")
	defer span.End()

	calculator := &CostSavingsCalculator{
		neuralCostPerExecution:   neuralCost,
		symbolicCostPerExecution: symbolicCost,
		executionsPerDay:         executionFrequency,
	}

	dailySavings := calculator.CalculateDailySavings()

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
		attribute.String("learning.task_name", taskName),
		attribute.Float64("learning.neural_cost", neuralCost),
		attribute.Float64("learning.symbolic_cost", symbolicCost),
		attribute.Int64("learning.execution_frequency", executionFrequency),
		attribute.Float64("learning.daily_savings", dailySavings),
	)

	mc.log.V(1).Info("Calculated cost savings for task conversion",
		"namespace", namespace,
		"agent", agent,
		"task", taskName,
		"daily_savings_usd", dailySavings,
		"execution_frequency", executionFrequency)

	return dailySavings
}

// LearningSuccessRateAggregator tracks and calculates learning success rates
type LearningSuccessRateAggregator struct {
	namespace            string
	agent                string
	totalAttempts        int64
	successfulAttempts   int64
	windowStart          time.Time
	windowDurationHours  int
}

// NewLearningSuccessRateAggregator creates a new success rate aggregator
func NewLearningSuccessRateAggregator(namespace, agent string, windowHours int) *LearningSuccessRateAggregator {
	return &LearningSuccessRateAggregator{
		namespace:           namespace,
		agent:              agent,
		windowStart:        time.Now(),
		windowDurationHours: windowHours,
	}
}

// AddAttempt records a learning attempt
func (a *LearningSuccessRateAggregator) AddAttempt(successful bool) {
	a.totalAttempts++
	if successful {
		a.successfulAttempts++
	}
}

// CalculateSuccessRate calculates the current success rate
func (a *LearningSuccessRateAggregator) CalculateSuccessRate() float64 {
	if a.totalAttempts == 0 {
		return 0.0
	}
	return float64(a.successfulAttempts) / float64(a.totalAttempts)
}

// ShouldReset checks if the window should be reset
func (a *LearningSuccessRateAggregator) ShouldReset() bool {
	return time.Since(a.windowStart) > time.Duration(a.windowDurationHours)*time.Hour
}

// Reset resets the aggregator window
func (a *LearningSuccessRateAggregator) Reset() {
	a.totalAttempts = 0
	a.successfulAttempts = 0
	a.windowStart = time.Now()
}

// UpdateLearningSuccessRates updates success rate metrics for agents
func (mc *MetricsCollector) UpdateLearningSuccessRates(ctx context.Context, namespace, agent string, aggregator *LearningSuccessRateAggregator) {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.update_success_rates")
	defer span.End()

	successRate := aggregator.CalculateSuccessRate()

	synthesis.UpdateLearningSuccessRate(namespace, agent, successRate)

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
		attribute.Float64("learning.success_rate", successRate),
		attribute.Int64("learning.total_attempts", aggregator.totalAttempts),
		attribute.Int64("learning.successful_attempts", aggregator.successfulAttempts),
	)

	mc.log.V(1).Info("Updated learning success rate",
		"namespace", namespace,
		"agent", agent,
		"success_rate", successRate,
		"total_attempts", aggregator.totalAttempts,
		"successful_attempts", aggregator.successfulAttempts)
}

// TriggerReasonCategorizer categorizes re-synthesis trigger reasons
type TriggerReasonCategorizer struct{}

// CategorizeReason categorizes a trigger reason for metrics consistency
func (trc *TriggerReasonCategorizer) CategorizeReason(originalReason string) string {
	switch originalReason {
	case "traces_accumulated":
		return "pattern_detection"
	case "error_threshold":
		return "error_rate_high"
	case "consecutive_failures":
		return "error_recovery"
	case "manual_trigger":
		return "manual"
	case "scheduled_learning":
		return "scheduled"
	default:
		return "other"
	}
}

// RecordCategorizedTrigger records a trigger with consistent categorization
func (mc *MetricsCollector) RecordCategorizedTrigger(ctx context.Context, namespace, agent, originalReason string) {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.record_categorized_trigger")
	defer span.End()

	categorizer := &TriggerReasonCategorizer{}
	categorizedReason := categorizer.CategorizeReason(originalReason)

	synthesis.RecordResynthesisTrigger(namespace, agent, categorizedReason)

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
		attribute.String("learning.original_reason", originalReason),
		attribute.String("learning.categorized_reason", categorizedReason),
	)

	mc.log.V(1).Info("Recorded categorized trigger",
		"namespace", namespace,
		"agent", agent,
		"original_reason", originalReason,
		"categorized_reason", categorizedReason)
}

// PatternConfidenceTracker tracks pattern confidence metrics
type PatternConfidenceTracker struct {
	namespace   string
	agent       string
	taskName    string
	confidence  float64
	timestamp   time.Time
}

// NewPatternConfidenceTracker creates a new confidence tracker
func NewPatternConfidenceTracker(namespace, agent, taskName string, confidence float64) *PatternConfidenceTracker {
	return &PatternConfidenceTracker{
		namespace:  namespace,
		agent:     agent,
		taskName:  taskName,
		confidence: confidence,
		timestamp: time.Now(),
	}
}

// IsConfidenceHigh checks if confidence is above threshold
func (pct *PatternConfidenceTracker) IsConfidenceHigh(threshold float64) bool {
	return pct.confidence >= threshold
}

// IsConfidenceLow checks if confidence is below threshold
func (pct *PatternConfidenceTracker) IsConfidenceLow(threshold float64) bool {
	return pct.confidence < threshold
}

// GetConfidenceCategory returns a category for the confidence level
func (pct *PatternConfidenceTracker) GetConfidenceCategory() string {
	switch {
	case pct.confidence >= 0.9:
		return "very_high"
	case pct.confidence >= 0.8:
		return "high"
	case pct.confidence >= 0.6:
		return "medium"
	case pct.confidence >= 0.4:
		return "low"
	default:
		return "very_low"
	}
}

// RecordPatternConfidenceMetrics records pattern confidence with additional context
func (mc *MetricsCollector) RecordPatternConfidenceMetrics(ctx context.Context, tracker *PatternConfidenceTracker) {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.record_pattern_confidence")
	defer span.End()

	synthesis.RecordPatternConfidence(tracker.namespace, tracker.agent, tracker.taskName, tracker.confidence)

	span.SetAttributes(
		attribute.String("learning.namespace", tracker.namespace),
		attribute.String("learning.agent", tracker.agent),
		attribute.String("learning.task_name", tracker.taskName),
		attribute.Float64("learning.confidence", tracker.confidence),
		attribute.String("learning.confidence_category", tracker.GetConfidenceCategory()),
	)

	mc.log.V(1).Info("Recorded pattern confidence metrics",
		"namespace", tracker.namespace,
		"agent", tracker.agent,
		"task", tracker.taskName,
		"confidence", tracker.confidence,
		"category", tracker.GetConfidenceCategory())
}

// LearningEventProcessor processes and records learning-related events with proper metrics
type LearningEventProcessor struct {
	metricsCollector *MetricsCollector
}

// NewLearningEventProcessor creates a new learning event processor
func NewLearningEventProcessor(collector *MetricsCollector) *LearningEventProcessor {
	return &LearningEventProcessor{
		metricsCollector: collector,
	}
}

// ProcessTaskLearned processes a task learned event with comprehensive metrics
func (lep *LearningEventProcessor) ProcessTaskLearned(ctx context.Context, namespace, agent, taskName, triggerType string, confidence float64, costSavings float64) error {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.process_task_learned")
	defer span.End()

	// Record the main learning task metric
	synthesis.RecordLearningTask(namespace, agent, taskName, triggerType)
	synthesis.RecordLearningAttempt(namespace, agent, "success")
	synthesis.RecordTaskSymbolicConversion(namespace, agent, taskName)

	// Record pattern confidence
	confidenceTracker := NewPatternConfidenceTracker(namespace, agent, taskName, confidence)
	lep.metricsCollector.RecordPatternConfidenceMetrics(ctx, confidenceTracker)

	// Record cost savings if applicable
	if costSavings > 0 {
		synthesis.RecordLearningCostSavings(namespace, agent, taskName, costSavings)
	}

	// Record categorized trigger
	lep.metricsCollector.RecordCategorizedTrigger(ctx, namespace, agent, triggerType)

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
		attribute.String("learning.task_name", taskName),
		attribute.String("learning.trigger_type", triggerType),
		attribute.Float64("learning.confidence", confidence),
		attribute.Float64("learning.cost_savings", costSavings),
		attribute.Bool("learning.success", true),
	)

	return nil
}

// ProcessLearningFailure processes a learning failure event
func (lep *LearningEventProcessor) ProcessLearningFailure(ctx context.Context, namespace, agent, taskName, reason string) error {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.process_learning_failure")
	defer span.End()

	synthesis.RecordLearningAttempt(namespace, agent, "failed")

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
		attribute.String("learning.task_name", taskName),
		attribute.String("learning.failure_reason", reason),
		attribute.Bool("learning.success", false),
	)

	return nil
}

// ProcessErrorTriggeredResynthesis processes an error-triggered re-synthesis event
func (lep *LearningEventProcessor) ProcessErrorTriggeredResynthesis(ctx context.Context, namespace, agent, taskName, errorType string) error {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.process_error_triggered_resynthesis")
	defer span.End()

	synthesis.RecordErrorTriggeredResynthesis(namespace, agent, taskName, errorType)
	lep.metricsCollector.RecordCategorizedTrigger(ctx, namespace, agent, "error_recovery")

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
		attribute.String("learning.task_name", taskName),
		attribute.String("learning.error_type", errorType),
	)

	return nil
}

// ProcessCooldownViolation processes a learning cooldown violation event
func (lep *LearningEventProcessor) ProcessCooldownViolation(ctx context.Context, namespace, agent string) error {
	ctx, span := learningMetricsTracer.Start(ctx, "learning.process_cooldown_violation")
	defer span.End()

	synthesis.RecordLearningCooldownViolation(namespace, agent)

	span.SetAttributes(
		attribute.String("learning.namespace", namespace),
		attribute.String("learning.agent", agent),
	)

	return nil
}

// HealthMetrics provides health-related metrics calculations
type HealthMetrics struct {
	namespace string
	agent     string
}

// NewHealthMetrics creates a new health metrics calculator
func NewHealthMetrics(namespace, agent string) *HealthMetrics {
	return &HealthMetrics{
		namespace: namespace,
		agent:     agent,
	}
}

// CalculateOverallLearningHealth calculates an overall learning health score
func (hm *HealthMetrics) CalculateOverallLearningHealth(successRate, avgConfidence float64, errorRate float64) float64 {
	// Weighted health score: success rate (40%), confidence (30%), inverse error rate (30%)
	healthScore := (successRate * 0.4) + (avgConfidence * 0.3) + ((1.0 - errorRate) * 0.3)
	
	// Ensure score is between 0 and 1
	if healthScore > 1.0 {
		healthScore = 1.0
	}
	if healthScore < 0.0 {
		healthScore = 0.0
	}
	
	return healthScore
}

// GetHealthCategory returns a health category string
func (hm *HealthMetrics) GetHealthCategory(healthScore float64) string {
	switch {
	case healthScore >= 0.8:
		return "excellent"
	case healthScore >= 0.6:
		return "good"
	case healthScore >= 0.4:
		return "fair"
	case healthScore >= 0.2:
		return "poor"
	default:
		return "critical"
	}
}