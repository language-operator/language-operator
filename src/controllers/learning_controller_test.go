package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/language-operator/language-operator/pkg/telemetry"
)

// MockSynthesizer implements synthesis.AgentSynthesizer for testing
type MockSynthesizer struct {
	ShouldFail    bool
	GeneratedCode string
	ResponseError string
}

func (m *MockSynthesizer) SynthesizeAgent(ctx context.Context, req synthesis.AgentSynthesisRequest) (*synthesis.AgentSynthesisResponse, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("mock synthesis error")
	}

	code := m.GeneratedCode
	if code == "" {
		code = fmt.Sprintf(`task :%s do |inputs|
  # Mock learned implementation
  result = mock_optimization(inputs)
  { result: result }
end`, req.AgentName)
	}

	return &synthesis.AgentSynthesisResponse{
		DSLCode:         code,
		Error:           m.ResponseError,
		DurationSeconds: 0.1,
	}, nil
}

func (m *MockSynthesizer) DistillPersona(ctx context.Context, persona synthesis.PersonaInfo, agentContext synthesis.AgentContext) (string, error) {
	return "mock distilled persona", nil
}

func TestLearningReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	tests := []struct {
		name           string
		agent          *langopv1alpha1.LanguageAgent
		initialObjects []client.Object
		reconciler     *LearningReconciler
		expectError    bool
		validateFunc   func(t *testing.T, client client.Client, result ctrl.Result)
	}{
		{
			name: "learning disabled globally",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: langopv1alpha1.LanguageAgentSpec{
					Instructions: "test instructions",
				},
			},
			reconciler: &LearningReconciler{
				LearningEnabled: false,
			},
			expectError: false,
			validateFunc: func(t *testing.T, client client.Client, result ctrl.Result) {
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "learning disabled for agent",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
					Annotations: map[string]string{
						"langop.io/learning-disabled": "true",
					},
				},
				Spec: langopv1alpha1.LanguageAgentSpec{
					Instructions: "test instructions",
				},
			},
			reconciler: &LearningReconciler{
				LearningEnabled: true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, client client.Client, result ctrl.Result) {
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "normal learning flow with no triggers",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: langopv1alpha1.LanguageAgentSpec{
					Instructions: "test instructions",
				},
			},
			reconciler: &LearningReconciler{
				LearningEnabled:      true,
				LearningThreshold:    10,
				LearningInterval:     5 * time.Minute,
				PatternConfidenceMin: 0.7,
				Synthesizer:          &MockSynthesizer{},
			},
			expectError: false,
			validateFunc: func(t *testing.T, client client.Client, result ctrl.Result) {
				assert.Greater(t, result.RequeueAfter, time.Duration(0))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := append(tt.initialObjects, tt.agent)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			reconciler := tt.reconciler
			reconciler.Client = fakeClient
			reconciler.Scheme = scheme
			reconciler.Log = logr.Discard()
			reconciler.Recorder = &record.FakeRecorder{}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.agent.Name,
					Namespace: tt.agent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, fakeClient, result)
			}
		})
	}
}

