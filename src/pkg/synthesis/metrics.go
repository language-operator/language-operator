package synthesis

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// SynthesisRequestsTotal tracks total synthesis requests
	SynthesisRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "synthesis_requests_total",
			Help: "Total number of synthesis requests by namespace and status",
		},
		[]string{"namespace", "status"},
	)

	// SynthesisTokensUsed tracks token usage for synthesis
	SynthesisTokensUsed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "synthesis_tokens_used_total",
			Help: "Total number of tokens used in synthesis by namespace and type",
		},
		[]string{"namespace", "type"}, // type: input or output
	)

	// SynthesisCostUSD tracks synthesis costs in USD
	SynthesisCostUSD = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "synthesis_cost_usd_total",
			Help: "Total synthesis cost in USD by namespace",
		},
		[]string{"namespace"},
	)

	// SynthesisRateLimitExceeded tracks rate limit violations
	SynthesisRateLimitExceeded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "synthesis_rate_limit_exceeded_total",
			Help: "Total number of times synthesis rate limit was exceeded by namespace",
		},
		[]string{"namespace"},
	)

	// SynthesisQuotaExceeded tracks quota violations
	SynthesisQuotaExceeded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "synthesis_quota_exceeded_total",
			Help: "Total number of times synthesis quota was exceeded by namespace and type",
		},
		[]string{"namespace", "type"}, // type: cost or attempts
	)

	// SynthesisDuration tracks synthesis duration
	SynthesisDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "synthesis_duration_seconds",
			Help:    "Duration of synthesis operations in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~100s
		},
		[]string{"namespace", "status"},
	)

	// NamespaceQuotaRemaining tracks remaining quota per namespace
	NamespaceQuotaRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "synthesis_namespace_quota_remaining",
			Help: "Remaining synthesis quota for namespace by type",
		},
		[]string{"namespace", "type"}, // type: cost or attempts
	)

	// Learning-specific metrics for tracking organic function evolution

	// LearningTasksTotal tracks total number of tasks that have been learned
	LearningTasksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "learning_tasks_total",
			Help: "Total number of tasks learned by namespace, agent, and trigger type",
		},
		[]string{"namespace", "agent", "task_name", "trigger_type"}, // trigger_type: pattern_detection, error_recovery, manual
	)

	// LearningSuccessRate tracks the success rate of learning attempts
	LearningSuccessRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "learning_success_rate",
			Help: "Success rate of learning attempts by namespace and agent",
		},
		[]string{"namespace", "agent"},
	)

	// LearningCostSavingsUSD tracks cost savings from neural to symbolic transitions
	LearningCostSavingsUSD = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "learning_cost_savings_usd_total",
			Help: "Total cost savings in USD from learning optimizations by namespace and agent",
		},
		[]string{"namespace", "agent", "task_name"},
	)

	// ResynthesisTriggerReasons tracks reasons for re-synthesis events
	ResynthesisTriggerReasons = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resynthesis_trigger_reasons_total",
			Help: "Total number of re-synthesis triggers by namespace, agent, and reason",
		},
		[]string{"namespace", "agent", "reason"}, // reason: traces_accumulated, error_threshold, manual_trigger, consecutive_failures
	)

	// PatternConfidenceDistribution tracks pattern confidence scores
	PatternConfidenceDistribution = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pattern_confidence_distribution",
			Help:    "Distribution of pattern confidence scores by namespace and agent",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 1.0},
		},
		[]string{"namespace", "agent", "task_name"},
	)

	// LearningAttempts tracks total learning attempts (successful and failed)
	LearningAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "learning_attempts_total",
			Help: "Total number of learning attempts by namespace, agent, and status",
		},
		[]string{"namespace", "agent", "status"}, // status: success, failed
	)

	// TaskSymbolicConversions tracks neural to symbolic task conversions
	TaskSymbolicConversions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_symbolic_conversions_total",
			Help: "Total number of tasks converted from neural to symbolic by namespace and agent",
		},
		[]string{"namespace", "agent", "task_name"},
	)

	// ErrorTriggeredResynthesis tracks error-based re-synthesis events
	ErrorTriggeredResynthesis = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_triggered_resynthesis_total",
			Help: "Total number of error-triggered re-synthesis events by namespace and agent",
		},
		[]string{"namespace", "agent", "task_name", "error_type"},
	)

	// LearningCooldownViolations tracks attempts blocked by cooldown periods
	LearningCooldownViolations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "learning_cooldown_violations_total",
			Help: "Total number of learning attempts blocked by cooldown periods",
		},
		[]string{"namespace", "agent"},
	)
)

