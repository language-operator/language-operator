package controllers

import (
	"context"
	"testing"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	"github.com/based/language-operator/pkg/synthesis"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// setupTestScheme creates a scheme with all required types registered
func setupTestScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := langopv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add langop scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add core scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add apps scheme: %v", err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add networking scheme: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add batch scheme: %v", err)
	}
	return scheme
}

// MockSynthesizer implements a mock synthesizer for testing
type MockSynthesizer struct {
	SynthesizeCalled int
	LastRequest      *synthesis.AgentSynthesisRequest
	Response         *synthesis.AgentSynthesisResponse
	Error            error
}

func (m *MockSynthesizer) SynthesizeAgent(ctx context.Context, req synthesis.AgentSynthesisRequest) (*synthesis.AgentSynthesisResponse, error) {
	m.SynthesizeCalled++
	m.LastRequest = &req
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response == nil {
		return &synthesis.AgentSynthesisResponse{
			DSLCode:         "require 'langop'\n\nagent 'test' do\n  schedule every: '1h'\nend",
			DurationSeconds: 1.5,
		}, nil
	}
	return m.Response, nil
}

func (m *MockSynthesizer) DistillPersona(ctx context.Context, persona synthesis.PersonaInfo, agentContext synthesis.AgentContext) (string, error) {
	return "Test persona description", nil
}

func TestLanguageAgentController_SynthesisNotCalledWithoutInstructions(t *testing.T) {
	scheme := setupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "langop/agent:latest",
			// No Instructions field - synthesis should not be called
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	mockSynth := &MockSynthesizer{}
	reconciler := &LanguageAgentReconciler{
		Client:         fakeClient,
		Scheme:         scheme,
		Log:            logr.Discard(),
		Synthesizer:    mockSynth,
		SynthesisModel: "test-model",
		Recorder:       &record.FakeRecorder{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify synthesizer was not called
	if mockSynth.SynthesizeCalled != 0 {
		t.Errorf("Expected synthesizer not to be called without instructions, but was called %d times", mockSynth.SynthesizeCalled)
	}
}

func TestLanguageAgentController_SynthesisCalledWithInstructions(t *testing.T) {
	scheme := setupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent-with-instructions",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "langop/agent:latest",
			Instructions: "Monitor workspace and report changes every hour",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	mockSynth := &MockSynthesizer{}
	reconciler := &LanguageAgentReconciler{
		Client:         fakeClient,
		Scheme:         scheme,
		Log:            logr.Discard(),
		Synthesizer:    mockSynth,
		SynthesisModel: "test-model",
		Recorder:       &record.FakeRecorder{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify synthesizer was called
	if mockSynth.SynthesizeCalled != 1 {
		t.Errorf("Expected synthesizer to be called once, but was called %d times", mockSynth.SynthesizeCalled)
	}

	// Verify request contains instructions
	if mockSynth.LastRequest == nil {
		t.Fatal("Expected synthesis request to be captured")
	}
	if mockSynth.LastRequest.Instructions != agent.Spec.Instructions {
		t.Errorf("Expected instructions '%s', got '%s'", agent.Spec.Instructions, mockSynth.LastRequest.Instructions)
	}

	// Verify ConfigMap was created with synthesized code
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "code"),
		Namespace: agent.Namespace,
	}, cm)
	if err != nil {
		t.Fatalf("Expected code ConfigMap to exist, but got error: %v", err)
	}

	// Verify ConfigMap contains code
	if cm.Data["agent.rb"] == "" {
		t.Error("Expected ConfigMap to contain agent.rb with code")
	}

	// Verify ConfigMap has synthesis annotations
	if cm.Annotations["langop.io/instructions-hash"] == "" {
		t.Error("Expected ConfigMap to have instructions-hash annotation")
	}
}

func TestLanguageAgentController_SmartChangeDetection(t *testing.T) {
	scheme := setupTestScheme(t)

	instructions := "Monitor workspace for changes"
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent-changes",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "langop/agent:latest",
			Instructions: instructions,
		},
	}

	// Pre-create ConfigMap with existing code
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenerateConfigMapName(agent.Name, "code"),
			Namespace: agent.Namespace,
			Annotations: map[string]string{
				"langop.io/instructions-hash": hashString(instructions),
				"langop.io/tools-hash":        hashString(""),
				"langop.io/models-hash":       hashString(""),
				"langop.io/persona-hash":      hashString(""),
			},
		},
		Data: map[string]string{
			"agent.rb": "require 'langop'\n\nagent 'test' do\n  schedule every: '1h'\nend",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, existingCM).
		WithStatusSubresource(agent).
		Build()

	mockSynth := &MockSynthesizer{}
	reconciler := &LanguageAgentReconciler{
		Client:         fakeClient,
		Scheme:         scheme,
		Log:            logr.Discard(),
		Synthesizer:    mockSynth,
		SynthesisModel: "test-model",
		Recorder:       &record.FakeRecorder{},
	}

	ctx := context.Background()

	// First reconcile with same instructions - should not re-synthesize
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	if mockSynth.SynthesizeCalled != 0 {
		t.Errorf("Expected no synthesis when instructions unchanged, but was called %d times", mockSynth.SynthesizeCalled)
	}

	// Refetch agent to get the latest version
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent); err != nil {
		t.Fatalf("Failed to refetch agent: %v", err)
	}

	// Update instructions - should trigger re-synthesis
	updatedAgent.Spec.Instructions = "Monitor workspace and alert on critical changes"
	if err := fakeClient.Update(ctx, updatedAgent); err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	if mockSynth.SynthesizeCalled != 1 {
		t.Errorf("Expected synthesis when instructions changed, but was called %d times", mockSynth.SynthesizeCalled)
	}
}
