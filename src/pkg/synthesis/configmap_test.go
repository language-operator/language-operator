package synthesis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

func TestConfigMapManager_CreateVersionedConfigMap(t *testing.T) {
	tests := []struct {
		name         string
		options      *ConfigMapOptions
		expectError  bool
		validateFunc func(t *testing.T, cm *corev1.ConfigMap)
	}{
		{
			name: "create initial version",
			options: &ConfigMapOptions{
				Code:           "agent 'test' do\nend",
				Version:        1,
				SynthesisType:  "initial",
				LearningSource: "manual",
			},
			expectError: false,
			validateFunc: func(t *testing.T, cm *corev1.ConfigMap) {
				assert.Equal(t, "test-agent-v1", cm.Name)
				assert.Equal(t, "test-namespace", cm.Namespace)
				assert.Equal(t, "1", cm.Labels["langop.io/version"])
				assert.Equal(t, "initial", cm.Labels["langop.io/synthesis-type"])
				assert.Equal(t, "test-agent", cm.Labels["langop.io/agent"])
				assert.Equal(t, "agent-code", cm.Labels["langop.io/component"])
				assert.Equal(t, "agent 'test' do\nend", cm.Data["agent.rb"])
				assert.Contains(t, cm.Annotations, "langop.io/created-at")
				assert.Equal(t, "manual", cm.Annotations["langop.io/learned-from"])
			},
		},
		{
			name: "create learned version with previous version tracking",
			options: &ConfigMapOptions{
				Code:            "agent 'test' do\n  # learned code\nend",
				Version:         2,
				SynthesisType:   "learned",
				PreviousVersion: ptr.To(int32(1)),
				LearnedTask:     "fetch_data",
				LearningSource:  "pattern-detection",
				CustomAnnotations: map[string]string{
					"langop.io/pattern-confidence": "0.95",
				},
			},
			expectError: false,
			validateFunc: func(t *testing.T, cm *corev1.ConfigMap) {
				assert.Equal(t, "test-agent-v2", cm.Name)
				assert.Equal(t, "2", cm.Labels["langop.io/version"])
				assert.Equal(t, "learned", cm.Labels["langop.io/synthesis-type"])
				assert.Equal(t, "1", cm.Labels["langop.io/previous-version"])
				assert.Equal(t, "fetch_data", cm.Labels["langop.io/learned-task"])
				assert.Equal(t, "pattern-detection", cm.Annotations["langop.io/learned-from"])
				assert.Equal(t, "0.95", cm.Annotations["langop.io/pattern-confidence"])
			},
		},
		{
			name:        "invalid version zero",
			options:     &ConfigMapOptions{Version: 0},
			expectError: true,
		},
		{
			name:        "nil options",
			options:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			scheme := runtime.NewScheme()
			require.NoError(t, langopv1alpha1.AddToScheme(scheme))
			require.NoError(t, corev1.AddToScheme(scheme))

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			logger := zap.New(zap.UseDevMode(true))

			manager := &ConfigMapManager{
				Client:   fakeClient,
				Scheme:   scheme,
				Log:      logger,
				Recorder: &record.FakeRecorder{},
			}

			agent := &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
			}

			// Create the agent first
			require.NoError(t, fakeClient.Create(context.Background(), agent))

			// Execute
			cm, err := manager.CreateVersionedConfigMap(context.Background(), agent, tt.options)

			// Validate
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cm)

				// Verify the ConfigMap was actually created in the fake client
				retrievedCM := &corev1.ConfigMap{}
				err = fakeClient.Get(context.Background(), types.NamespacedName{
					Name:      cm.Name,
					Namespace: cm.Namespace,
				}, retrievedCM)
				require.NoError(t, err)

				// Run validation function
				if tt.validateFunc != nil {
					tt.validateFunc(t, retrievedCM)
				}
			}
		})
	}
}

