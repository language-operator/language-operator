/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/language-operator/language-operator/pkg/telemetry"
	"github.com/language-operator/language-operator/pkg/telemetry/adapters"
)

func TestInitializeTelemetryAdapter(t *testing.T) {
	// Store original env vars to restore after tests
	originalType := os.Getenv("TELEMETRY_ADAPTER_TYPE")
	originalEndpoint := os.Getenv("TELEMETRY_ADAPTER_ENDPOINT")
	originalAPIKey := os.Getenv("TELEMETRY_ADAPTER_API_KEY")
	originalTimeout := os.Getenv("TELEMETRY_ADAPTER_TIMEOUT")

	defer func() {
		os.Setenv("TELEMETRY_ADAPTER_TYPE", originalType)
		os.Setenv("TELEMETRY_ADAPTER_ENDPOINT", originalEndpoint)
		os.Setenv("TELEMETRY_ADAPTER_API_KEY", originalAPIKey)
		os.Setenv("TELEMETRY_ADAPTER_TIMEOUT", originalTimeout)
	}()

	tests := []struct {
		name         string
		envVars      map[string]string
		expectedType string
		shouldBeNoop bool
	}{
		{
			name: "no configuration - defaults to NoOpAdapter",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_TYPE": "",
			},
			expectedType: "*telemetry.NoOpAdapter",
			shouldBeNoop: true,
		},
		{
			name: "explicitly disabled - NoOpAdapter",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_TYPE": "disabled",
			},
			expectedType: "*telemetry.NoOpAdapter",
			shouldBeNoop: true,
		},
		{
			name: "noop type - NoOpAdapter",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_TYPE": "noop",
			},
			expectedType: "*telemetry.NoOpAdapter",
			shouldBeNoop: true,
		},
		{
			name: "signoz without config - falls back to NoOpAdapter",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_TYPE":     "signoz",
				"TELEMETRY_ADAPTER_ENDPOINT": "",
				"TELEMETRY_ADAPTER_API_KEY":  "",
			},
			expectedType: "*telemetry.NoOpAdapter",
			shouldBeNoop: true,
		},
		{
			name: "signoz with valid config - creates SigNozAdapter",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_TYPE":     "signoz",
				"TELEMETRY_ADAPTER_ENDPOINT": "https://signoz.example.com",
				"TELEMETRY_ADAPTER_API_KEY":  "test-api-key",
			},
			expectedType: "*adapters.SignozAdapter",
			shouldBeNoop: false,
		},
		{
			name: "unknown adapter type - falls back to NoOpAdapter",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_TYPE": "unknown",
			},
			expectedType: "*telemetry.NoOpAdapter",
			shouldBeNoop: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("TELEMETRY_ADAPTER_TYPE")
			os.Unsetenv("TELEMETRY_ADAPTER_ENDPOINT")
			os.Unsetenv("TELEMETRY_ADAPTER_API_KEY")
			os.Unsetenv("TELEMETRY_ADAPTER_TIMEOUT")

			// Set test env vars
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			adapter := initializeTelemetryAdapter()

			// Check that we got an adapter
			if adapter == nil {
				t.Fatal("Expected adapter to be non-nil")
			}

			// Check Available() behavior
			available := adapter.Available()
			if tc.shouldBeNoop && available {
				t.Errorf("Expected NoOpAdapter to report as unavailable, got available=true")
			}
			// Note: Real adapters may report unavailable if endpoint is not reachable
			// We only test that NoOpAdapters are properly unavailable

			// Verify adapter type by checking interface implementation
			switch tc.expectedType {
			case "*telemetry.NoOpAdapter":
				if _, ok := adapter.(*telemetry.NoOpAdapter); !ok {
					t.Errorf("Expected NoOpAdapter, got %T", adapter)
				}
			case "*adapters.SignozAdapter":
				if _, ok := adapter.(*adapters.SignozAdapter); !ok {
					t.Errorf("Expected SignozAdapter, got %T", adapter)
				}
			}
		})
	}
}