func TestLearningReconciler_isLearningEnabled(t *testing.T) {
	tests := []struct {
		name     string
		agent    *langopv1alpha1.LanguageAgent
		expected bool
	}{
		{
			name: "learning enabled by default",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-agent",
				},
			},
			expected: true,
		},
		{
			name: "learning explicitly disabled",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-agent",
					Annotations: map[string]string{
						"langop.io/learning-disabled": "true",
					},
				},
			},
			expected: false,
		},
		{
			name: "learning annotation set to false",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-agent",
					Annotations: map[string]string{
						"langop.io/learning-disabled": "false",
					},
				},
			},
			expected: true,
		},
	}

	reconciler := &LearningReconciler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.isLearningEnabled(tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLearningReconciler_analyzeTaskPatterns(t *testing.T) {
	reconciler := &LearningReconciler{
		PatternConfidenceMin: 0.7,
	}

	tests := []struct {
		name                  string
		taskName              string
		traces                []TaskTrace
		expectedDeterministic bool
		expectedConfidence    float64
		expectedPattern       string
	}{
		{
			name:                  "no traces",
			taskName:              "test_task",
			traces:                []TaskTrace{},
			expectedDeterministic: false,
			expectedConfidence:    0.0,
			expectedPattern:       "insufficient_data",
		},
		{
			name:     "deterministic pattern",
			taskName: "test_task",
			traces: []TaskTrace{
				{
					TaskName:  "test_task",
					Success:   true,
					ToolCalls: []ToolCall{{ToolName: "fetch", Method: "get"}, {ToolName: "process", Method: "transform"}},
				},
				{
					TaskName:  "test_task",
					Success:   true,
					ToolCalls: []ToolCall{{ToolName: "fetch", Method: "get"}, {ToolName: "process", Method: "transform"}},
				},
				{
					TaskName:  "test_task",
					Success:   true,
					ToolCalls: []ToolCall{{ToolName: "fetch", Method: "get"}, {ToolName: "process", Method: "transform"}},
				},
				{
					TaskName:  "test_task",
					Success:   true,
					ToolCalls: []ToolCall{{ToolName: "fetch", Method: "get"}, {ToolName: "process", Method: "transform"}},
				},
			},
			expectedDeterministic: true,
			expectedConfidence:    0.95, // High confidence due to identical patterns (capped at 0.95)
			expectedPattern:       "simple_tool_sequence",
		},
		{
			name:     "variable pattern",
			taskName: "test_task",
			traces: []TaskTrace{
				{
					TaskName:  "test_task",
					Success:   true,
					ToolCalls: []ToolCall{{ToolName: "fetch", Method: "get"}},
				},
				{
					TaskName:  "test_task",
					Success:   true,
					ToolCalls: []ToolCall{{ToolName: "process", Method: "transform"}},
				},
				{
					TaskName:  "test_task",
					Success:   false,
					ToolCalls: []ToolCall{{ToolName: "analyze", Method: "compute"}},
				},
			},
			expectedDeterministic: false,
			expectedConfidence:    0.5, // Medium confidence based on mixed patterns
			expectedPattern:       "conditional_logic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := reconciler.analyzeTaskPatterns(tt.taskName, tt.traces)
			require.NoError(t, err)

			assert.Equal(t, tt.taskName, analysis.TaskName)
			assert.Equal(t, tt.expectedDeterministic, analysis.IsDeterministic)
			assert.Equal(t, tt.expectedPattern, analysis.CommonPattern)

			// Allow some tolerance for confidence calculations
			assert.InDelta(t, tt.expectedConfidence, analysis.Confidence, 0.1)
		})
	}
}

func TestLearningReconciler_generatePatternBasedCode(t *testing.T) {
	reconciler := &LearningReconciler{}

	tests := []struct {
		name          string
		taskName      string
		analysis      *PatternAnalysis
		shouldContain []string
	}{
		{
			name:     "deterministic tool sequence",
			taskName: "test_task",
			analysis: &PatternAnalysis{
				CommonPattern: "deterministic_tool_sequence",
				Confidence:    0.9,
			},
			shouldContain: []string{"task :test_task", "do |inputs|", "execute_optimized_sequence", "{ result: result }"},
		},
		{
			name:     "simple transformation",
			taskName: "transform_task",
			analysis: &PatternAnalysis{
				CommonPattern: "simple_transformation",
				Confidence:    0.8,
			},
			shouldContain: []string{"task :transform_task", "transform_data(inputs)", "end"},
		},
		{
			name:     "conditional logic",
			taskName: "decision_task",
			analysis: &PatternAnalysis{
				CommonPattern: "conditional_logic",
				Confidence:    0.75,
			},
			shouldContain: []string{"task :decision_task", "if condition_check", "else", "end"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := reconciler.generatePatternBasedCode(tt.taskName, tt.analysis)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, code, expected, "Generated code should contain '%s'", expected)
			}

			assert.Contains(t, code, fmt.Sprintf("confidence: %.2f", tt.analysis.Confidence))
			assert.Contains(t, code, fmt.Sprintf("Pattern: %s", tt.analysis.CommonPattern))
		})
	}
}

func TestLearningReconciler_ConfigMapManagerIntegration(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "test instructions",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		Build()

	configMapManager := &synthesis.ConfigMapManager{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	reconciler := &LearningReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		Log:              logr.Discard(),
		ConfigMapManager: configMapManager,
	}

	ctx := context.Background()
	taskName := "test_task"
	learnedCode := "mock learned code"
	version := int32(2)

	// Test using ConfigMapManager directly
	options := &synthesis.ConfigMapOptions{
		Code:           learnedCode,
		Version:        version,
		SynthesisType:  "learned",
		LearnedTask:    taskName,
		LearningSource: "pattern-detection",
	}

	_, err := reconciler.ConfigMapManager.CreateVersionedConfigMap(ctx, agent, options)
	require.NoError(t, err)

	// Verify ConfigMap was created
	configMapName := fmt.Sprintf("%s-v%d", agent.Name, version)
	var configMap corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, &configMap)
	require.NoError(t, err)

	// Verify ConfigMap content
	assert.Equal(t, learnedCode, configMap.Data["agent.rb"])
	assert.Equal(t, agent.Name, configMap.Labels["langop.io/agent"])
	assert.Equal(t, fmt.Sprintf("%d", version), configMap.Labels["langop.io/version"])
	assert.Equal(t, "learned", configMap.Labels["langop.io/synthesis-type"])
	assert.Equal(t, taskName, configMap.Labels["langop.io/learned-task"])

	// Verify owner reference
	require.Len(t, configMap.OwnerReferences, 1)
	assert.Equal(t, agent.Name, configMap.OwnerReferences[0].Name)
}