func TestConfigMapManager_GetVersionedConfigMaps(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := zap.New(zap.UseDevMode(true))

	manager := &ConfigMapManager{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logger,
		Recorder: &record.FakeRecorder{},
	}

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
			UID:       "test-uid",
		},
	}

	// Create test ConfigMaps
	testConfigMaps := []*corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-agent-v1",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"langop.io/agent":          "test-agent",
					"langop.io/version":        "1",
					"langop.io/synthesis-type": "initial",
					"langop.io/component":      "agent-code",
				},
				Annotations: map[string]string{
					"langop.io/created-at": "2025-11-24T10:00:00Z",
				},
			},
			Data: map[string]string{
				"agent.rb": "initial code",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-agent-v2",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"langop.io/agent":            "test-agent",
					"langop.io/version":          "2",
					"langop.io/synthesis-type":   "learned",
					"langop.io/component":        "agent-code",
					"langop.io/previous-version": "1",
					"langop.io/learned-task":     "fetch_data",
				},
				Annotations: map[string]string{
					"langop.io/created-at": "2025-11-24T11:00:00Z",
				},
			},
			Data: map[string]string{
				"agent.rb": "learned code",
			},
		},
		// ConfigMap for different agent (should not be included)
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-agent-v1",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"langop.io/agent":          "other-agent",
					"langop.io/version":        "1",
					"langop.io/synthesis-type": "initial",
					"langop.io/component":      "agent-code",
				},
			},
			Data: map[string]string{
				"agent.rb": "other code",
			},
		},
	}

	// Create ConfigMaps
	ctx := context.Background()
	require.NoError(t, fakeClient.Create(ctx, agent))
	for _, cm := range testConfigMaps {
		require.NoError(t, fakeClient.Create(ctx, cm))
	}

	// Execute
	versions, err := manager.GetVersionedConfigMaps(ctx, agent)

	// Validate
	require.NoError(t, err)
	assert.Len(t, versions, 2)

	// Find v1 and v2 in results
	var v1, v2 *ConfigMapVersion
	for _, version := range versions {
		if version.Version == 1 {
			v1 = version
		} else if version.Version == 2 {
			v2 = version
		}
	}

	require.NotNil(t, v1)
	require.NotNil(t, v2)

	// Validate v1
	assert.Equal(t, "test-agent-v1", v1.Name)
	assert.Equal(t, int32(1), v1.Version)
	assert.Equal(t, "initial", v1.SynthesisType)
	assert.Nil(t, v1.PreviousVersion)
	assert.Equal(t, "", v1.LearnedTask)

	// Validate v2
	assert.Equal(t, "test-agent-v2", v2.Name)
	assert.Equal(t, int32(2), v2.Version)
	assert.Equal(t, "learned", v2.SynthesisType)
	assert.NotNil(t, v2.PreviousVersion)
	assert.Equal(t, int32(1), *v2.PreviousVersion)
	assert.Equal(t, "fetch_data", v2.LearnedTask)
}