// init registers all synthesis metrics with the controller-runtime metrics registry
func init() {
	metrics.Registry.MustRegister(
		SynthesisRequestsTotal,
		SynthesisTokensUsed,
		SynthesisCostUSD,
		SynthesisRateLimitExceeded,
		SynthesisQuotaExceeded,
		SynthesisDuration,
		NamespaceQuotaRemaining,
		// Learning metrics
		LearningTasksTotal,
		LearningSuccessRate,
		LearningCostSavingsUSD,
		ResynthesisTriggerReasons,
		PatternConfidenceDistribution,
		LearningAttempts,
		TaskSymbolicConversions,
		ErrorTriggeredResynthesis,
		LearningCooldownViolations,
	)
}

// RecordSynthesisRequest records a synthesis request metric
func RecordSynthesisRequest(namespace, status string) {
	SynthesisRequestsTotal.WithLabelValues(namespace, status).Inc()
}

// RecordSynthesisTokens records token usage metrics
func RecordSynthesisTokens(namespace string, inputTokens, outputTokens int64) {
	SynthesisTokensUsed.WithLabelValues(namespace, "input").Add(float64(inputTokens))
	SynthesisTokensUsed.WithLabelValues(namespace, "output").Add(float64(outputTokens))
}

// RecordSynthesisCost records synthesis cost metric
func RecordSynthesisCost(namespace string, cost float64) {
	SynthesisCostUSD.WithLabelValues(namespace).Add(cost)
}

// RecordSynthesisRateLimitExceeded records rate limit violation
func RecordSynthesisRateLimitExceeded(namespace string) {
	SynthesisRateLimitExceeded.WithLabelValues(namespace).Inc()
}

// RecordSynthesisQuotaExceeded records quota violation
func RecordSynthesisQuotaExceeded(namespace, quotaType string) {
	SynthesisQuotaExceeded.WithLabelValues(namespace, quotaType).Inc()
}

// RecordSynthesisDuration records synthesis duration
func RecordSynthesisDuration(namespace, status string, duration float64) {
	SynthesisDuration.WithLabelValues(namespace, status).Observe(duration)
}

// UpdateNamespaceQuotaRemaining updates the remaining quota gauge
func UpdateNamespaceQuotaRemaining(namespace, quotaType string, remaining float64) {
	NamespaceQuotaRemaining.WithLabelValues(namespace, quotaType).Set(remaining)
}

// Learning metric recording functions

// RecordLearningTask records when a task has been successfully learned
func RecordLearningTask(namespace, agent, taskName, triggerType string) {
	LearningTasksTotal.WithLabelValues(namespace, agent, taskName, triggerType).Inc()
}

// UpdateLearningSuccessRate updates the success rate gauge for an agent
func UpdateLearningSuccessRate(namespace, agent string, successRate float64) {
	LearningSuccessRate.WithLabelValues(namespace, agent).Set(successRate)
}

// RecordLearningCostSavings records cost savings from neural to symbolic conversion
func RecordLearningCostSavings(namespace, agent, taskName string, savingsUSD float64) {
	LearningCostSavingsUSD.WithLabelValues(namespace, agent, taskName).Add(savingsUSD)
}

// RecordResynthesisTrigger records a re-synthesis trigger event
func RecordResynthesisTrigger(namespace, agent, reason string) {
	ResynthesisTriggerReasons.WithLabelValues(namespace, agent, reason).Inc()
}

// RecordPatternConfidence records a pattern confidence measurement
func RecordPatternConfidence(namespace, agent, taskName string, confidence float64) {
	PatternConfidenceDistribution.WithLabelValues(namespace, agent, taskName).Observe(confidence)
}

// RecordLearningAttempt records a learning attempt (success or failure)
func RecordLearningAttempt(namespace, agent, status string) {
	LearningAttempts.WithLabelValues(namespace, agent, status).Inc()
}

// RecordTaskSymbolicConversion records when a task is converted from neural to symbolic
func RecordTaskSymbolicConversion(namespace, agent, taskName string) {
	TaskSymbolicConversions.WithLabelValues(namespace, agent, taskName).Inc()
}

// RecordErrorTriggeredResynthesis records an error-triggered re-synthesis event
func RecordErrorTriggeredResynthesis(namespace, agent, taskName, errorType string) {
	ErrorTriggeredResynthesis.WithLabelValues(namespace, agent, taskName, errorType).Inc()
}

// RecordLearningCooldownViolation records when a learning attempt is blocked by cooldown
func RecordLearningCooldownViolation(namespace, agent string) {
	LearningCooldownViolations.WithLabelValues(namespace, agent).Inc()
}
