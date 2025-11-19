package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/pkg/synthesis"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent-with-instructions",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

	instructions := "Monitor workspace for changes"
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent-changes",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-status-agent",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
	scheme := testutil.SetupTestScheme(t)

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
	scheme := testutil.SetupTestScheme(t)

	// Test with empty ExecutionMode (should skip workload creation until synthesis detects mode)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-default-mode",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			// ExecutionMode not specified - should NOT create any workload yet
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

	// Verify NO Deployment was created (should wait for synthesis to detect mode)
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err == nil {
		t.Fatal("Expected no Deployment to exist when ExecutionMode is empty")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected NotFound error, got: %v", err)
	}

	// Verify NO CronJob was created either
	cronjob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, cronjob)
	if err == nil {
		t.Fatal("Expected no CronJob to exist when ExecutionMode is empty")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected NotFound error, got: %v", err)
	}
}

func TestLanguageAgentController_PodSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-security-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify Pod security context
	podSec := deployment.Spec.Template.Spec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}

	if podSec.RunAsNonRoot == nil || !*podSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if podSec.RunAsUser == nil || *podSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", podSec.RunAsUser)
	}

	if podSec.FSGroup == nil || *podSec.FSGroup != 101 {
		t.Errorf("Expected FSGroup to be 101, got %v", podSec.FSGroup)
	}

	if podSec.SeccompProfile == nil || podSec.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("Expected SeccompProfile type to be RuntimeDefault")
	}
}

func TestLanguageAgentController_ContainerSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-container-security-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify container security context
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in deployment")
	}

	containerSec := deployment.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext is nil")
	}

	if containerSec.AllowPrivilegeEscalation == nil || *containerSec.AllowPrivilegeEscalation {
		t.Error("Expected AllowPrivilegeEscalation to be false")
	}

	if containerSec.RunAsNonRoot == nil || !*containerSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if containerSec.RunAsUser == nil || *containerSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", containerSec.RunAsUser)
	}

	if containerSec.ReadOnlyRootFilesystem == nil || !*containerSec.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem to be true")
	}

	if containerSec.Capabilities == nil {
		t.Fatal("Capabilities is nil")
	}

	if len(containerSec.Capabilities.Drop) != 1 || containerSec.Capabilities.Drop[0] != "ALL" {
		t.Errorf("Expected capabilities to drop ALL, got %v", containerSec.Capabilities.Drop)
	}
}

func TestLanguageAgentController_TmpfsVolumes(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tmpfs-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Check for tmpfs volumes
	expectedVolumes := map[string]string{
		"tmp":         "/tmp",
		"ruby-bundle": "/home/langop/.bundle",
		"ruby-gem":    "/home/langop/.gem",
	}

	volumes := deployment.Spec.Template.Spec.Volumes
	volumeNames := make(map[string]bool)
	for _, vol := range volumes {
		volumeNames[vol.Name] = true
		// Verify it's an EmptyDir with Memory medium
		if vol.EmptyDir != nil && vol.EmptyDir.Medium == corev1.StorageMediumMemory {
			// Good - it's a tmpfs volume
		} else if _, ok := expectedVolumes[vol.Name]; ok {
			t.Errorf("Volume %s should be EmptyDir with Memory medium", vol.Name)
		}
	}

	// Check all expected volumes exist
	for volName := range expectedVolumes {
		if !volumeNames[volName] {
			t.Errorf("Expected volume %s to exist", volName)
		}
	}

	// Check volume mounts on container
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in deployment")
	}

	volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	mountPaths := make(map[string]string)
	for _, mount := range volumeMounts {
		mountPaths[mount.Name] = mount.MountPath
	}

	// Verify all expected mounts
	for volName, expectedPath := range expectedVolumes {
		if actualPath, ok := mountPaths[volName]; ok {
			if actualPath != expectedPath {
				t.Errorf("Volume %s expected to be mounted at %s, got %s", volName, expectedPath, actualPath)
			}
		} else {
			t.Errorf("Expected volume mount for %s at %s", volName, expectedPath)
		}
	}
}