func TestLearningReconciler_findAgentDeployment(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"langop.io/agent": "test-agent",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, deployment).
		Build()

	reconciler := &LearningReconciler{
		Client: fakeClient,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	foundDeployment, err := reconciler.findAgentDeployment(ctx, agent)
	require.NoError(t, err)
	require.NotNil(t, foundDeployment)

	assert.Equal(t, deployment.Name, foundDeployment.Name)
	assert.Equal(t, deployment.Labels["langop.io/agent"], foundDeployment.Labels["langop.io/agent"])
}

func TestLearningReconciler_calculatePatternConfidence(t *testing.T) {
	reconciler := &LearningReconciler{}

	tests := []struct {
		name             string
		traces           []TaskTrace
		toolCallPatterns map[string]int
		ioConsistency    float64
		expectedMin      float64
		expectedMax      float64
	}{
		{
			name:             "no traces",
			traces:           []TaskTrace{},
			toolCallPatterns: map[string]int{},
			ioConsistency:    0.0,
			expectedMin:      0.0,
			expectedMax:      0.0,
		},
		{
			name: "high consistency",
			traces: []TaskTrace{
				{Success: true}, {Success: true}, {Success: true}, {Success: true}, {Success: true},
				{Success: true}, {Success: true}, {Success: true}, {Success: true}, {Success: true},
			},
			toolCallPatterns: map[string]int{"pattern1": 10},
			ioConsistency:    1.0,
			expectedMin:      0.95, // Should be capped at 0.95
			expectedMax:      0.95,
		},
		{
			name: "medium consistency",
			traces: []TaskTrace{
				{Success: true}, {Success: true}, {Success: false},
			},
			toolCallPatterns: map[string]int{"pattern1": 2, "pattern2": 1},
			ioConsistency:    0.67,
			expectedMin:      0.5,
			expectedMax:      0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := reconciler.calculatePatternConfidence(tt.traces, tt.toolCallPatterns, tt.ioConsistency)

			assert.GreaterOrEqual(t, confidence, tt.expectedMin)
			assert.LessOrEqual(t, confidence, tt.expectedMax)
		})
	}
}

func TestLearningReconciler_ProcessLearningTrigger_Integration(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "test instructions",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		Build()

	configMapManager := &synthesis.ConfigMapManager{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	reconciler := &LearningReconciler{
		Client:               fakeClient,
		Scheme:               scheme,
		Log:                  logr.Discard(),
		Recorder:             &record.FakeRecorder{},
		LearningInterval:     time.Minute,
		PatternConfidenceMin: 0.7,
		Synthesizer:          &MockSynthesizer{GeneratedCode: "mock learned code"},
		ConfigMapManager:     configMapManager,
	}

	ctx := context.Background()
	trigger := LearningEvent{
		AgentName:  agent.Name,
		Namespace:  agent.Namespace,
		TaskName:   "test_task",
		EventType:  "traces_accumulated",
		TraceCount: 15,
		Confidence: 0.85,
		Timestamp:  time.Now(),
	}

	learningStatus := map[string]*TaskLearningStatus{
		"test_task": {
			TaskName:       "test_task",
			CurrentVersion: 1,
			TraceCount:     15,
		},
	}

	err := reconciler.processLearningTrigger(ctx, agent, trigger, learningStatus)
	require.NoError(t, err)

	// Verify ConfigMap was created
	var configMap corev1.ConfigMap
	configMapName := fmt.Sprintf("%s-v%d", agent.Name, 2)
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, &configMap)
	require.NoError(t, err)

	// Verify learning status was updated
	status := learningStatus["test_task"]
	assert.Equal(t, int32(2), status.CurrentVersion)
	assert.True(t, status.IsSymbolic)
	assert.Equal(t, 0.85, status.PatternConfidence)
	assert.Equal(t, int32(1), status.LearningAttempts)
}

