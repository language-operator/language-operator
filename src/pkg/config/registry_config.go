package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RegistryConfigManager manages the allowed container registries configuration
// by watching the operator-config ConfigMap for dynamic updates
type RegistryConfigManager struct {
	clientset         kubernetes.Interface
	operatorNamespace string
	registries        []string
	mu                sync.RWMutex
	informer          cache.Controller
	stopCh            chan struct{}
}

// NewRegistryConfigManager creates a new registry configuration manager
func NewRegistryConfigManager(clientset kubernetes.Interface) *RegistryConfigManager {
	operatorNamespace := os.Getenv("OPERATOR_NAMESPACE")
	if operatorNamespace == "" {
		operatorNamespace = "kube-system" // Default namespace for the operator
	}

	return &RegistryConfigManager{
		clientset:         clientset,
		operatorNamespace: operatorNamespace,
		registries:        getDefaultRegistries(),
		stopCh:            make(chan struct{}),
	}
}

// GetRegistries returns the current list of allowed registries (thread-safe)
func (r *RegistryConfigManager) GetRegistries() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(r.registries))
	copy(result, r.registries)
	return result
}

// StartWatcher starts the ConfigMap watcher in a separate goroutine
func (r *RegistryConfigManager) StartWatcher(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("registry-config-manager")

	// Initial load attempt
	if err := r.loadRegistries(ctx); err != nil {
		logger.Info("Failed to load initial registries, using defaults", "error", err.Error())
	}

	// Create a ListWatch for the operator-config ConfigMap
	listWatch := cache.NewListWatchFromClient(
		r.clientset.CoreV1().RESTClient(),
		"configmaps",
		r.operatorNamespace,
		fields.OneTermEqualSelector("metadata.name", "operator-config"),
	)

	// Create informer
	_, controller := cache.NewInformer(
		listWatch,
		&v1.ConfigMap{},
		30*time.Second, // resync period
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*v1.ConfigMap); ok {
					logger.Info("ConfigMap added", "name", cm.Name)
					r.handleConfigMapUpdate(ctx, cm)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if cm, ok := newObj.(*v1.ConfigMap); ok {
					logger.Info("ConfigMap updated", "name", cm.Name)
					r.handleConfigMapUpdate(ctx, cm)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if cm, ok := obj.(*v1.ConfigMap); ok {
					logger.Info("ConfigMap deleted, falling back to defaults", "name", cm.Name)
					r.handleConfigMapDelete(ctx)
				}
			},
		},
	)

	r.informer = controller

	// Start the informer in a goroutine
	go func() {
		logger.Info("Starting ConfigMap watcher", "namespace", r.operatorNamespace)
		defer runtime.HandleCrash()
		r.informer.Run(r.stopCh)
		logger.Info("ConfigMap watcher stopped")
	}()

	// Wait for cache sync
	go func() {
		if !cache.WaitForCacheSync(r.stopCh, r.informer.HasSynced) {
			logger.Error(fmt.Errorf("cache sync failed"), "Failed to sync ConfigMap cache")
			return
		}
		logger.Info("ConfigMap cache synced successfully")
	}()

	return nil
}

// Stop stops the ConfigMap watcher
func (r *RegistryConfigManager) Stop() {
	if r.stopCh != nil {
		close(r.stopCh)
	}
}

// loadRegistries loads registries from the ConfigMap
func (r *RegistryConfigManager) loadRegistries(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("registry-config-manager")

	configMap, err := r.clientset.CoreV1().ConfigMaps(r.operatorNamespace).Get(ctx, "operator-config", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get operator-config ConfigMap: %w", err)
	}

	// Validate ConfigMap structure
	if err := r.validateConfigMapSchema(configMap.Data); err != nil {
		return fmt.Errorf("invalid ConfigMap structure: %w", err)
	}

	// Parse registries
	registriesData, ok := configMap.Data["allowed-registries"]
	if !ok {
		return fmt.Errorf("allowed-registries key not found in ConfigMap")
	}

	registries, err := r.parseRegistries(registriesData)
	if err != nil {
		return fmt.Errorf("failed to parse registries: %w", err)
	}

	// Update the registries atomically
	r.mu.Lock()
	r.registries = registries
	r.mu.Unlock()

	logger.Info("Registries loaded from ConfigMap", "count", len(registries), "registries", registries)
	return nil
}

// handleConfigMapUpdate handles ConfigMap add/update events
func (r *RegistryConfigManager) handleConfigMapUpdate(ctx context.Context, cm *v1.ConfigMap) {
	logger := log.FromContext(ctx).WithName("registry-config-manager")

	// Validate the ConfigMap structure
	if err := r.validateConfigMapSchema(cm.Data); err != nil {
		logger.Error(err, "Invalid ConfigMap structure, ignoring update")
		return
	}

	// Parse the allowed-registries data
	registriesData, ok := cm.Data["allowed-registries"]
	if !ok {
		logger.Error(fmt.Errorf("missing key"), "allowed-registries key not found in ConfigMap, ignoring update")
		return
	}

	registries, err := r.parseRegistries(registriesData)
	if err != nil {
		logger.Error(err, "Failed to parse registries from ConfigMap, ignoring update")
		return
	}

	// Update the registries atomically
	r.mu.Lock()
	oldCount := len(r.registries)
	r.registries = registries
	r.mu.Unlock()

	logger.Info("Registry configuration updated",
		"oldCount", oldCount,
		"newCount", len(registries),
		"registries", registries)
}

// handleConfigMapDelete handles ConfigMap delete events
func (r *RegistryConfigManager) handleConfigMapDelete(ctx context.Context) {
	logger := log.FromContext(ctx).WithName("registry-config-manager")

	// Fall back to default registries
	defaults := getDefaultRegistries()

	r.mu.Lock()
	oldCount := len(r.registries)
	r.registries = defaults
	r.mu.Unlock()

	logger.Info("Registry configuration reset to defaults",
		"oldCount", oldCount,
		"newCount", len(defaults),
		"registries", defaults)
}

// parseRegistries parses the registry list from ConfigMap data
func (r *RegistryConfigManager) parseRegistries(data string) ([]string, error) {
	var registries []string

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			registries = append(registries, line)
		}
	}

	if len(registries) == 0 {
		return nil, fmt.Errorf("no registries found")
	}

	return registries, nil
}

// validateConfigMapSchema validates that the ConfigMap contains only supported fields
func (r *RegistryConfigManager) validateConfigMapSchema(data map[string]string) error {
	// Define supported fields for operator-config ConfigMap
	supportedFields := map[string]bool{
		"allowed-registries": true,
	}

	// Check for unknown fields
	for key := range data {
		if !supportedFields[key] {
			return fmt.Errorf("unknown field in ConfigMap: %s", key)
		}
	}

	return nil
}

// getDefaultRegistries returns the default list of allowed registries
// These are used as fallback when ConfigMap is unavailable
func getDefaultRegistries() []string {
	return []string{
		"docker.io",
		"gcr.io",
		"*.gcr.io",
		"quay.io",
		"ghcr.io",
		"registry.k8s.io",
		"codeberg.org",
		"gitlab.com",
	}
}