func TestLanguageAgentController_CronJobSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob-security",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
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
		t.Fatalf("Expected CronJob to exist, but got error: %v", err)
	}

	// Verify Pod security context
	podSec := cronJob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}

	if podSec.RunAsNonRoot == nil || !*podSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if podSec.RunAsUser == nil || *podSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", podSec.RunAsUser)
	}

	// Verify container security context
	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in cronjob")
	}

	containerSec := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext is nil")
	}

	if containerSec.ReadOnlyRootFilesystem == nil || !*containerSec.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem to be true")
	}

	if containerSec.Capabilities == nil || len(containerSec.Capabilities.Drop) != 1 || containerSec.Capabilities.Drop[0] != "ALL" {
		t.Error("Expected capabilities to drop ALL")
	}
}

func TestLanguageAgentController_SidecarToolInjection(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create a sidecar tool
	sidecarTool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sidecar-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
			Type:           "mcp",
		},
	}

	// Create an agent that references the sidecar tool
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sidecar-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			ToolRefs: []langopv1alpha1.ToolReference{
				{
					Name:    "test-sidecar-tool",
					Enabled: true,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sidecarTool).
		WithStatusSubresource(agent, sidecarTool).
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify deployment has 2 containers: agent + sidecar tool
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) != 2 {
		t.Fatalf("Expected 2 containers (agent + sidecar), got %d", len(containers))
	}

	// Verify first container is the agent
	if containers[0].Name != "agent" {
		t.Errorf("Expected first container to be 'agent', got '%s'", containers[0].Name)
	}
	if containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected agent image '%s', got '%s'", agent.Spec.Image, containers[0].Image)
	}

	// Verify second container is the sidecar tool
	if containers[1].Name != "tool-test-sidecar-tool" {
		t.Errorf("Expected second container to be 'tool-test-sidecar-tool', got '%s'", containers[1].Name)
	}
	if containers[1].Image != sidecarTool.Spec.Image {
		t.Errorf("Expected tool image '%s', got '%s'", sidecarTool.Spec.Image, containers[1].Image)
	}

	// Verify sidecar has the correct port
	if len(containers[1].Ports) != 1 {
		t.Fatalf("Expected 1 port on sidecar, got %d", len(containers[1].Ports))
	}
	if containers[1].Ports[0].Name != "mcp" {
		t.Errorf("Expected port name 'mcp', got '%s'", containers[1].Ports[0].Name)
	}
	if containers[1].Ports[0].ContainerPort != 8080 {
		t.Errorf("Expected port 8080, got %d", containers[1].Ports[0].ContainerPort)
	}

	// Verify sidecar has readiness probe
	if containers[1].ReadinessProbe == nil {
		t.Fatal("Expected readiness probe on sidecar container")
	}
	if containers[1].ReadinessProbe.TCPSocket == nil {
		t.Fatal("Expected TCP readiness probe")
	}
	if containers[1].ReadinessProbe.TCPSocket.Port.IntVal != 8080 {
		t.Errorf("Expected readiness probe on port 8080, got %d", containers[1].ReadinessProbe.TCPSocket.Port.IntVal)
	}

	// Verify sidecar has RestartPolicy set to Always (native sidecar support)
	if containers[1].RestartPolicy == nil {
		t.Fatal("Expected RestartPolicy to be set on sidecar container")
	}
	expectedRestartPolicy := corev1.ContainerRestartPolicyAlways
	if *containers[1].RestartPolicy != expectedRestartPolicy {
		t.Errorf("Expected RestartPolicy to be %s, got %s", expectedRestartPolicy, *containers[1].RestartPolicy)
	}
}