func TestLearningReconciler_generateLearnedCode(t *testing.T) {
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
	}

	tests := []struct {
		name          string
		synthesizer   *MockSynthesizer
		trigger       LearningEvent
		expectError   bool
		shouldContain []string
	}{
		{
			name: "successful synthesis",
			synthesizer: &MockSynthesizer{
				GeneratedCode: "task :learned_task do |inputs|\n  result = optimized_implementation(inputs)\n  { result: result }\nend",
			},
			trigger: LearningEvent{
				TaskName:   "learned_task",
				TraceCount: 10,
				Confidence: 0.9,
			},
			expectError:   false,
			shouldContain: []string{"task :learned_task", "optimized_implementation"},
		},
		{
			name: "synthesis service failure with fallback",
			synthesizer: &MockSynthesizer{
				ShouldFail: true,
			},
			trigger: LearningEvent{
				TaskName:   "fallback_task",
				TraceCount: 5,
				Confidence: 0.8,
			},
			expectError:   false,
			shouldContain: []string{"task :fallback_task", "execute_learned_pattern"},
		},
		{
			name: "synthesis response error",
			synthesizer: &MockSynthesizer{
				ResponseError: "synthesis validation failed",
			},
			trigger: LearningEvent{
				TaskName: "error_task",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &LearningReconciler{
				Synthesizer: tt.synthesizer,
			}

			code, err := reconciler.generateLearnedCode(context.Background(), agent, tt.trigger, map[string]*TaskLearningStatus{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for _, expected := range tt.shouldContain {
					assert.Contains(t, code, expected)
				}
			}
		})
	}
}

func TestLearningReconciler_updateAlternativeWorkload(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	tests := []struct {
		name           string
		agent          *langopv1alpha1.LanguageAgent
		workload       client.Object
		taskName       string
		version        int32
		expectUpdate   bool
		expectError    bool
		validateResult func(t *testing.T, client client.Client, workload client.Object)
	}{
		{
			name: "update CronJob ConfigMap reference",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
			},
			workload: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
					Labels: map[string]string{
						"langop.io/agent": "test-agent",
					},
				},
				Spec: batchv1.CronJobSpec{
					Schedule: "0 * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Volumes: []corev1.Volume{
										{
											Name: "agent-code",
											VolumeSource: corev1.VolumeSource{
												ConfigMap: &corev1.ConfigMapVolumeSource{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "test-agent-v1",
													},
												},
											},
										},
									},
									Containers: []corev1.Container{
										{
											Name:  "agent",
											Image: "test-image",
										},
									},
								},
							},
						},
					},
				},
			},
			taskName:     "test_task",
			version:      2,
			expectUpdate: true,
			expectError:  false,
			validateResult: func(t *testing.T, client client.Client, workload client.Object) {
				var cronJob batchv1.CronJob
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "test-agent",
					Namespace: "default",
				}, &cronJob)
				require.NoError(t, err)

				// Verify ConfigMap reference was updated
				assert.Equal(t, "test-agent-v2", cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes[0].ConfigMap.Name)

				// Verify learning annotations were added
				assert.Contains(t, cronJob.Spec.JobTemplate.Spec.Template.Annotations, "langop.io/learning-update")
				assert.Equal(t, "test-agent-v2", cronJob.Spec.JobTemplate.Spec.Template.Annotations["langop.io/learned-configmap"])
			},
		},
		{
			name: "no workload found",
			agent: &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
			},
			workload:     nil, // No workload objects
			taskName:     "test_task",
			version:      2,
			expectUpdate: false,
			expectError:  false,
			validateResult: func(t *testing.T, client client.Client, workload client.Object) {
				// Should not create any new objects
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{tt.agent}
			if tt.workload != nil {
				objects = append(objects, tt.workload)
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			reconciler := &LearningReconciler{
				Client:   fakeClient,
				Log:      logr.Discard(),
				Recorder: &record.FakeRecorder{},
			}

			ctx := context.Background()
			err := reconciler.updateAlternativeWorkload(ctx, tt.agent, tt.taskName, tt.version)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, fakeClient, tt.workload)
			}
		})
	}
}

func TestLearningReconciler_extractCronJobConfigMapReference(t *testing.T) {
	tests := []struct {
		name           string
		configMapName  string
		expectedResult string
	}{
		{
			name:           "extract from CronJob volume",
			configMapName:  "test-agent-v3",
			expectedResult: "test-agent-v3",
		},
		{
			name:           "fallback when no ConfigMap found",
			configMapName:  "",              // No ConfigMap configured
			expectedResult: "test-agent-v1", // Should fallback to v1
		},
	}

	reconciler := &LearningReconciler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronJob := &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"langop.io/agent": "test-agent",
					},
				},
				Spec: batchv1.CronJobSpec{
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{Name: "test"}},
								},
							},
						},
					},
				},
			}

			if tt.configMapName != "" {
				cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
					{
						Name: "agent-code",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: tt.configMapName,
								},
							},
						},
					},
				}
			}

			result := reconciler.extractCronJobConfigMapReference(cronJob)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// Tests for error-triggered re-synthesis functionality

