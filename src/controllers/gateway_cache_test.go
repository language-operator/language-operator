package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

func TestGatewayAPICaching(t *testing.T) {
	// Create a basic reconciler with cache
	scheme := runtime.NewScheme()
	_ = langopv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()

	t.Run("Cache TTL Behavior", func(t *testing.T) {
		// First call - cache miss, should perform discovery
		result1, err1 := reconciler.hasGatewayAPI(ctx)
		if err1 != nil {
			t.Fatalf("First call failed: %v", err1)
		}

		// Record the check time
		firstCheck := reconciler.gatewayCache.lastCheck

		// Immediate second call - should use cache
		result2, err2 := reconciler.hasGatewayAPI(ctx)
		if err2 != nil {
			t.Fatalf("Second call failed: %v", err2)
		}

		// Results should be the same
		if result1 != result2 {
			t.Errorf("Cached result differs: first=%v, second=%v", result1, result2)
		}

		// Cache timestamp should not have changed
		if !reconciler.gatewayCache.lastCheck.Equal(firstCheck) {
			t.Errorf("Cache was refreshed unexpectedly")
		}
	})

	t.Run("Cache Expiry", func(t *testing.T) {
		// Manually set cache to be stale
		reconciler.gatewayCache.mutex.Lock()
		reconciler.gatewayCache.lastCheck = time.Now().Add(-10 * time.Minute) // Way past TTL
		reconciler.gatewayCache.available = false // Set to different value
		reconciler.gatewayCache.mutex.Unlock()

		oldCheck := reconciler.gatewayCache.lastCheck

		// Call should refresh cache
		_, err := reconciler.hasGatewayAPI(ctx)
		if err != nil {
			t.Fatalf("Cache refresh failed: %v", err)
		}

		// Cache timestamp should have been updated
		if !reconciler.gatewayCache.lastCheck.After(oldCheck) {
			t.Errorf("Cache was not refreshed after expiry")
		}
	})

	t.Run("Concurrent Access Safety", func(t *testing.T) {
		// Test multiple goroutines accessing cache simultaneously
		const numGoroutines = 10
		results := make([]bool, numGoroutines)
		errors := make([]error, numGoroutines)
		done := make(chan int, numGoroutines)

		// Manually expire cache first
		reconciler.gatewayCache.mutex.Lock()
		reconciler.gatewayCache.lastCheck = time.Now().Add(-10 * time.Minute)
		reconciler.gatewayCache.mutex.Unlock()

		// Launch multiple goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				results[index], errors[index] = reconciler.hasGatewayAPI(ctx)
				done <- index
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check that all results are consistent
		firstResult := results[0]
		for i := 1; i < numGoroutines; i++ {
			if errors[i] != nil {
				t.Errorf("Goroutine %d failed: %v", i, errors[i])
			}
			if results[i] != firstResult {
				t.Errorf("Goroutine %d got different result: %v vs %v", i, results[i], firstResult)
			}
		}
	})
}

func TestInitializeGatewayCache(t *testing.T) {
	reconciler := &LanguageAgentReconciler{}

	// Before initialization, cache should be nil
	if reconciler.gatewayCache != nil {
		t.Errorf("Cache should be nil before initialization")
	}

	// After initialization, cache should be ready
	reconciler.InitializeGatewayCache()
	if reconciler.gatewayCache == nil {
		t.Errorf("Cache should not be nil after initialization")
	}

	// lastCheck should be zero initially
	if !reconciler.gatewayCache.lastCheck.IsZero() {
		t.Errorf("Initial lastCheck should be zero")
	}
}

func TestGatewayCacheTTL(t *testing.T) {
	if gatewayAPICacheTTL != 5*time.Minute {
		t.Errorf("Expected cache TTL to be 5 minutes, got %v", gatewayAPICacheTTL)
	}
}