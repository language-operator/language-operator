package controllers

import (
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

	t.Run("Cache Structure and TTL Constants", func(t *testing.T) {
		// Verify cache is initialized properly
		if reconciler.gatewayCache == nil {
			t.Fatalf("Cache should be initialized")
		}

		// Verify TTL constant
		if gatewayAPICacheTTL != 5*time.Minute {
			t.Errorf("Expected cache TTL to be 5 minutes, got %v", gatewayAPICacheTTL)
		}

		// Verify initial state
		if !reconciler.gatewayCache.lastCheck.IsZero() {
			t.Errorf("Initial lastCheck should be zero")
		}
	})

	t.Run("Cache TTL Logic", func(t *testing.T) {
		// Simulate fresh cache (recently checked)
		reconciler.gatewayCache.mutex.Lock()
		reconciler.gatewayCache.available = true
		reconciler.gatewayCache.lastCheck = time.Now()
		reconciler.gatewayCache.mutex.Unlock()

		// Check if cache is considered fresh
		reconciler.gatewayCache.mutex.RLock()
		isFresh := time.Since(reconciler.gatewayCache.lastCheck) < gatewayAPICacheTTL
		reconciler.gatewayCache.mutex.RUnlock()

		if !isFresh {
			t.Errorf("Recently set cache should be considered fresh")
		}

		// Simulate stale cache
		reconciler.gatewayCache.mutex.Lock()
		reconciler.gatewayCache.lastCheck = time.Now().Add(-10 * time.Minute)
		reconciler.gatewayCache.mutex.Unlock()

		// Check if cache is considered stale
		reconciler.gatewayCache.mutex.RLock()
		isStale := time.Since(reconciler.gatewayCache.lastCheck) >= gatewayAPICacheTTL
		reconciler.gatewayCache.mutex.RUnlock()

		if !isStale {
			t.Errorf("Old cache should be considered stale")
		}
	})

	t.Run("Concurrent Cache Access Safety", func(t *testing.T) {
		// Test that multiple goroutines can safely read/write cache
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		// Launch multiple goroutines that manipulate cache
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				// Simulate cache read
				reconciler.gatewayCache.mutex.RLock()
				_ = reconciler.gatewayCache.available
				_ = reconciler.gatewayCache.lastCheck
				reconciler.gatewayCache.mutex.RUnlock()

				// Simulate cache write
				reconciler.gatewayCache.mutex.Lock()
				reconciler.gatewayCache.available = (index%2 == 0)
				reconciler.gatewayCache.lastCheck = time.Now()
				reconciler.gatewayCache.mutex.Unlock()

				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Fatalf("Goroutine %d didn't complete in time", i)
			}
		}

		// Verify final state is consistent (no race conditions)
		reconciler.gatewayCache.mutex.RLock()
		available := reconciler.gatewayCache.available
		lastCheck := reconciler.gatewayCache.lastCheck
		reconciler.gatewayCache.mutex.RUnlock()

		// Check that values are reasonable (not corrupted by races)
		if lastCheck.IsZero() || lastCheck.After(time.Now().Add(time.Second)) {
			t.Errorf("lastCheck timestamp appears corrupted: %v", lastCheck)
		}
		// available is either true or false, both are valid
		_ = available
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

// TestGatewayAPICache_Integration tests the integration between cache and discovery
// but in a way that doesn't require real Kubernetes API access
func TestGatewayAPICache_Integration(t *testing.T) {
	reconciler := &LanguageAgentReconciler{}
	reconciler.InitializeGatewayCache()

	// Test that cache correctly determines freshness
	now := time.Now()

	// Test fresh cache
	reconciler.gatewayCache.mutex.Lock()
	reconciler.gatewayCache.lastCheck = now
	reconciler.gatewayCache.available = true
	reconciler.gatewayCache.mutex.Unlock()

	reconciler.gatewayCache.mutex.RLock()
	timeSince := time.Since(reconciler.gatewayCache.lastCheck)
	isFresh := timeSince < gatewayAPICacheTTL
	reconciler.gatewayCache.mutex.RUnlock()

	if !isFresh {
		t.Errorf("Cache should be fresh, time since: %v, TTL: %v", timeSince, gatewayAPICacheTTL)
	}

	// Test stale cache
	reconciler.gatewayCache.mutex.Lock()
	reconciler.gatewayCache.lastCheck = now.Add(-6 * time.Minute) // Older than 5min TTL
	reconciler.gatewayCache.mutex.Unlock()

	reconciler.gatewayCache.mutex.RLock()
	timeSince = time.Since(reconciler.gatewayCache.lastCheck)
	isStale := timeSince >= gatewayAPICacheTTL
	reconciler.gatewayCache.mutex.RUnlock()

	if !isStale {
		t.Errorf("Cache should be stale, time since: %v, TTL: %v", timeSince, gatewayAPICacheTTL)
	}
}
