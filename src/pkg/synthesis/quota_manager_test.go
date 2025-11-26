package synthesis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
)

// TestQuotaManagerRaceConditions tests for data races in concurrent quota operations
func TestQuotaManagerRaceConditions(t *testing.T) {
	// This test is specifically designed to be run with: go test -race
	qm := NewQuotaManager(10.0, 100, "USD", testr.New(t))

	const (
		numGoroutines = 50
		numOperations = 100
		namespace     = "test-namespace"
	)

	// Start multiple goroutines performing concurrent quota operations
	var wg sync.WaitGroup

	// Goroutines reading quota (triggers GetRemainingQuota)
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				remainingCost, remainingAttempts := qm.GetRemainingQuota(namespace)
				// Validate returned values are reasonable
				if remainingCost < 0 || remainingCost > 10.0 {
					t.Errorf("Invalid remaining cost: %f", remainingCost)
				}
				if remainingAttempts < 0 || remainingAttempts > 100 {
					t.Errorf("Invalid remaining attempts: %d", remainingAttempts)
				}
			}
		}()
	}

	// Goroutines writing quota (triggers CheckCostQuota/RecordCost)
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Check quota
				if err := qm.CheckCostQuota(context.Background(), namespace, 0.1); err != nil {
					// Expected when quota is exceeded
					continue
				}

				// Record cost
				cost := &SynthesisCost{
					TotalCost: 0.1,
					Currency:  "USD",
				}
				qm.RecordCost(context.Background(), namespace, "test-agent", cost)

				// Record attempt
				qm.RecordAttempt(context.Background(), namespace, "test-agent", true, "")
			}
		}()
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify final state is consistent
	remainingCost, remainingAttempts := qm.GetRemainingQuota(namespace)
	if remainingCost < 0 || remainingAttempts < 0 {
		t.Errorf("Final state inconsistent: cost=%f, attempts=%d", remainingCost, remainingAttempts)
	}
}

// TestGetRemainingQuotaResetRace specifically tests the reset race condition that was fixed
func TestGetRemainingQuotaResetRace(t *testing.T) {
	qm := NewQuotaManager(10.0, 100, "USD", testr.New(t))
	namespace := "test-namespace"

	// Create a quota that needs reset
	quota := NewNamespaceQuota(namespace)
	quota.dailyCost = 5.0
	quota.dailyAttempts = 50
	// Set reset times in the past to trigger reset
	quota.dailyResetAt = time.Now().Add(-1 * time.Hour)
	quota.attemptsResetAt = time.Now().Add(-1 * time.Hour)

	qm.mu.Lock()
	qm.namespaceQuotas[namespace] = quota
	qm.mu.Unlock()

	const numGoroutines = 20
	var wg sync.WaitGroup

	// Multiple goroutines calling GetRemainingQuota simultaneously
	// This previously caused race conditions due to resetIfNeeded being called with RLock
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				remainingCost, remainingAttempts := qm.GetRemainingQuota(namespace)
				// After reset, should have full quota available
				if remainingCost < 0 || remainingCost > 10.0 {
					t.Errorf("Race condition detected: invalid remaining cost: %f", remainingCost)
				}
				if remainingAttempts < 0 || remainingAttempts > 100 {
					t.Errorf("Race condition detected: invalid remaining attempts: %d", remainingAttempts)
				}
			}
		}()
	}

	wg.Wait()

	// Verify quota was properly reset and values are consistent
	finalCost, finalAttempts := qm.GetRemainingQuota(namespace)
	// Should be close to full quota since reset occurred
	if finalCost < 9.0 || finalAttempts < 99 { // Allow for small discrepancies
		t.Errorf("Reset not properly handled: cost=%f, attempts=%d", finalCost, finalAttempts)
	}
}