func TestLanguageAgentController_MCPServersEnvironmentVariable(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create a sidecar tool
	sidecarTool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sidecar-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
		},
	}

	// Create a service tool
	serviceTool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/service-tool:latest",
			DeploymentMode: "service",
			Port:           9090,
		},
	}

	// Create an agent that references both tools
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcp-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			ToolRefs: []langopv1alpha1.ToolReference{
				{Name: "sidecar-tool", Enabled: true},
				{Name: "service-tool", Enabled: true},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sidecarTool, serviceTool).
		WithStatusSubresource(agent, sidecarTool, serviceTool).
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Find MCP_SERVERS environment variable on agent container
	var mcpServers string
	for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "MCP_SERVERS" {
			mcpServers = env.Value
			break
		}
	}

	if mcpServers == "" {
		t.Fatal("MCP_SERVERS environment variable not found")
	}

	// Verify MCP_SERVERS contains both localhost (sidecar) and service URLs
	if !containsString(mcpServers, "http://localhost:8080") {
		t.Errorf("Expected MCP_SERVERS to contain localhost:8080 (sidecar), got: %s", mcpServers)
	}
	if !containsString(mcpServers, "http://service-tool.default.svc.cluster.local:9090") {
		t.Errorf("Expected MCP_SERVERS to contain service URL, got: %s", mcpServers)
	}
}

func TestLanguageAgentController_SidecarWorkspaceSharing(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create a sidecar tool
	sidecarTool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workspace-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
		},
	}

	// Create an agent with workspace enabled
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workspace-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled:   true,
				Size:      "10Gi",
				MountPath: "/workspace",
			},
			ToolRefs: []langopv1alpha1.ToolReference{
				{Name: "workspace-tool", Enabled: true},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sidecarTool).
		WithStatusSubresource(agent, sidecarTool).
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify deployment has workspace volume
	var workspaceVolumeExists bool
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		if vol.Name == "workspace" {
			workspaceVolumeExists = true
			if vol.PersistentVolumeClaim == nil {
				t.Error("Expected workspace volume to be PVC")
			}
			break
		}
	}
	if !workspaceVolumeExists {
		t.Fatal("Workspace volume not found")
	}

	// Verify both agent and sidecar mount the workspace
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) != 2 {
		t.Fatalf("Expected 2 containers, got %d", len(containers))
	}

	// Check agent container has workspace mount
	var agentHasWorkspace bool
	for _, mount := range containers[0].VolumeMounts {
		if mount.Name == "workspace" && mount.MountPath == "/workspace" {
			agentHasWorkspace = true
			break
		}
	}
	if !agentHasWorkspace {
		t.Error("Agent container doesn't have workspace mount")
	}

	// Check sidecar container has workspace mount
	var sidecarHasWorkspace bool
	for _, mount := range containers[1].VolumeMounts {
		if mount.Name == "workspace" && mount.MountPath == "/workspace" {
			sidecarHasWorkspace = true
			break
		}
	}
	if !sidecarHasWorkspace {
		t.Error("Sidecar container doesn't have workspace mount")
	}
}