func TestInitializeSigNozAdapter(t *testing.T) {
	// Store original env vars to restore after tests
	originalEndpoint := os.Getenv("TELEMETRY_ADAPTER_ENDPOINT")
	originalAPIKey := os.Getenv("TELEMETRY_ADAPTER_API_KEY")
	originalTimeout := os.Getenv("TELEMETRY_ADAPTER_TIMEOUT")

	defer func() {
		os.Setenv("TELEMETRY_ADAPTER_ENDPOINT", originalEndpoint)
		os.Setenv("TELEMETRY_ADAPTER_API_KEY", originalAPIKey)
		os.Setenv("TELEMETRY_ADAPTER_TIMEOUT", originalTimeout)
	}()

	tests := []struct {
		name         string
		endpoint     string
		apiKey       string
		timeout      string
		shouldBeNoop bool
	}{
		{
			name:         "missing endpoint - falls back to NoOpAdapter",
			endpoint:     "",
			apiKey:       "test-key",
			timeout:      "",
			shouldBeNoop: true,
		},
		{
			name:         "missing API key - falls back to NoOpAdapter",
			endpoint:     "https://signoz.example.com",
			apiKey:       "",
			timeout:      "",
			shouldBeNoop: true,
		},
		{
			name:         "valid config - creates SigNozAdapter",
			endpoint:     "https://signoz.example.com",
			apiKey:       "test-key",
			timeout:      "",
			shouldBeNoop: false,
		},
		{
			name:         "valid config with custom timeout - creates SigNozAdapter",
			endpoint:     "https://signoz.example.com",
			apiKey:       "test-key",
			timeout:      "60s",
			shouldBeNoop: false,
		},
		{
			name:         "invalid timeout format - uses default and creates SigNozAdapter",
			endpoint:     "https://signoz.example.com",
			apiKey:       "test-key",
			timeout:      "invalid",
			shouldBeNoop: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment variables
			os.Setenv("TELEMETRY_ADAPTER_ENDPOINT", tc.endpoint)
			os.Setenv("TELEMETRY_ADAPTER_API_KEY", tc.apiKey)
			os.Setenv("TELEMETRY_ADAPTER_TIMEOUT", tc.timeout)

			adapter := initializeSigNozAdapter()

			// Check that we got an adapter
			if adapter == nil {
				t.Fatal("Expected adapter to be non-nil")
			}

			// Check type and availability
			if tc.shouldBeNoop {
				if _, ok := adapter.(*telemetry.NoOpAdapter); !ok {
					t.Errorf("Expected NoOpAdapter fallback, got %T", adapter)
				}
				if adapter.Available() {
					t.Error("Expected NoOpAdapter to report unavailable")
				}
			} else {
				if _, ok := adapter.(*adapters.SignozAdapter); !ok {
					t.Errorf("Expected SignozAdapter, got %T", adapter)
				}
				// Note: SigNoz adapter availability depends on network connectivity to the endpoint
				// In tests, this will likely be false unless the endpoint is actually reachable
			}
		})
	}
}

func TestStartupTimeoutConfiguration(t *testing.T) {
	// Store original env var to restore after tests
	originalTimeout := os.Getenv("STARTUP_TIMEOUT")
	defer func() {
		if originalTimeout == "" {
			os.Unsetenv("STARTUP_TIMEOUT")
		} else {
			os.Setenv("STARTUP_TIMEOUT", originalTimeout)
		}
	}()

	tests := []struct {
		name            string
		envValue        string
		expectedTimeout time.Duration
		expectDefault   bool
	}{
		{
			name:            "no env var - uses default",
			envValue:        "",
			expectedTimeout: 60 * time.Second,
			expectDefault:   true,
		},
		{
			name:            "valid timeout value",
			envValue:        "120s",
			expectedTimeout: 120 * time.Second,
			expectDefault:   false,
		},
		{
			name:            "valid timeout in minutes",
			envValue:        "5m",
			expectedTimeout: 5 * time.Minute,
			expectDefault:   false,
		},
		{
			name:            "invalid timeout - falls back to default",
			envValue:        "invalid",
			expectedTimeout: 60 * time.Second,
			expectDefault:   true,
		},
		{
			name:            "zero timeout - uses zero",
			envValue:        "0s",
			expectedTimeout: 0,
			expectDefault:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment variable
			if tc.envValue == "" {
				os.Unsetenv("STARTUP_TIMEOUT")
			} else {
				os.Setenv("STARTUP_TIMEOUT", tc.envValue)
			}

			// Parse timeout (mimics main.go logic)
			startupTimeout := 60 * time.Second
			if timeoutStr := os.Getenv("STARTUP_TIMEOUT"); timeoutStr != "" {
				if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
					startupTimeout = parsedTimeout
				}
				// Note: In real code, invalid values log an error but we don't test that here
			}

			if startupTimeout != tc.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tc.expectedTimeout, startupTimeout)
			}
		})
	}
}

