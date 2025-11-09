package synthesis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// QuotaManager tracks synthesis quotas and costs per namespace
type QuotaManager struct {
	mu sync.RWMutex

	// Per-namespace tracking
	namespaceQuotas map[string]*NamespaceQuota

	// Global limits
	maxCostPerNamespacePerDay float64
	maxAttemptsPerDay         int
	currency                  string
	log                       logr.Logger
}

// NamespaceQuota tracks quota usage for a single namespace
type NamespaceQuota struct {
	Namespace string

	// Daily cost tracking
	dailyCost    float64
	dailyResetAt time.Time
	costHistory  []CostEntry

	// Daily attempt tracking
	dailyAttempts   int
	attemptsResetAt time.Time
	attemptHistory  []AttemptEntry

	mu sync.RWMutex
}

// CostEntry represents a single synthesis cost record
type CostEntry struct {
	Timestamp time.Time
	Cost      float64
	AgentName string
	Currency  string
}

// AttemptEntry represents a single synthesis attempt
type AttemptEntry struct {
	Timestamp time.Time
	AgentName string
	Success   bool
	ErrorMsg  string
}

// NewQuotaManager creates a new quota manager
func NewQuotaManager(maxCostPerDay float64, maxAttemptsPerDay int, currency string, log logr.Logger) *QuotaManager {
	return &QuotaManager{
		namespaceQuotas:           make(map[string]*NamespaceQuota),
		maxCostPerNamespacePerDay: maxCostPerDay,
		maxAttemptsPerDay:         maxAttemptsPerDay,
		currency:                  currency,
		log:                       log,
	}
}

// NewNamespaceQuota creates a new namespace quota tracker
func NewNamespaceQuota(namespace string) *NamespaceQuota {
	now := time.Now()
	return &NamespaceQuota{
		Namespace:       namespace,
		dailyCost:       0,
		dailyResetAt:    now.Add(24 * time.Hour),
		costHistory:     make([]CostEntry, 0),
		dailyAttempts:   0,
		attemptsResetAt: now.Add(24 * time.Hour),
		attemptHistory:  make([]AttemptEntry, 0),
	}
}

// CheckCostQuota checks if synthesis would exceed cost quota
func (qm *QuotaManager) CheckCostQuota(ctx context.Context, namespace string, estimatedCost float64) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Get or create quota tracker for namespace
	quota, exists := qm.namespaceQuotas[namespace]
	if !exists {
		quota = NewNamespaceQuota(namespace)
		qm.namespaceQuotas[namespace] = quota
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// Reset daily counters if needed
	quota.resetIfNeeded()

	// Check if adding this cost would exceed quota
	projectedCost := quota.dailyCost + estimatedCost
	if projectedCost > qm.maxCostPerNamespacePerDay {
		qm.log.Info("Cost quota would be exceeded",
			"namespace", namespace,
			"currentCost", quota.dailyCost,
			"estimatedCost", estimatedCost,
			"projectedCost", projectedCost,
			"limit", qm.maxCostPerNamespacePerDay,
			"currency", qm.currency)

		return fmt.Errorf("synthesis cost quota exceeded for namespace %s: current %.4f + estimated %.4f = %.4f > limit %.4f %s (resets at %s)",
			namespace,
			quota.dailyCost,
			estimatedCost,
			projectedCost,
			qm.maxCostPerNamespacePerDay,
			qm.currency,
			quota.dailyResetAt.Format(time.RFC3339))
	}

	return nil
}