func TestLearningReconciler_checkErrorTriggers(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
	}

	tests := []struct {
		name                string
		events              []corev1.Event
		learningStatus      map[string]*TaskLearningStatus
		errorThreshold      int32
		expectedTriggers    int
		expectedTaskTrigger string
	}{
		{
			name: "no triggers - below threshold",
			events: []corev1.Event{
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-10*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-5*time.Minute)),
			},
			learningStatus:      map[string]*TaskLearningStatus{},
			errorThreshold:      3,
			expectedTriggers:    0,
			expectedTaskTrigger: "",
		},
		{
			name: "trigger - reaches threshold",
			events: []corev1.Event{
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-30*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-20*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-10*time.Minute)),
			},
			learningStatus: map[string]*TaskLearningStatus{
				"fetch_data": {
					TaskName:        "fetch_data",
					TraceCount:      5,                              // Has some execution history
					LastSuccessTime: time.Now().Add(-2 * time.Hour), // Had success before
				},
			},
			errorThreshold:      3,
			expectedTriggers:    1,
			expectedTaskTrigger: "fetch_data",
		},
		{
			name: "no trigger - cooldown active",
			events: []corev1.Event{
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-30*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-20*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-10*time.Minute)),
			},
			learningStatus: map[string]*TaskLearningStatus{
				"fetch_data": {
					TaskName:            "fetch_data",
					TraceCount:          5,
					LastSuccessTime:     time.Now().Add(-2 * time.Hour),
					LastLearningAttempt: time.Now().Add(-1 * time.Minute), // Recent attempt
				},
			},
			errorThreshold:   3,
			expectedTriggers: 0,
		},
		{
			name: "no trigger - max attempts exceeded",
			events: []corev1.Event{
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-30*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-20*time.Minute)),
				createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed", time.Now().Add(-10*time.Minute)),
			},
			learningStatus: map[string]*TaskLearningStatus{
				"fetch_data": {
					TaskName:                 "fetch_data",
					TraceCount:               5,
					LastSuccessTime:          time.Now().Add(-2 * time.Hour),
					ErrorResynthesisAttempts: 3, // Already at max attempts
				},
			},
			errorThreshold:   3,
			expectedTriggers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with events
			objects := []client.Object{agent}
			for i := range tt.events {
				objects = append(objects, &tt.events[i])
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			reconciler := &LearningReconciler{
				Client:                      fakeClient,
				Log:                         logr.Discard(),
				ErrorFailureThreshold:       tt.errorThreshold,
				ErrorCooldownPeriod:         5 * time.Minute,
				MaxErrorResynthesisAttempts: 3,
			}

			ctx := context.Background()
			triggers, err := reconciler.checkErrorTriggers(ctx, agent, tt.learningStatus)

			require.NoError(t, err)

			assert.Len(t, triggers, tt.expectedTriggers)

			if tt.expectedTriggers > 0 {
				assert.Equal(t, "consecutive_failures", triggers[0].EventType)
				assert.Equal(t, tt.expectedTaskTrigger, triggers[0].TaskName)
				assert.Equal(t, float64(0.8), triggers[0].Confidence)
			}
		})
	}
}

func TestLearningReconciler_parseTaskFailureFromEvent(t *testing.T) {
	reconciler := &LearningReconciler{}

	tests := []struct {
		name         string
		event        corev1.Event
		expectNil    bool
		expectedTask string
		expectedType string
	}{
		{
			name: "valid task failure event",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Message: "Task 'fetch_data' failed with timeout error",
			},
			expectNil:    false,
			expectedTask: "fetch_data",
			expectedType: "failed",
		},
		{
			name: "normal event - should ignore",
			event: corev1.Event{
				Type:    corev1.EventTypeNormal,
				Message: "Task completed successfully",
			},
			expectNil: true,
		},
		{
			name: "no failure pattern - should ignore",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Message: "Pod started successfully",
			},
			expectNil: true,
		},
		{
			name: "runtime error with task name",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Message: "Task process_data encountered runtime error: nil pointer",
			},
			expectNil:    false,
			expectedTask: "process_data",
			expectedType: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := reconciler.parseTaskFailureFromEvent(tt.event)

			if tt.expectNil {
				assert.Nil(t, failure)
			} else {
				require.NotNil(t, failure)
				assert.Equal(t, tt.expectedTask, failure.TaskName)
				assert.Equal(t, tt.expectedType, failure.ErrorType)
				assert.Equal(t, tt.event.Message, failure.ErrorMessage)
			}
		})
	}
}