func TestLoadAllowedRegistries(t *testing.T) {
	// Store and restore original env var
	originalNamespace := os.Getenv("OPERATOR_NAMESPACE")
	defer func() {
		if originalNamespace == "" {
			os.Unsetenv("OPERATOR_NAMESPACE")
		} else {
			os.Setenv("OPERATOR_NAMESPACE", originalNamespace)
		}
	}()

	tests := []struct {
		name          string
		namespace     string
		configMap     *corev1.ConfigMap
		expectError   bool
		expectedRegs  []string
		errorContains string
	}{
		{
			name:      "valid configmap with multiple registries",
			namespace: "test-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"allowed-registries": "docker.io\ngcr.io\n*.gcr.io\nquay.io\n# comment line\n\n  # another comment",
				},
			},
			expectError:  false,
			expectedRegs: []string{"docker.io", "gcr.io", "*.gcr.io", "quay.io"},
		},
		{
			name:      "valid configmap with single registry",
			namespace: "kube-system", // default namespace
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"allowed-registries": "docker.io",
				},
			},
			expectError:  false,
			expectedRegs: []string{"docker.io"},
		},
		{
			name:      "configmap with unknown field (version) - should fail",
			namespace: "test-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"allowed-registries": "docker.io\ngcr.io",
					"version":            "-1", // This should cause validation failure
				},
			},
			expectError:   true,
			errorContains: "unsupported fields found: [version]",
		},
		{
			name:      "configmap with multiple unknown fields - should fail",
			namespace: "test-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"allowed-registries": "docker.io",
					"version":            "1.0",
					"extra-field":        "value",
				},
			},
			expectError:   true,
			errorContains: "unsupported fields found:",
		},
		{
			name:      "missing allowed-registries key - should fail",
			namespace: "test-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"other-field": "value",
				},
			},
			expectError:   true,
			errorContains: "unsupported fields found: [other-field]",
		},
		{
			name:      "empty registries data - should fail",
			namespace: "test-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"allowed-registries": "# only comments\n\n   \n",
				},
			},
			expectError:   true,
			errorContains: "no registries found",
		},
		{
			name:      "registries with whitespace handling",
			namespace: "test-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"allowed-registries": "  docker.io  \n\n  gcr.io\n   \n\t*.gcr.io\t",
				},
			},
			expectError:  false,
			expectedRegs: []string{"docker.io", "gcr.io", "*.gcr.io"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment
			if tc.namespace != "" {
				os.Setenv("OPERATOR_NAMESPACE", tc.namespace)
			} else {
				os.Unsetenv("OPERATOR_NAMESPACE")
			}

			// Create fake clientset with the test ConfigMap
			clientset := fake.NewSimpleClientset(tc.configMap)

			// Call function under test
			registries, err := loadAllowedRegistries(context.Background(), clientset)

			// Check error expectations
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tc.errorContains, err.Error())
				}
				return
			}

			// Check success expectations
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			// Check registry list
			if len(registries) != len(tc.expectedRegs) {
				t.Errorf("Expected %d registries, got %d: %v", len(tc.expectedRegs), len(registries), registries)
				return
			}

			for i, expected := range tc.expectedRegs {
				if registries[i] != expected {
					t.Errorf("Expected registry[%d] = %s, got %s", i, expected, registries[i])
				}
			}
		})
	}
}

func TestLoadAllowedRegistries_ConfigMapNotFound(t *testing.T) {
	// Store and restore original env var
	originalNamespace := os.Getenv("OPERATOR_NAMESPACE")
	defer func() {
		if originalNamespace == "" {
			os.Unsetenv("OPERATOR_NAMESPACE")
		} else {
			os.Setenv("OPERATOR_NAMESPACE", originalNamespace)
		}
	}()

	os.Setenv("OPERATOR_NAMESPACE", "test-namespace")

	// Create fake clientset without the ConfigMap
	clientset := fake.NewSimpleClientset()

	registries, err := loadAllowedRegistries(context.Background(), clientset)

	if err == nil {
		t.Errorf("Expected error when ConfigMap not found, got registries: %v", registries)
		return
	}

	if !contains(err.Error(), "failed to get operator-config ConfigMap") {
		t.Errorf("Expected error about missing ConfigMap, got: %v", err)
	}
}

func TestLoadAllowedRegistries_DefaultNamespace(t *testing.T) {
	// Store and restore original env var
	originalNamespace := os.Getenv("OPERATOR_NAMESPACE")
	defer func() {
		if originalNamespace == "" {
			os.Unsetenv("OPERATOR_NAMESPACE")
		} else {
			os.Setenv("OPERATOR_NAMESPACE", originalNamespace)
		}
	}()

	// Clear the namespace env var to test default behavior
	os.Unsetenv("OPERATOR_NAMESPACE")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator-config",
			Namespace: "kube-system", // Should default to kube-system
		},
		Data: map[string]string{
			"allowed-registries": "docker.io",
		},
	}

	clientset := fake.NewSimpleClientset(configMap)

	registries, err := loadAllowedRegistries(context.Background(), clientset)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
		return
	}

	expectedRegs := []string{"docker.io"}
	if len(registries) != len(expectedRegs) || registries[0] != expectedRegs[0] {
		t.Errorf("Expected registries %v, got %v", expectedRegs, registries)
	}
}

func TestValidateOperatorConfigMapSchema(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid - only allowed-registries field",
			data: map[string]string{
				"allowed-registries": "docker.io\ngcr.io",
			},
			expectError: false,
		},
		{
			name:          "valid - empty data map",
			data:          map[string]string{},
			expectError:   false,
		},
		{
			name: "invalid - version field not supported",
			data: map[string]string{
				"allowed-registries": "docker.io",
				"version":            "1.0",
			},
			expectError:   true,
			errorContains: "unsupported fields found: [version]",
		},
		{
			name: "invalid - multiple unknown fields",
			data: map[string]string{
				"allowed-registries": "docker.io",
				"version":            "1.0",
				"extra-field":        "value",
				"another-field":      "value2",
			},
			expectError:   true,
			errorContains: "unsupported fields found:",
		},
		{
			name: "invalid - only unknown fields",
			data: map[string]string{
				"version":     "1.0",
				"extra-field": "value",
			},
			expectError:   true,
			errorContains: "unsupported fields found:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOperatorConfigMapSchema(tc.data)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