func TestConfigMapManager_ApplyRetentionPolicy(t *testing.T) {
	tests := []struct {
		name              string
		existingVersions  []int32
		retentionPolicy   *RetentionPolicy
		expectedDeletions []int32
		expectedRemaining []int32
	}{
		{
			name:             "keep last 2 versions",
			existingVersions: []int32{1, 2, 3, 4, 5},
			retentionPolicy: &RetentionPolicy{
				KeepLastN: 2,
			},
			expectedDeletions: []int32{1, 2, 3},
			expectedRemaining: []int32{4, 5},
		},
		{
			name:             "keep last 3 versions, always preserve initial",
			existingVersions: []int32{1, 2, 3, 4, 5},
			retentionPolicy: &RetentionPolicy{
				KeepLastN:         3,
				AlwaysKeepInitial: true,
			},
			expectedDeletions: []int32{2},
			expectedRemaining: []int32{1, 3, 4, 5},
		},
		{
			name:             "cleanup by age - delete versions older than 1 day",
			existingVersions: []int32{1, 2, 3},
			retentionPolicy: &RetentionPolicy{
				CleanupAfterDays: 1,
			},
			expectedDeletions: []int32{1, 2}, // These will be created with old timestamps
			expectedRemaining: []int32{3},
		},
		{
			name:              "no retention policy",
			existingVersions:  []int32{1, 2, 3, 4, 5},
			retentionPolicy:   nil,
			expectedDeletions: []int32{},
			expectedRemaining: []int32{1, 2, 3, 4, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			scheme := runtime.NewScheme()
			require.NoError(t, langopv1alpha1.AddToScheme(scheme))
			require.NoError(t, corev1.AddToScheme(scheme))

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			logger := zap.New(zap.UseDevMode(true))

			manager := &ConfigMapManager{
				Client:   fakeClient,
				Scheme:   scheme,
				Log:      logger,
				Recorder: &record.FakeRecorder{},
			}

			agent := &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
			}

			ctx := context.Background()
			require.NoError(t, fakeClient.Create(ctx, agent))

			// Create test ConfigMaps with different timestamps
			now := time.Now()
			for i, version := range tt.existingVersions {
				// Make older versions have older timestamps
				timestamp := now.Add(-time.Duration(len(tt.existingVersions)-i) * 25 * time.Hour) // 25 hours ago, 50 hours ago, etc.

				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-agent-v%d", version),
						Namespace: "test-namespace",
						Labels: map[string]string{
							"langop.io/agent":          "test-agent",
							"langop.io/version":        fmt.Sprintf("%d", version),
							"langop.io/synthesis-type": "learned",
							"langop.io/component":      "agent-code",
						},
						Annotations: map[string]string{
							"langop.io/created-at": timestamp.Format(time.RFC3339),
						},
					},
					Data: map[string]string{
						"agent.rb": fmt.Sprintf("code v%d", version),
					},
				}

				// Set v1 as initial synthesis type
				if version == 1 {
					cm.Labels["langop.io/synthesis-type"] = "initial"
				}

				require.NoError(t, fakeClient.Create(ctx, cm))
			}

			// Execute
			err := manager.ApplyRetentionPolicy(ctx, agent, tt.retentionPolicy)
			require.NoError(t, err)

			// Validate - check which ConfigMaps still exist
			versions, err := manager.GetVersionedConfigMaps(ctx, agent)
			require.NoError(t, err)

			remainingVersions := make([]int32, len(versions))
			for i, version := range versions {
				remainingVersions[i] = version.Version
			}

			assert.ElementsMatch(t, tt.expectedRemaining, remainingVersions)

			// Validate deletions by checking if deleted ConfigMaps return NotFound errors
			for _, deletedVersion := range tt.expectedDeletions {
				cm := &corev1.ConfigMap{}
				err := fakeClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("test-agent-v%d", deletedVersion),
					Namespace: "test-namespace",
				}, cm)
				assert.True(t, errors.IsNotFound(err), "ConfigMap v%d should have been deleted", deletedVersion)
			}
		})
	}
}

func TestConfigMapManager_GetLatestVersion(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := zap.New(zap.UseDevMode(true))

	manager := &ConfigMapManager{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logger,
		Recorder: &record.FakeRecorder{},
	}

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
			UID:       "test-uid",
		},
	}

	ctx := context.Background()
	require.NoError(t, fakeClient.Create(ctx, agent))

	// Test with no ConfigMaps
	version, err := manager.GetLatestVersion(ctx, agent)
	require.NoError(t, err)
	assert.Equal(t, int32(0), version)

	// Create test ConfigMaps
	versions := []int32{1, 3, 2, 5, 4} // Intentionally out of order
	for _, v := range versions {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-agent-v%d", v),
				Namespace: "test-namespace",
				Labels: map[string]string{
					"langop.io/agent":     "test-agent",
					"langop.io/version":   fmt.Sprintf("%d", v),
					"langop.io/component": "agent-code",
				},
			},
			Data: map[string]string{
				"agent.rb": fmt.Sprintf("code v%d", v),
			},
		}
		require.NoError(t, fakeClient.Create(ctx, cm))
	}

	// Test with multiple ConfigMaps - should return highest version
	version, err = manager.GetLatestVersion(ctx, agent)
	require.NoError(t, err)
	assert.Equal(t, int32(5), version)
}

