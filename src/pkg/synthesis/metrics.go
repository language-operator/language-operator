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