func TestLearningReconciler_updateConsecutiveFailures(t *testing.T) {
	reconciler := &LearningReconciler{}

	tests := []struct {
		name                string
		failures            []TaskFailure
		expectedConsecutive int32
		expectReset         bool
	}{
		{
			name:                "no failures",
			failures:            []TaskFailure{},
			expectedConsecutive: 0,
		},
		{
			name: "recent consecutive failures",
			failures: []TaskFailure{
				{Timestamp: time.Now().Add(-10 * time.Minute)},
				{Timestamp: time.Now().Add(-20 * time.Minute)},
				{Timestamp: time.Now().Add(-30 * time.Minute)},
			},
			expectedConsecutive: 3,
		},
		{
			name: "old failure - should reset",
			failures: []TaskFailure{
				{Timestamp: time.Now().Add(-2 * time.Hour)},
			},
			expectedConsecutive: 0,
			expectReset:         true,
		},
		{
			name: "mixed old and new failures",
			failures: []TaskFailure{
				{Timestamp: time.Now().Add(-10 * time.Minute)},
				{Timestamp: time.Now().Add(-20 * time.Minute)},
				{Timestamp: time.Now().Add(-2 * time.Hour)}, // This one is too old
			},
			expectedConsecutive: 2, // Only count the recent ones
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &TaskLearningStatus{
				TaskName: "test_task",
			}

			reconciler.updateConsecutiveFailures(status, tt.failures)

			assert.Equal(t, tt.expectedConsecutive, status.ConsecutiveFailures)

			if !tt.expectReset && len(tt.failures) > 0 {
				// Should have updated failure info
				assert.False(t, status.LastFailureTime.IsZero())
			}
		})
	}
}

