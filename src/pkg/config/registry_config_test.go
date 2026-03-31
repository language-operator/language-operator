package config

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewRegistryConfigManager(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	manager := NewRegistryConfigManager(clientset)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.clientset != clientset {
		t.Error("Expected clientset to be set")
	}

	if manager.operatorNamespace != "kube-system" {
		t.Errorf("Expected default namespace 'kube-system', got %s", manager.operatorNamespace)
	}

	// Should start with default registries
	registries := manager.GetRegistries()
	if len(registries) == 0 {
		t.Error("Expected default registries to be set")
	}

	// Verify some known defaults
	found := false
	for _, reg := range registries {
		if reg == "docker.io" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected docker.io in default registries")
	}
}

func TestGetRegistries(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manager := NewRegistryConfigManager(clientset)

	// Test that GetRegistries returns a copy (not the original slice)
	registries1 := manager.GetRegistries()
	registries2 := manager.GetRegistries()

	// Should have same content
	if len(registries1) != len(registries2) {
		t.Error("Expected same number of registries")
	}

	// But different slices
	registries1[0] = "modified"
	if registries2[0] == "modified" {
		t.Error("Expected GetRegistries to return a copy, not original slice")
	}
}

func TestLoadRegistries(t *testing.T) {
	tests := []struct {
		name          string
		configMap     *v1.ConfigMap
		expectError   bool
		expectedCount int
	}{
		{
			name: "Valid ConfigMap",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "language-operator-config",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"allowed-registries": "gcr.io/company\ndocker.io\n# comment line\nquay.io",
				},
			},
			expectError:   false,
			expectedCount: 3,
		},
		{
			name: "Missing allowed-registries key",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "language-operator-config",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"other-key": "value",
				},
			},
			expectError: true,
		},
		{
			name: "Invalid field in ConfigMap",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "language-operator-config",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"allowed-registries": "docker.io",
					"invalid-field":      "value",
				},
			},
			expectError: true,
		},
		{
			name: "Empty registries list",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "language-operator-config",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"allowed-registries": "# only comments\n   \n",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			// Create the ConfigMap if provided
			if tt.configMap != nil {
				_, err := clientset.CoreV1().ConfigMaps("kube-system").Create(
					context.Background(), tt.configMap, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create test ConfigMap: %v", err)
				}
			}

			manager := NewRegistryConfigManager(clientset)
			err := manager.loadRegistries(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && tt.expectedCount > 0 {
				registries := manager.GetRegistries()
				if len(registries) != tt.expectedCount {
					t.Errorf("Expected %d registries, got %d", tt.expectedCount, len(registries))
				}
			}
		})
	}
}

func TestParseRegistries(t *testing.T) {
	manager := &RegistryConfigManager{}

	tests := []struct {
		name        string
		input       string
		expected    []string
		expectError bool
	}{
		{
			name:        "Valid registries with comments",
			input:       "docker.io\ngcr.io\n# comment\nquay.io\n   \n",
			expected:    []string{"docker.io", "gcr.io", "quay.io"},
			expectError: false,
		},
		{
			name:        "Single registry",
			input:       "docker.io",
			expected:    []string{"docker.io"},
			expectError: false,
		},
		{
			name:        "Empty input",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Only comments and whitespace",
			input:       "# comment\n   \n# another comment",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Registries with whitespace",
			input:       "  docker.io  \n  gcr.io  ",
			expected:    []string{"docker.io", "gcr.io"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.parseRegistries(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d registries, got %d", len(tt.expected), len(result))
					return
				}

				for i, expected := range tt.expected {
					if result[i] != expected {
						t.Errorf("Expected registry[%d] = %s, got %s", i, expected, result[i])
					}
				}
			}
		})
	}
}