func TestConfigMapManager_CreateCleanupCronJob(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := zap.New(zap.UseDevMode(true))

	manager := &ConfigMapManager{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logger,
		Recorder: &record.FakeRecorder{},
	}

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
			UID:       "test-uid",
		},
	}

	ctx := context.Background()
	require.NoError(t, fakeClient.Create(ctx, agent))

	// Test with retention policy that includes cleanup interval
	retentionPolicy := &RetentionPolicy{
		KeepLastN:         3,
		CleanupAfterDays:  7,
		AlwaysKeepInitial: true,
		CleanupInterval:   24 * time.Hour,
	}

	// Execute
	err := manager.CreateCleanupCronJob(ctx, agent, retentionPolicy)
	require.NoError(t, err)

	// Validate CronJob was created
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-agent-configmap-cleanup",
		Namespace: "test-namespace",
	}, cronJob)
	require.NoError(t, err)

	// Validate CronJob properties
	assert.Equal(t, "test-agent-configmap-cleanup", cronJob.Name)
	assert.Equal(t, "test-namespace", cronJob.Namespace)
	assert.Equal(t, "test-agent", cronJob.Labels["langop.io/agent"])
	assert.Equal(t, "configmap-cleanup", cronJob.Labels["langop.io/component"])
	assert.Equal(t, "0 3 * * *", cronJob.Spec.Schedule) // Daily at 3 AM
	assert.Equal(t, batchv1.ForbidConcurrent, cronJob.Spec.ConcurrencyPolicy)

	// Validate container command
	containers := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
	require.Len(t, containers, 1)
	container := containers[0]
	assert.Equal(t, "cleanup", container.Name)
	assert.Equal(t, "ghcr.io/language-operator/aictl:latest", container.Image)

	expectedCommand := []string{
		"/usr/local/bin/aictl",
		"agent", "cleanup",
		"--agent", "test-agent",
		"--namespace", "test-namespace",
		"--keep-last", "3",
		"--cleanup-after-days", "7",
		"--always-keep-initial=true",
	}
	assert.Equal(t, expectedCommand, container.Command)

	// Test with nil policy (should not create CronJob)
	err = manager.CreateCleanupCronJob(ctx, agent, nil)
	require.NoError(t, err)

	// Test with zero cleanup interval (should not create CronJob)
	retentionPolicy.CleanupInterval = 0
	err = manager.CreateCleanupCronJob(ctx, agent, retentionPolicy)
	require.NoError(t, err)
}

func TestParseConfigMapVersion(t *testing.T) {
	tests := []struct {
		name        string
		configMap   *corev1.ConfigMap
		expected    *ConfigMapVersion
		expectError bool
	}{
		{
			name: "valid initial version",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-agent-v1",
					Labels: map[string]string{
						"langop.io/version":        "1",
						"langop.io/synthesis-type": "initial",
					},
					Annotations: map[string]string{
						"langop.io/created-at": "2025-11-24T10:00:00Z",
					},
				},
			},
			expected: &ConfigMapVersion{
				Name:          "test-agent-v1",
				Version:       1,
				SynthesisType: "initial",
				CreatedAt:     time.Date(2025, 11, 24, 10, 0, 0, 0, time.UTC),
			},
			expectError: false,
		},
		{
			name: "learned version with previous version",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-agent-v2",
					Labels: map[string]string{
						"langop.io/version":          "2",
						"langop.io/synthesis-type":   "learned",
						"langop.io/previous-version": "1",
						"langop.io/learned-task":     "fetch_data",
					},
					Annotations: map[string]string{
						"langop.io/created-at": "2025-11-24T11:00:00Z",
					},
				},
			},
			expected: &ConfigMapVersion{
				Name:            "test-agent-v2",
				Version:         2,
				SynthesisType:   "learned",
				PreviousVersion: ptr.To(int32(1)),
				LearnedTask:     "fetch_data",
				CreatedAt:       time.Date(2025, 11, 24, 11, 0, 0, 0, time.UTC),
			},
			expectError: false,
		},
		{
			name: "missing version label",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-agent-v1",
					Labels: map[string]string{},
				},
			},
			expectError: true,
		},
		{
			name: "invalid version format",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-agent-v1",
					Labels: map[string]string{
						"langop.io/version": "invalid",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseConfigMapVersion(tt.configMap)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.Version, result.Version)
				assert.Equal(t, tt.expected.SynthesisType, result.SynthesisType)
				if tt.expected.PreviousVersion != nil {
					require.NotNil(t, result.PreviousVersion)
					assert.Equal(t, *tt.expected.PreviousVersion, *result.PreviousVersion)
				} else {
					assert.Nil(t, result.PreviousVersion)
				}
				assert.Equal(t, tt.expected.LearnedTask, result.LearnedTask)

				// Check timestamp (allow small difference due to parsing)
				if !tt.expected.CreatedAt.IsZero() {
					assert.True(t, result.CreatedAt.Equal(tt.expected.CreatedAt) ||
						result.CreatedAt.Sub(tt.expected.CreatedAt) < time.Second)
				}
			}
		})
	}
}