// CheckAttemptQuota checks if synthesis would exceed attempt quota
func (qm *QuotaManager) CheckAttemptQuota(ctx context.Context, namespace string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Get or create quota tracker for namespace
	quota, exists := qm.namespaceQuotas[namespace]
	if !exists {
		quota = NewNamespaceQuota(namespace)
		qm.namespaceQuotas[namespace] = quota
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// Reset daily counters if needed
	quota.resetIfNeeded()

	// Check if we've hit the attempt limit
	if quota.dailyAttempts >= qm.maxAttemptsPerDay {
		qm.log.Info("Attempt quota exceeded",
			"namespace", namespace,
			"attempts", quota.dailyAttempts,
			"limit", qm.maxAttemptsPerDay)

		return fmt.Errorf("synthesis attempt quota exceeded for namespace %s: %d attempts today, limit is %d (resets at %s)",
			namespace,
			quota.dailyAttempts,
			qm.maxAttemptsPerDay,
			quota.attemptsResetAt.Format(time.RFC3339))
	}

	return nil
}

// RecordCost records an actual synthesis cost
func (qm *QuotaManager) RecordCost(ctx context.Context, namespace, agentName string, cost *SynthesisCost) error {
	if cost == nil {
		return nil // Cost tracking disabled
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Get or create quota tracker
	quota, exists := qm.namespaceQuotas[namespace]
	if !exists {
		quota = NewNamespaceQuota(namespace)
		qm.namespaceQuotas[namespace] = quota
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// Reset daily counters if needed
	quota.resetIfNeeded()

	// Record the cost
	quota.dailyCost += cost.TotalCost
	quota.costHistory = append(quota.costHistory, CostEntry{
		Timestamp: time.Now(),
		Cost:      cost.TotalCost,
		AgentName: agentName,
		Currency:  cost.Currency,
	})

	qm.log.Info("Synthesis cost recorded",
		"namespace", namespace,
		"agent", agentName,
		"cost", cost.TotalCost,
		"currency", cost.Currency,
		"dailyTotal", quota.dailyCost,
		"limit", qm.maxCostPerNamespacePerDay)

	return nil
}

// RecordAttempt records a synthesis attempt
func (qm *QuotaManager) RecordAttempt(ctx context.Context, namespace, agentName string, success bool, errorMsg string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Get or create quota tracker
	quota, exists := qm.namespaceQuotas[namespace]
	if !exists {
		quota = NewNamespaceQuota(namespace)
		qm.namespaceQuotas[namespace] = quota
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// Reset daily counters if needed
	quota.resetIfNeeded()

	// Record the attempt
	quota.dailyAttempts++
	quota.attemptHistory = append(quota.attemptHistory, AttemptEntry{
		Timestamp: time.Now(),
		AgentName: agentName,
		Success:   success,
		ErrorMsg:  errorMsg,
	})

	qm.log.V(1).Info("Synthesis attempt recorded",
		"namespace", namespace,
		"agent", agentName,
		"success", success,
		"dailyAttempts", quota.dailyAttempts,
		"limit", qm.maxAttemptsPerDay)
}

// GetNamespaceStats returns current quota statistics for a namespace
func (qm *QuotaManager) GetNamespaceStats(namespace string) (dailyCost float64, dailyAttempts int, exists bool) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	quota, exists := qm.namespaceQuotas[namespace]
	if !exists {
		return 0, 0, false
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	return quota.dailyCost, quota.dailyAttempts, true
}

// GetRemainingQuota returns remaining budget for a namespace
func (qm *QuotaManager) GetRemainingQuota(namespace string) (remainingCost float64, remainingAttempts int) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	quota, exists := qm.namespaceQuotas[namespace]
	if !exists {
		return qm.maxCostPerNamespacePerDay, qm.maxAttemptsPerDay
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	quota.resetIfNeeded()

	remainingCost = qm.maxCostPerNamespacePerDay - quota.dailyCost
	if remainingCost < 0 {
		remainingCost = 0
	}

	remainingAttempts = qm.maxAttemptsPerDay - quota.dailyAttempts
	if remainingAttempts < 0 {
		remainingAttempts = 0
	}

	return remainingCost, remainingAttempts
}

// resetIfNeeded resets daily counters if the reset time has passed
// Must be called with quota.mu locked
func (nq *NamespaceQuota) resetIfNeeded() {
	now := time.Now()

	// Reset cost counter if needed
	if now.After(nq.dailyResetAt) {
		nq.dailyCost = 0
		nq.dailyResetAt = now.Add(24 * time.Hour)
		// Keep last 7 days of history
		cutoff := now.Add(-7 * 24 * time.Hour)
		nq.costHistory = filterCostHistory(nq.costHistory, cutoff)
	}

	// Reset attempt counter if needed
	if now.After(nq.attemptsResetAt) {
		nq.dailyAttempts = 0
		nq.attemptsResetAt = now.Add(24 * time.Hour)
		// Keep last 7 days of history
		cutoff := now.Add(-7 * 24 * time.Hour)
		nq.attemptHistory = filterAttemptHistory(nq.attemptHistory, cutoff)
	}
}

// filterCostHistory removes entries older than cutoff
func filterCostHistory(history []CostEntry, cutoff time.Time) []CostEntry {
	filtered := make([]CostEntry, 0, len(history))
	for _, entry := range history {
		if entry.Timestamp.After(cutoff) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// filterAttemptHistory removes entries older than cutoff
func filterAttemptHistory(history []AttemptEntry, cutoff time.Time) []AttemptEntry {
	filtered := make([]AttemptEntry, 0, len(history))
	for _, entry := range history {
		if entry.Timestamp.After(cutoff) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// Reset clears all quota state (useful for testing)
func (qm *QuotaManager) Reset() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.namespaceQuotas = make(map[string]*NamespaceQuota)
}