func TestValidateConfigMapSchema(t *testing.T) {
	manager := &RegistryConfigManager{}

	tests := []struct {
		name        string
		data        map[string]string
		expectError bool
	}{
		{
			name: "Valid schema",
			data: map[string]string{
				"allowed-registries": "docker.io",
			},
			expectError: false,
		},
		{
			name:        "Empty data",
			data:        map[string]string{},
			expectError: false,
		},
		{
			name: "Invalid field",
			data: map[string]string{
				"allowed-registries": "docker.io",
				"unknown-field":      "value",
			},
			expectError: true,
		},
		{
			name: "Multiple invalid fields",
			data: map[string]string{
				"allowed-registries": "docker.io",
				"field1":             "value1",
				"field2":             "value2",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateConfigMapSchema(tt.data)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestHandleConfigMapUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid ConfigMap updates registries", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		manager := NewRegistryConfigManager(clientset)

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "language-operator-config", Namespace: "kube-system"},
			Data:       map[string]string{"allowed-registries": "custom.io\nmy-registry.example.com"},
		}
		manager.handleConfigMapUpdate(ctx, cm)

		registries := manager.GetRegistries()
		if len(registries) != 2 {
			t.Fatalf("expected 2 registries, got %d", len(registries))
		}
		if registries[0] != "custom.io" || registries[1] != "my-registry.example.com" {
			t.Errorf("unexpected registries: %v", registries)
		}
	})

	t.Run("invalid schema leaves registries unchanged", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		manager := NewRegistryConfigManager(clientset)
		original := manager.GetRegistries()

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "language-operator-config", Namespace: "kube-system"},
			Data: map[string]string{
				"allowed-registries": "custom.io",
				"unknown-field":      "bad",
			},
		}
		manager.handleConfigMapUpdate(ctx, cm)

		registries := manager.GetRegistries()
		if len(registries) != len(original) {
			t.Errorf("expected registries to be unchanged, got %v", registries)
		}
	})

	t.Run("missing allowed-registries key leaves registries unchanged", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		manager := NewRegistryConfigManager(clientset)
		original := manager.GetRegistries()

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "language-operator-config", Namespace: "kube-system"},
			Data:       map[string]string{},
		}
		manager.handleConfigMapUpdate(ctx, cm)

		registries := manager.GetRegistries()
		if len(registries) != len(original) {
			t.Errorf("expected registries to be unchanged, got %v", registries)
		}
	})

	t.Run("only comments in registry data leaves registries unchanged", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		manager := NewRegistryConfigManager(clientset)
		original := manager.GetRegistries()

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "language-operator-config", Namespace: "kube-system"},
			Data:       map[string]string{"allowed-registries": "# only a comment\n\n"},
		}
		manager.handleConfigMapUpdate(ctx, cm)

		registries := manager.GetRegistries()
		if len(registries) != len(original) {
			t.Errorf("expected registries to be unchanged, got %v", registries)
		}
	})
}

func TestHandleConfigMapDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("resets to defaults after custom registries", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		manager := NewRegistryConfigManager(clientset)

		// Set non-default registries first
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "language-operator-config", Namespace: "kube-system"},
			Data:       map[string]string{"allowed-registries": "custom.io"},
		}
		manager.handleConfigMapUpdate(ctx, cm)
		if regs := manager.GetRegistries(); len(regs) != 1 || regs[0] != "custom.io" {
			t.Fatalf("setup failed: expected [custom.io], got %v", regs)
		}

		manager.handleConfigMapDelete(ctx)

		defaults := getDefaultRegistries()
		registries := manager.GetRegistries()
		if len(registries) != len(defaults) {
			t.Errorf("expected %d default registries, got %d: %v", len(defaults), len(registries), registries)
		}
		for i, def := range defaults {
			if registries[i] != def {
				t.Errorf("registry[%d]: expected %s, got %s", i, def, registries[i])
			}
		}
	})

	t.Run("no panic when called with no prior custom registries", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		manager := NewRegistryConfigManager(clientset)

		// Should not panic
		manager.handleConfigMapDelete(ctx)

		defaults := getDefaultRegistries()
		registries := manager.GetRegistries()
		if len(registries) != len(defaults) {
			t.Errorf("expected %d defaults, got %d", len(defaults), len(registries))
		}
	})
}

func TestStop_DoubleClose(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manager := NewRegistryConfigManager(clientset)

	// First call closes the channel — should not panic
	manager.Stop()
	// Second call must also not panic (double-close would panic without sync.Once)
	manager.Stop()
}

func TestConcurrentAccess(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manager := NewRegistryConfigManager(clientset)

	// Test concurrent reads and writes
	done := make(chan bool)

	// Start multiple readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = manager.GetRegistries()
			}
			done <- true
		}()
	}

	// Start a writer
	go func() {
		for j := 0; j < 10; j++ {
			manager.mu.Lock()
			manager.registries = []string{"test-registry"}
			manager.mu.Unlock()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 11; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}