func TestConfigMapManager_CompressCodeData(t *testing.T) {
	logger := zap.New()
	recorder := record.NewFakeRecorder(10)

	cm := &ConfigMapManager{
		Log:      logger,
		Recorder: recorder,
	}

	tests := []struct {
		name            string
		input           string
		expectCompressed bool
		validateFunc    func(t *testing.T, result string, compressed bool)
	}{
		{
			name:            "small code not compressed",
			input:           "agent 'test' do\nend",
			expectCompressed: false,
			validateFunc: func(t *testing.T, result string, compressed bool) {
				assert.False(t, compressed)
				assert.Equal(t, "agent 'test' do\nend", result)
			},
		},
		{
			name:            "large code compressed",
			input:           generateLargeCode(900 * 1024), // 900KB - exceeds threshold
			expectCompressed: true,
			validateFunc: func(t *testing.T, result string, compressed bool) {
				assert.True(t, compressed)
				assert.True(t, len(result) > 0)
				assert.Contains(t, result, CompressionPrefix)
				// Compressed size should be smaller than original
				assert.True(t, len(result) < 900*1024)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, compressed, err := cm.compressCodeData(tt.input)
			
			require.NoError(t, err)
			assert.Equal(t, tt.expectCompressed, compressed)
			tt.validateFunc(t, result, compressed)
		})
	}
}