func TestLearningReconciler_shouldTriggerErrorResynthesis(t *testing.T) {
	reconciler := &LearningReconciler{
		ErrorFailureThreshold:       3,
		ErrorCooldownPeriod:         5 * time.Minute,
		MaxErrorResynthesisAttempts: 3,
	}

	tests := []struct {
		name     string
		status   *TaskLearningStatus
		expected bool
		reason   string
	}{
		{
			name: "should trigger - meets all criteria",
			status: &TaskLearningStatus{
				ConsecutiveFailures:      3,
				LastLearningAttempt:      time.Now().Add(-10 * time.Minute), // Outside cooldown
				ErrorResynthesisAttempts: 1,                                 // Below max attempts
				LastSuccessTime:          time.Now().Add(-1 * time.Hour),    // Had success before
				TraceCount:               5,
			},
			expected: true,
		},
		{
			name: "below failure threshold",
			status: &TaskLearningStatus{
				ConsecutiveFailures:      2, // Below threshold
				LastLearningAttempt:      time.Now().Add(-10 * time.Minute),
				ErrorResynthesisAttempts: 1,
				LastSuccessTime:          time.Now().Add(-1 * time.Hour),
				TraceCount:               5,
			},
			expected: false,
			reason:   "below failure threshold",
		},
		{
			name: "in cooldown period",
			status: &TaskLearningStatus{
				ConsecutiveFailures:      3,
				LastLearningAttempt:      time.Now().Add(-1 * time.Minute), // Within cooldown
				ErrorResynthesisAttempts: 1,
				LastSuccessTime:          time.Now().Add(-1 * time.Hour),
				TraceCount:               5,
			},
			expected: false,
			reason:   "in cooldown",
		},
		{
			name: "max attempts exceeded",
			status: &TaskLearningStatus{
				ConsecutiveFailures:      3,
				LastLearningAttempt:      time.Now().Add(-10 * time.Minute),
				ErrorResynthesisAttempts: 3, // At max attempts
				LastSuccessTime:          time.Now().Add(-1 * time.Hour),
				TraceCount:               5,
			},
			expected: false,
			reason:   "max attempts exceeded",
		},
		{
			name: "never worked - no success history",
			status: &TaskLearningStatus{
				ConsecutiveFailures:      3,
				LastLearningAttempt:      time.Now().Add(-10 * time.Minute),
				ErrorResynthesisAttempts: 1,
				LastSuccessTime:          time.Time{}, // No success
				TraceCount:               0,           // No traces
			},
			expected: false,
			reason:   "never worked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.shouldTriggerErrorResynthesis(tt.status)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

func TestLearningReconciler_analyzeErrorPatterns(t *testing.T) {
	reconciler := &LearningReconciler{}

	failures := []TaskFailure{
		{ErrorType: "timeout", ErrorMessage: "connection timeout occurred"},
		{ErrorType: "timeout", ErrorMessage: "network timeout"},
		{ErrorMessage: "unauthorized access"},
		{ErrorMessage: "API service unavailable with status 500"},
		{ErrorMessage: "invalid input format provided"},
		{ErrorMessage: "nil pointer dereference error"},
	}

	patterns := reconciler.analyzeErrorPatterns(failures)

	// Check that patterns were detected
	assert.Equal(t, 2, patterns["timeout"])                 // Two timeout errors
	assert.Equal(t, 2, patterns["network_connectivity"])    // timeout messages count as network
	assert.Equal(t, 1, patterns["auth_errors"])             // unauthorized
	assert.Equal(t, 1, patterns["external_service_errors"]) // API 500
	assert.Equal(t, 1, patterns["input_validation_errors"]) // invalid input
	assert.Equal(t, 1, patterns["runtime_logic_errors"])    // nil pointer
}

func TestLearningReconciler_buildErrorContext(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, langopv1alpha1.AddToScheme(scheme))

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
	}

	// Create events for task failures
	events := []corev1.Event{
		createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed with timeout", time.Now().Add(-30*time.Minute)),
		createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed with connection error", time.Now().Add(-20*time.Minute)),
		createTaskFailureEvent("test-agent", "fetch_data", "Task 'fetch_data' failed with timeout", time.Now().Add(-10*time.Minute)),
	}

	objects := []client.Object{agent}
	for i := range events {
		objects = append(objects, &events[i])
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()

	reconciler := &LearningReconciler{
		Client: fakeClient,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	errorContext := reconciler.buildErrorContext(ctx, agent, "fetch_data")

	// Verify error context contains expected information
	assert.Contains(t, errorContext, "fetch_data")
	assert.Contains(t, errorContext, "3 recent failures")
	assert.Contains(t, errorContext, "timeout")
	assert.Contains(t, errorContext, "connection error")
	assert.Contains(t, errorContext, "Common error patterns")
	assert.Contains(t, errorContext, "more robust implementation")
}

// Helper function to create task failure events
func createTaskFailureEvent(agentName, taskName, message string, timestamp time.Time) corev1.Event {
	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("event-%d", timestamp.Unix()),
			Namespace: "default",
		},
		Type:           corev1.EventTypeWarning,
		Message:        message,
		FirstTimestamp: metav1.Time{Time: timestamp},
		InvolvedObject: corev1.ObjectReference{
			Name:      agentName + "-pod",
			Namespace: "default",
			Kind:      "Pod",
		},
	}
}

func TestTaskLearningStatus_SerializeParse(t *testing.T) {
	reconciler := &LearningReconciler{}

	original := &TaskLearningStatus{
		TaskName:                 "test_task",
		TraceCount:               10,
		LearningAttempts:         2,
		CurrentVersion:           3,
		IsSymbolic:               true,
		PatternConfidence:        0.85,
		ConsecutiveFailures:      2,
		ErrorResynthesisAttempts: 1,
	}

	// Serialize
	serialized := reconciler.serializeTaskLearningStatus(original)

	// Parse
	parsed, err := reconciler.parseTaskLearningStatus(serialized)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, original.TaskName, parsed.TaskName)
	assert.Equal(t, original.TraceCount, parsed.TraceCount)
	assert.Equal(t, original.LearningAttempts, parsed.LearningAttempts)
	assert.Equal(t, original.CurrentVersion, parsed.CurrentVersion)
	assert.Equal(t, original.IsSymbolic, parsed.IsSymbolic)
	assert.Equal(t, original.PatternConfidence, parsed.PatternConfidence)
	assert.Equal(t, original.ConsecutiveFailures, parsed.ConsecutiveFailures)
	assert.Equal(t, original.ErrorResynthesisAttempts, parsed.ErrorResynthesisAttempts)
}

