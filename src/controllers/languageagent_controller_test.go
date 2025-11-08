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
			DSLCode:         "require 'language-operator'\n\nagent 'test' do\n  schedule every: '1h'\nend",
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
			Image: "git.theryans.io/language-operator/agent:latest",
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
			Image:        "git.theryans.io/language-operator/agent:latest",
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
			Image:        "git.theryans.io/language-operator/agent:latest",
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
			"agent.rb": "require 'language-operator'\n\nagent 'test' do\n  schedule every: '1h'\nend",
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

func TestLanguageAgentController_DeploymentCreation(t *testing.T) {
	scheme := setupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "git.theryans.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist for autonomous agent, but got error: %v", err)
	}

	// Verify Deployment has correct image
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected image '%s', got '%s'", agent.Spec.Image, deployment.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestLanguageAgentController_CronJobCreation(t *testing.T) {
	scheme := setupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "git.theryans.io/language-operator/agent:latest",
			ExecutionMode: "scheduled",
			Schedule:      "0 * * * *",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify CronJob was created
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, cronJob)
	if err != nil {
		t.Fatalf("Expected CronJob to exist for scheduled agent, but got error: %v", err)
	}

	// Verify CronJob schedule
	if cronJob.Spec.Schedule != agent.Spec.Schedule {
		t.Errorf("Expected schedule '%s', got '%s'", agent.Spec.Schedule, cronJob.Spec.Schedule)
	}

	// Verify CronJob has correct image
	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	}
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected image '%s', got '%s'", agent.Spec.Image, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestLanguageAgentController_WorkspacePVCCreation(t *testing.T) {
	scheme := setupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "git.theryans.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: true,
				Size:    "10Gi",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify PVC was created
	pvc := &corev1.PersistentVolumeClaim{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name + "-workspace",
		Namespace: agent.Namespace,
	}, pvc)
	if err != nil {
		t.Fatalf("Expected PVC to exist when workspace is enabled, but got error: %v", err)
	}

	// Verify PVC size
	requestedStorage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	expectedStorage := agent.Spec.Workspace.Size
	if requestedStorage.String() != expectedStorage {
		t.Errorf("Expected storage size '%s', got '%s'", expectedStorage, requestedStorage.String())
	}
}

func TestLanguageAgentController_StatusConditions(t *testing.T) {
	scheme := setupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-status-agent",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "git.theryans.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Fetch updated agent
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to fetch updated agent: %v", err)
	}

	// Verify status phase
	if updatedAgent.Status.Phase != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", updatedAgent.Status.Phase)
	}

	// Verify Ready condition
	var readyCondition *metav1.Condition
	for i := range updatedAgent.Status.Conditions {
		if updatedAgent.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedAgent.Status.Conditions[i]
			break
		}
	}
	if readyCondition == nil {
		t.Fatal("Ready condition not found")
	}
	if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected condition status True, got %s", readyCondition.Status)
	}
	if readyCondition.Reason != "ReconcileSuccess" {
		t.Errorf("Expected reason 'ReconcileSuccess', got '%s'", readyCondition.Reason)
	}
}

func TestLanguageAgentController_NotFoundHandling(t *testing.T) {
	scheme := setupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-agent",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found agent, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found agent")
	}
}

func TestLanguageAgentController_DefaultExecutionMode(t *testing.T) {
	scheme := setupTestScheme(t)

	// Test with empty ExecutionMode (should default to autonomous/Deployment)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-default-mode",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "git.theryans.io/language-operator/agent:latest",
			// ExecutionMode not specified - should create Deployment
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify Deployment was created (default behavior for empty ExecutionMode)
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist for default execution mode, but got error: %v", err)
	}
}