func TestConfigMapManager_ValidateConfigMapSize(t *testing.T) {
	logger := zap.New()
	recorder := record.NewFakeRecorder(10)

	cm := &ConfigMapManager{
		Log:      logger,
		Recorder: recorder,
	}

	tests := []struct {
		name         string
		configMapName string
		data         map[string]string
		compressed   bool
		originalSize int
		expectError  bool
		errorType    string
	}{
		{
			name:         "small configmap valid",
			configMapName: "test-v1",
			data:         map[string]string{"agent.rb": "agent 'test' do\nend"},
			compressed:   false,
			originalSize: 0,
			expectError:  false,
		},
		{
			name:         "large configmap exceeds limit",
			configMapName: "test-v1",
			data:         map[string]string{"agent.rb": generateLargeCode(1200 * 1024)}, // 1.2MB
			compressed:   false,
			originalSize: 1200 * 1024,
			expectError:  true,
			errorType:    "*synthesis.ConfigMapSizeError",
		},
		{
			name:         "compressed large data valid",
			configMapName: "test-v1",
			data:         map[string]string{"agent.rb": generateLargeCode(500 * 1024)}, // 500KB compressed
			compressed:   true,
			originalSize: 1200 * 1024,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.validateConfigMapSize(tt.configMapName, tt.data, tt.compressed, tt.originalSize)
			
			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != "" {
					_, ok := err.(*ConfigMapSizeError)
					assert.True(t, ok, "Expected ConfigMapSizeError")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigMapManager_CreateVersionedConfigMap_WithCompression(t *testing.T) {
	scheme := runtime.NewScheme()
	err := langopv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	logger := zap.New()
	recorder := record.NewFakeRecorder(10)

	cm := &ConfigMapManager{
		Client:   client,
		Scheme:   scheme,
		Log:      logger,
		Recorder: recorder,
	}

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
			UID:       "test-uid",
		},
	}

	ctx := context.Background()

	tests := []struct {
		name        string
		codeSize    int
		expectError bool
		validateFunc func(t *testing.T, cm *corev1.ConfigMap, originalSize int)
	}{
		{
			name:        "small code no compression",
			codeSize:    1024, // 1KB
			expectError: false,
			validateFunc: func(t *testing.T, cm *corev1.ConfigMap, originalSize int) {
				assert.NotContains(t, cm.Data["agent.rb"], CompressionPrefix)
				assert.Equal(t, "", cm.Annotations["langop.io/compressed"])
				assert.Equal(t, "", cm.Labels["langop.io/compressed"])
			},
		},
		{
			name:        "large code with compression",
			codeSize:    850 * 1024, // 850KB - exceeds compression threshold
			expectError: false,
			validateFunc: func(t *testing.T, cm *corev1.ConfigMap, originalSize int) {
				assert.Contains(t, cm.Data["agent.rb"], CompressionPrefix)
				assert.Equal(t, "true", cm.Annotations["langop.io/compressed"])
				assert.Equal(t, "true", cm.Labels["langop.io/compressed"])
				assert.Equal(t, fmt.Sprintf("%d", originalSize), cm.Annotations["langop.io/original-size"])
				assert.Contains(t, cm.Annotations, "langop.io/compression-ratio")
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateLargeCode(tt.codeSize)
			
			options := &ConfigMapOptions{
				Code:           code,
				Version:        int32(i + 1), // Use unique version for each test
				SynthesisType:  "learned",
				LearningSource: "test",
			}

			result, err := cm.CreateVersionedConfigMap(ctx, agent, options)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				tt.validateFunc(t, result, len(code))
			}
		})
	}
}

func TestConfigMapSizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigMapSizeError
		expected string
	}{
		{
			name: "uncompressed error",
			err: &ConfigMapSizeError{
				Name:         "test-v1",
				ActualSize:   1500000,
				MaxSize:      1048576,
				Compressed:   false,
				OriginalSize: 0,
			},
			expected: "ConfigMap test-v1 exceeds size limit: 1500000 bytes > 1048576 bytes max",
		},
		{
			name: "compressed error",
			err: &ConfigMapSizeError{
				Name:         "test-v1",
				ActualSize:   1200000,
				MaxSize:      1048576,
				Compressed:   true,
				OriginalSize: 2000000,
			},
			expected: "ConfigMap test-v1 exceeds size limit: 1200000 bytes (compressed from 2000000) > 1048576 bytes max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// generateLargeCode generates code of specified size for testing
func generateLargeCode(size int) string {
	baseCode := "agent 'test' do\n  # Large generated code\n"
	padding := "  # This is padding to make the code larger\n"
	
	code := baseCode
	for len(code) < size {
		code += padding
	}
	
	code += "end\n"
	return code[:size] // Truncate to exact size
}

// generateIncompressibleCode generates code with random data that won't compress well
func generateIncompressibleCode(size int) string {
	baseCode := "agent 'test' do\n"
	randomPart := ""
	
	// Generate random variable assignments that won't compress
	for len(baseCode + randomPart) < size-10 {
		varName := fmt.Sprintf("var_%d", len(randomPart)%1000)
		// Create pseudo-random values using a simple hash
		value := (len(randomPart)*137 + 42) % 100000
		randomPart += fmt.Sprintf("  %s = %d\n", varName, value)
	}
	
	return baseCode + randomPart + "end\n"
}