func TestLanguageAgentController_MultipleSidecarTools(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create multiple sidecar tools
	tool1 := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tool-one",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool1:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
		},
	}

	tool2 := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tool-two",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool2:latest",
			DeploymentMode: "sidecar",
			Port:           8081,
		},
	}

	tool3 := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tool-three",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool3:latest",
			DeploymentMode: "sidecar",
			Port:           8082,
		},
	}

	// Create an agent that references all three tools
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-multi-sidecar-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			ToolRefs: []langopv1alpha1.ToolReference{
				{Name: "tool-one", Enabled: true},
				{Name: "tool-two", Enabled: true},
				{Name: "tool-three", Enabled: true},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, tool1, tool2, tool3).
		WithStatusSubresource(agent, tool1, tool2, tool3).
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify deployment has 4 containers: agent + 3 sidecars
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) != 4 {
		t.Fatalf("Expected 4 containers (agent + 3 sidecars), got %d", len(containers))
	}

	// Verify container names
	expectedContainers := map[string]bool{
		"agent":           false,
		"tool-tool-one":   false,
		"tool-tool-two":   false,
		"tool-tool-three": false,
	}

	for _, container := range containers {
		if _, expected := expectedContainers[container.Name]; expected {
			expectedContainers[container.Name] = true
		}
	}

	for name, found := range expectedContainers {
		if !found {
			t.Errorf("Expected container '%s' not found", name)
		}
	}

	// Verify MCP_SERVERS contains all three localhost URLs
	var mcpServers string
	for _, env := range containers[0].Env {
		if env.Name == "MCP_SERVERS" {
			mcpServers = env.Value
			break
		}
	}

	expectedURLs := []string{
		"http://localhost:8080",
		"http://localhost:8081",
		"http://localhost:8082",
	}

	for _, url := range expectedURLs {
		if !containsString(mcpServers, url) {
			t.Errorf("Expected MCP_SERVERS to contain '%s', got: %s", url, mcpServers)
		}
	}
}

func TestLanguageAgentController_SidecarToolsInCronJob(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create a sidecar tool
	sidecarTool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cronjob-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
		},
	}

	// Create a scheduled agent with sidecar tool
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob-sidecar-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "scheduled",
			Schedule:      "0 * * * *",
			ToolRefs: []langopv1alpha1.ToolReference{
				{Name: "cronjob-tool", Enabled: true},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sidecarTool).
		WithStatusSubresource(agent, sidecarTool).
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
		t.Fatalf("Expected CronJob to exist, but got error: %v", err)
	}

	// Verify CronJob has 2 containers: agent + sidecar
	containers := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
	if len(containers) != 2 {
		t.Fatalf("Expected 2 containers in CronJob (agent + sidecar), got %d", len(containers))
	}

	// Verify sidecar container
	if containers[1].Name != "tool-cronjob-tool" {
		t.Errorf("Expected sidecar container 'tool-cronjob-tool', got '%s'", containers[1].Name)
	}

	// Verify MCP_SERVERS in agent container
	var mcpServers string
	for _, env := range containers[0].Env {
		if env.Name == "MCP_SERVERS" {
			mcpServers = env.Value
			break
		}
	}

	if !containsString(mcpServers, "http://localhost:8080") {
		t.Errorf("Expected MCP_SERVERS to contain localhost:8080, got: %s", mcpServers)
	}

	// Verify sidecar has RestartPolicy set to Always (native sidecar support)
	if containers[1].RestartPolicy == nil {
		t.Fatal("Expected RestartPolicy to be set on sidecar container in CronJob")
	}
	expectedRestartPolicy := corev1.ContainerRestartPolicyAlways
	if *containers[1].RestartPolicy != expectedRestartPolicy {
		t.Errorf("Expected RestartPolicy to be %s, got %s", expectedRestartPolicy, *containers[1].RestartPolicy)
	}
}

func TestLanguageAgentController_CrossNamespaceTools(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create a tool in a different namespace
	toolNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tools",
		},
	}

	crossNSTool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cross-ns-tool",
			Namespace: "tools",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image:          "ghcr.io/language-operator/tool:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
		},
	}

	// Create an agent in default namespace
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cross-ns-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			ToolRefs: []langopv1alpha1.ToolReference{
				{
					Name:      "cross-ns-tool",
					Namespace: "tools", // Reference tool from different namespace
					Enabled:   true,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(toolNamespace, agent, crossNSTool).
		WithStatusSubresource(agent, crossNSTool).
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
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify sidecar was injected
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) != 2 {
		t.Fatalf("Expected 2 containers (agent + cross-namespace sidecar), got %d", len(containers))
	}

	if containers[1].Name != "tool-cross-ns-tool" {
		t.Errorf("Expected sidecar 'tool-cross-ns-tool', got '%s'", containers[1].Name)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