func TestLearningReconciler_getExecutionTraces_withTelemetryAdapter(t *testing.T) {
	ctx := context.Background()

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
	}

	t.Run("adapter unavailable", func(t *testing.T) {
		// Create reconciler with unavailable adapter
		reconciler := &LearningReconciler{
			Log:              ctrl.Log.WithName("test"),
			TelemetryAdapter: telemetry.NewNoOpAdapter(),
		}

		traces, err := reconciler.getExecutionTraces(ctx, agent)
		require.NoError(t, err)
		assert.Empty(t, traces, "Should return empty traces when adapter unavailable")
	})

	t.Run("adapter available with spans", func(t *testing.T) {
		// Create mock adapter with test data
		mockSpans := []telemetry.Span{
			{
				SpanID:        "span-1",
				TraceID:       "trace-1",
				OperationName: "execute_task",
				TaskName:      "fetch_user",
				StartTime:     time.Now().Add(-time.Hour),
				EndTime:       time.Now().Add(-time.Hour).Add(2 * time.Second),
				Duration:      2 * time.Second,
				Status:        true,
				Attributes: map[string]string{
					"task.inputs":  `{"user_id": 123}`,
					"task.outputs": `{"user": {"name": "Alice"}}`,
				},
			},
			{
				SpanID:        "span-2",
				TraceID:       "trace-1",
				OperationName: "execute_task",
				TaskName:      "process_user",
				StartTime:     time.Now().Add(-30 * time.Minute),
				EndTime:       time.Now().Add(-30 * time.Minute).Add(time.Second),
				Duration:      time.Second,
				Status:        false,
				ErrorMessage:  "API timeout",
				Attributes: map[string]string{
					"task.inputs": `{"user": {"name": "Alice"}}`,
				},
			},
			{
				SpanID:        "span-3",
				TraceID:       "trace-2",
				OperationName: "tool_call", // Should be filtered out
				TaskName:      "",
				StartTime:     time.Now().Add(-15 * time.Minute),
				EndTime:       time.Now().Add(-15 * time.Minute).Add(500 * time.Millisecond),
				Duration:      500 * time.Millisecond,
				Status:        true,
			},
		}

		mockAdapter := &telemetry.MockAdapter{
			AvailableReturn: true,
			SpanResults:     mockSpans,
		}

		reconciler := &LearningReconciler{
			Log:              ctrl.Log.WithName("test"),
			TelemetryAdapter: mockAdapter,
		}

		traces, err := reconciler.getExecutionTraces(ctx, agent)
		require.NoError(t, err)

		// Should convert 2 task execution spans (filter out tool_call span)
		require.Len(t, traces, 2, "Should convert execute_task spans to TaskTrace")

		// Verify first trace
		trace1 := traces[0]
		assert.Equal(t, "fetch_user", trace1.TaskName)
		assert.Equal(t, 2*time.Second, trace1.Duration)
		assert.True(t, trace1.Success)
		assert.Empty(t, trace1.ErrorMessage)

		// Verify second trace (error case)
		trace2 := traces[1]
		assert.Equal(t, "process_user", trace2.TaskName)
		assert.Equal(t, time.Second, trace2.Duration)
		assert.False(t, trace2.Success)
		assert.Equal(t, "API timeout", trace2.ErrorMessage)
	})

	t.Run("adapter query error", func(t *testing.T) {
		// Create mock adapter that returns error
		mockAdapter := &telemetry.MockAdapter{
			AvailableReturn: true,
			SpanResults:     nil, // Will cause error in real implementation
		}

		// Override QuerySpans to return error
		errorAdapter := &mockErrorAdapter{
			MockAdapter: mockAdapter,
		}

		reconciler := &LearningReconciler{
			Log:              ctrl.Log.WithName("test"),
			TelemetryAdapter: errorAdapter,
		}

		traces, err := reconciler.getExecutionTraces(ctx, agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query execution traces")
		assert.Empty(t, traces)
	})
}

// mockErrorAdapter wraps MockAdapter to return errors for testing
type mockErrorAdapter struct {
	*telemetry.MockAdapter
}

func (m *mockErrorAdapter) QuerySpans(ctx context.Context, filter telemetry.SpanFilter) ([]telemetry.Span, error) {
	return nil, fmt.Errorf("mock telemetry error")
}

func TestLearningReconciler_convertSpansToTaskTraces(t *testing.T) {
	reconciler := &LearningReconciler{
		Log: ctrl.Log.WithName("test"),
	}

	spans := []telemetry.Span{
		{
			SpanID:        "span-1",
			TraceID:       "trace-1",
			OperationName: "execute_task",
			TaskName:      "fetch_data",
			StartTime:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2025, 1, 1, 12, 0, 2, 0, time.UTC),
			Duration:      2 * time.Second,
			Status:        true,
			Attributes: map[string]string{
				"task.inputs":  `{"id": 123}`,
				"task.outputs": `{"result": "success"}`,
				"tool.name":    "database",
			},
		},
		{
			OperationName: "tool_call", // Should be filtered out
			TaskName:      "database_query",
		},
		{
			OperationName: "execute_task",
			TaskName:      "", // Should be filtered out (empty task name)
		},
	}

	traces := reconciler.convertSpansToTaskTraces(spans)

	// Should only convert the first span
	require.Len(t, traces, 1)

	trace := traces[0]
	assert.Equal(t, "fetch_data", trace.TaskName)
	assert.Equal(t, time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), trace.Timestamp)
	assert.Equal(t, 2*time.Second, trace.Duration)
	assert.True(t, trace.Success)
	assert.Empty(t, trace.ErrorMessage)

	// parseJSONAttribute returns empty map in current implementation
	assert.Equal(t, map[string]interface{}{}, trace.Inputs)
	assert.Equal(t, map[string]interface{}{}, trace.Outputs)

	// extractToolCallsFromSpan returns empty slice in current implementation
	assert.Empty(t, trace.ToolCalls)
}