// TestMixedReadWriteOperations tests concurrent read/write operations
func TestMixedReadWriteOperations(t *testing.T) {
	qm := NewQuotaManager(5.0, 50, "USD", testr.New(t))
	namespace := "test-namespace"

	const numGoroutines = 30
	var wg sync.WaitGroup

	// Reader goroutines
	for i := 0; i < numGoroutines/3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				qm.GetRemainingQuota(namespace)
				qm.GetNamespaceStats(namespace)
			}
		}()
	}

	// Writer goroutines (cost recording)
	for i := 0; i < numGoroutines/3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if err := qm.CheckCostQuota(context.Background(), namespace, 0.1); err == nil {
					cost := &SynthesisCost{TotalCost: 0.1, Currency: "USD"}
					qm.RecordCost(context.Background(), namespace, "test-agent", cost)
				}
			}
		}()
	}

	// Writer goroutines (attempt recording)
	for i := 0; i < numGoroutines/3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if err := qm.CheckAttemptQuota(context.Background(), namespace); err == nil {
					qm.RecordAttempt(context.Background(), namespace, "test-agent", true, "")
				}
			}
		}()
	}

	wg.Wait()

	// Verify consistency - no crashes and reasonable final state
	remainingCost, remainingAttempts := qm.GetRemainingQuota(namespace)
	currentCost, currentAttempts, exists := qm.GetNamespaceStats(namespace)

	if !exists {
		t.Error("Quota should exist after operations")
	}

	if remainingCost < 0 || remainingAttempts < 0 {
		t.Errorf("Negative quotas detected: cost=%f, attempts=%d", remainingCost, remainingAttempts)
	}

	if currentCost < 0 || currentAttempts < 0 {
		t.Errorf("Negative current usage: cost=%f, attempts=%d", currentCost, currentAttempts)
	}

	// Verify quota math consistency
	expectedRemainingCost := qm.maxCostPerNamespacePerDay - currentCost
	if expectedRemainingCost < 0 {
		expectedRemainingCost = 0
	}

	if abs(remainingCost-expectedRemainingCost) > 0.001 { // Allow for floating point precision
		t.Errorf("Quota math inconsistent: remaining=%f, expected=%f", remainingCost, expectedRemainingCost)
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestQuotaManagerBasicFunctionality tests basic quota operations for correctness
func TestQuotaManagerBasicFunctionality(t *testing.T) {
	qm := NewQuotaManager(5.0, 10, "USD", testr.New(t))
	namespace := "test-namespace"

	// Test initial state
	remainingCost, remainingAttempts := qm.GetRemainingQuota(namespace)
	if remainingCost != 5.0 || remainingAttempts != 10 {
		t.Errorf("Initial quota incorrect: cost=%f, attempts=%d", remainingCost, remainingAttempts)
	}

	// Test cost quota check and recording
	err := qm.CheckCostQuota(context.Background(), namespace, 2.0)
	if err != nil {
		t.Errorf("Should allow cost within quota: %v", err)
	}

	cost := &SynthesisCost{TotalCost: 2.0, Currency: "USD"}
	qm.RecordCost(context.Background(), namespace, "test-agent", cost)

	remainingCost, _ = qm.GetRemainingQuota(namespace)
	if remainingCost != 3.0 {
		t.Errorf("Remaining cost should be 3.0, got %f", remainingCost)
	}

	// Test attempt quota check and recording
	err = qm.CheckAttemptQuota(context.Background(), namespace)
	if err != nil {
		t.Errorf("Should allow attempt within quota: %v", err)
	}

	qm.RecordAttempt(context.Background(), namespace, "test-agent", true, "")

	_, remainingAttempts = qm.GetRemainingQuota(namespace)
	if remainingAttempts != 9 {
		t.Errorf("Remaining attempts should be 9, got %d", remainingAttempts)
	}

	// Test quota exceeded
	err = qm.CheckCostQuota(context.Background(), namespace, 4.0)
	if err == nil {
		t.Error("Should reject cost that exceeds quota")
	}
}

// Benchmarks to ensure the race condition fix doesn't significantly impact performance

// BenchmarkGetRemainingQuota measures performance of the main read operation
func BenchmarkGetRemainingQuota(b *testing.B) {
	qm := NewQuotaManager(100.0, 1000, "USD", testr.New(&testing.T{}))
	namespace := "benchmark-namespace"

	// Pre-populate with some data
	qm.RecordCost(context.Background(), namespace, "test-agent", &SynthesisCost{TotalCost: 5.0, Currency: "USD"})
	qm.RecordAttempt(context.Background(), namespace, "test-agent", true, "")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qm.GetRemainingQuota(namespace)
	}
}

// BenchmarkGetRemainingQuotaConcurrent measures performance under concurrent access
func BenchmarkGetRemainingQuotaConcurrent(b *testing.B) {
	qm := NewQuotaManager(100.0, 1000, "USD", testr.New(&testing.T{}))
	namespace := "benchmark-namespace"

	// Pre-populate with some data
	qm.RecordCost(context.Background(), namespace, "test-agent", &SynthesisCost{TotalCost: 5.0, Currency: "USD"})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			qm.GetRemainingQuota(namespace)
		}
	})
}

// BenchmarkMixedOperations measures performance of typical mixed workload
func BenchmarkMixedOperations(b *testing.B) {
	qm := NewQuotaManager(100.0, 1000, "USD", testr.New(&testing.T{}))
	namespace := "benchmark-namespace"

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				// 10% writes (quota checks and recordings)
				if err := qm.CheckCostQuota(context.Background(), namespace, 0.1); err == nil {
					qm.RecordCost(context.Background(), namespace, "test-agent", &SynthesisCost{TotalCost: 0.1, Currency: "USD"})
				}
			} else {
				// 90% reads
				qm.GetRemainingQuota(namespace)
			}
			i++
		}
	})
}

// BenchmarkQuotaReset measures performance impact of reset operations
func BenchmarkQuotaReset(b *testing.B) {
	qm := NewQuotaManager(100.0, 1000, "USD", testr.New(&testing.T{}))
	namespace := "benchmark-namespace"

	// Create quota that will need frequent resets
	quota := NewNamespaceQuota(namespace)
	quota.dailyCost = 50.0
	quota.dailyAttempts = 500

	qm.mu.Lock()
	qm.namespaceQuotas[namespace] = quota
	qm.mu.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Set reset time in past to trigger reset on every call
		quota.dailyResetAt = time.Now().Add(-1 * time.Minute)
		quota.attemptsResetAt = time.Now().Add(-1 * time.Minute)

		qm.GetRemainingQuota(namespace)
	}
}
