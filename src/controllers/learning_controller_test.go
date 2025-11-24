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

			code, err := reconciler.generateLearnedCode(context.Background(), agent, tt.trigger)

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
