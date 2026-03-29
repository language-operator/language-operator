package controllers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Note: Synthesis is now configured per-agent via Models
// Tests that need to verify synthesis behavior require integration tests with actual LanguageModel resources

func TestLanguageAgentController_NoSynthesisWithoutModels(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "Do something", // Has instructions but no Models
			// No Models - synthesis should not happen
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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

	// Verify no code ConfigMap was created (no Models means no synthesis)
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "code"),
		Namespace: agent.Namespace,
	}, cm)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected no code ConfigMap without ModelRefs, but found one or got error: %v", err)
	}
}

// Note: TestLanguageAgentController_SynthesisCalledWithInstructions and
// TestLanguageAgentController_SmartChangeDetection were removed because synthesis
// is now configured per-agent via Models. Testing synthesis behavior requires
// integration tests with actual LanguageModel CRDs and Secrets.

func TestLanguageAgentController_DeploymentCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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

func TestLanguageAgentController_WorkspacePVCCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := true; return &b }(),
				Size:    "10Gi",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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

func TestLanguageAgentController_ReplicaStatusSync(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "replica-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	// First reconcile: adds finalizer
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("first Reconcile failed: %v", err)
	}

	// Seed the Deployment status to simulate the real Deployment controller
	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist after first reconcile: %v", err)
	}
	deployment.Status.Replicas = 2
	deployment.Status.ReadyReplicas = 1
	if err := fakeClient.Status().Update(ctx, deployment); err != nil {
		t.Fatalf("Failed to seed Deployment status: %v", err)
	}

	// Second reconcile: should pick up replica counts
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("second Reconcile failed: %v", err)
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updatedAgent); err != nil {
		t.Fatalf("Failed to fetch updated agent: %v", err)
	}

	if updatedAgent.Status.ActiveReplicas != 2 {
		t.Errorf("expected ActiveReplicas=2, got %d", updatedAgent.Status.ActiveReplicas)
	}
	if updatedAgent.Status.ReadyReplicas != 1 {
		t.Errorf("expected ReadyReplicas=1, got %d", updatedAgent.Status.ReadyReplicas)
	}
}

func TestLanguageAgentController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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

func TestLanguageAgentController_PodSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-security-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
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
		"tmp": "/tmp",
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

func TestLanguageAgentController_ResourceCleanup(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
			Finalizers: []string{FinalizerName},
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Test agent for cleanup",
		},
	}

	// Create resources that should be cleaned up
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Service to cleanup
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 80}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, service).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()

	// Run reconcile - should trigger cleanup since agent has DeletionTimestamp
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify the service was deleted
	svc := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, svc)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected service to be deleted, but it still exists or got different error: %v", err)
	}

	// Verify the agent was either deleted or finalizer was removed
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)

	if errors.IsNotFound(err) {
		// Agent was fully deleted - this is expected and good
		t.Log("Agent was successfully deleted after cleanup")
	} else if err != nil {
		t.Fatalf("Unexpected error getting updated agent: %v", err)
	} else {
		// Agent still exists, check that finalizer was removed
		for _, finalizer := range updatedAgent.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}
}

func TestLanguageAgentController_UUIDAssignmentRaceCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
		// Status.UUID should be empty initially
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()

	// First reconcile should assign UUID
	result1, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Fetch updated agent to get UUID
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to fetch updated agent: %v", err)
	}

	// Verify UUID was assigned
	if updatedAgent.Status.UUID == "" {
		t.Fatal("Expected UUID to be assigned on first reconcile")
	}
	firstUUID := updatedAgent.Status.UUID

	// Second reconcile should NOT change the UUID
	result2, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Fetch agent again
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to fetch agent after second reconcile: %v", err)
	}

	// Verify UUID remained the same
	if updatedAgent.Status.UUID != firstUUID {
		t.Errorf("Expected UUID to remain %s, but got %s", firstUUID, updatedAgent.Status.UUID)
	}

	// Both results should not requeue for UUID reasons
	if result1.Requeue || result2.Requeue {
		t.Error("Reconciles should not requeue when UUID assignment succeeds")
	}
}

func TestLanguageAgentController_UUIDConflictHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-conflict-agent",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
		Status: langopv1alpha1.LanguageAgentStatus{},
	}

	// Create a client that will simulate version conflicts on status updates
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()

	// Simulate updating the agent's observed generation externally (as if another reconciler updated it)
	// This would happen in practice when multiple reconcilers are running
	err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, agent)
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	// Update the agent to have newer generation to simulate conflict conditions
	agent.Generation = 2
	err = fakeClient.Update(ctx, agent)
	if err != nil {
		t.Fatalf("Failed to update agent generation: %v", err)
	}

	// Now reconcile with the old agent object (ObservedGeneration: 0, but actual Generation: 2)
	// This should trigger the UUID assignment logic
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})

	// The reconcile should succeed (not return an error) even if there's a conflict
	// The conflict handling should cause a requeue, not an error
	if err != nil {
		t.Fatalf("Reconcile should handle conflicts gracefully, but got error: %v", err)
	}

	// Verify agent eventually has UUID assigned
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to get updated agent: %v", err)
	}

	// Should have UUID assigned
	if updatedAgent.Status.UUID == "" {
		t.Error("Expected UUID to be assigned after conflict resolution")
	}
}

func TestLanguageAgentController_CleanupMethods(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Test agent for cleanup methods",
		},
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Create a service that should be cleaned up
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 80}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, service).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()

	t.Run("cleanupServices", func(t *testing.T) {
		// Test service cleanup
		err := reconciler.cleanupServices(ctx, agent)
		if err != nil {
			t.Fatalf("cleanupServices failed: %v", err)
		}

		// Verify service was deleted
		svc := &corev1.Service{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-service",
			Namespace: agent.Namespace,
		}, svc)
		if !errors.IsNotFound(err) {
			t.Errorf("Expected service to be deleted, but it still exists or got different error: %v", err)
		}
	})

	t.Run("cleanupIngresses_empty_list", func(t *testing.T) {
		// Test Ingress cleanup with no ingresses present
		err := reconciler.cleanupIngresses(ctx, agent)
		if err != nil {
			t.Errorf("cleanupIngresses should handle empty list gracefully, got error: %v", err)
		}
	})

}

func TestLanguageAgentController_BasicReconcile(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment created
	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	// Verify Service created
	svc := &corev1.Service{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc); err != nil {
		t.Fatalf("Expected Service to exist: %v", err)
	}

	// Verify agent ConfigMap created (contains instructions + config)
	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: GenerateConfigMapName(agent.Name, "agent"), Namespace: agent.Namespace}, cm); err != nil {
		t.Fatalf("Expected agent ConfigMap to exist: %v", err)
	}
}

func TestLanguageAgentController_EnvVarInjection(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "env-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "test instructions",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers in deployment")
	}

	envMap := make(map[string]string)
	for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		envMap[e.Name] = e.Value
	}

	if envMap["AGENT_NAME"] != agent.Name {
		t.Errorf("Expected AGENT_NAME=%s, got %s", agent.Name, envMap["AGENT_NAME"])
	}
	if envMap["AGENT_NAMESPACE"] != agent.Namespace {
		t.Errorf("Expected AGENT_NAMESPACE=%s, got %s", agent.Namespace, envMap["AGENT_NAMESPACE"])
	}
}

func TestLanguageAgentController_ResourceRequests(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers in deployment")
	}

	res := deployment.Spec.Template.Spec.Containers[0].Resources
	if res.Requests.Cpu().Cmp(resource.MustParse("200m")) != 0 {
		t.Errorf("Expected CPU request 200m, got %s", res.Requests.Cpu().String())
	}
	if res.Requests.Memory().Cmp(resource.MustParse("256Mi")) != 0 {
		t.Errorf("Expected memory request 256Mi, got %s", res.Requests.Memory().String())
	}
	if res.Limits.Cpu().Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("Expected CPU limit 500m, got %s", res.Limits.Cpu().String())
	}
	if res.Limits.Memory().Cmp(resource.MustParse("512Mi")) != 0 {
		t.Errorf("Expected memory limit 512Mi, got %s", res.Limits.Memory().String())
	}
}

func TestLanguageAgentController_ServiceAccountCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify ServiceAccount created in agent namespace
	sa := &corev1.ServiceAccount{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: agent.Namespace}, sa); err != nil {
		t.Fatalf("Expected ServiceAccount 'language-agent' to exist in namespace %s: %v", agent.Namespace, err)
	}

	// Verify ClusterRoleBinding created
	crb := &rbacv1.ClusterRoleBinding{}
	crbName := "language-agent-" + agent.Namespace + "-language-agent"
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: crbName}, crb); err != nil {
		t.Fatalf("Expected ClusterRoleBinding %q to exist: %v", crbName, err)
	}
	if crb.RoleRef.Name != "language-operator" {
		t.Errorf("Expected ClusterRoleBinding to reference 'language-operator' ClusterRole, got %q", crb.RoleRef.Name)
	}
}

// --- Group 1: Pure/stateless functions ---

func TestLanguageAgentController_HashString(t *testing.T) {
	h1 := hashString("hello")
	h2 := hashString("hello")
	h3 := hashString("world")

	if h1 != h2 {
		t.Errorf("hashString not deterministic: %s vs %s", h1, h2)
	}
	if h1 == h3 {
		t.Error("hashString should differ for different inputs")
	}
	if h1 == "" {
		t.Error("hashString returned empty string")
	}
}

func TestLanguageAgentController_GetNames(t *testing.T) {
	r := &LanguageAgentReconciler{}

	t.Run("get_tool_names", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{
					{Name: "tool-a"},
					{Name: "tool-b"},
				},
			},
		}
		names := r.getToolNames(agent)
		if len(names) != 2 || names[0] != "tool-a" || names[1] != "tool-b" {
			t.Errorf("getToolNames: got %v", names)
		}
	})

	t.Run("get_model_names", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{
					{Name: "model-x"},
				},
			},
		}
		names := r.getModelNames(agent)
		if len(names) != 1 || names[0] != "model-x" {
			t.Errorf("getModelNames: got %v", names)
		}
	})

	t.Run("get_persona_names", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "persona-1",
			},
		}
		names := r.getPersonaNames(agent)
		if len(names) != 1 || names[0] != "persona-1" {
			t.Errorf("getPersonaNames: got %v", names)
		}
	})

	t.Run("empty_refs", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{}
		if got := r.getToolNames(agent); len(got) != 0 {
			t.Errorf("expected empty tool names, got %v", got)
		}
		if got := r.getModelNames(agent); len(got) != 0 {
			t.Errorf("expected empty model names, got %v", got)
		}
		if got := r.getPersonaNames(agent); len(got) != 0 {
			t.Errorf("expected empty persona names, got %v", got)
		}
	})
}

// --- Group 2: NetworkPolicy ---

func TestLanguageAgentController_NetworkPolicy(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "np-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:               fakeClient,
		Scheme:               scheme,
		Log:                  logr.Discard(),
		Recorder:             &record.FakeRecorder{},
		RegistryManager:      &mockRegistryManager{},
		NetworkPolicyTimeout: 30 * time.Second,
		NetworkPolicyRetries: 3,
	}

	ctx := context.Background()
	if err := reconciler.reconcileNetworkPolicy(ctx, agent); err != nil {
		t.Fatalf("reconcileNetworkPolicy failed: %v", err)
	}

	t.Run("network_policy_created", func(t *testing.T) {
		np := &networkingv1.NetworkPolicy{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np); err != nil {
			t.Fatalf("expected NetworkPolicy to exist: %v", err)
		}
	})

	t.Run("network_policy_has_ingress_rules", func(t *testing.T) {
		np := &networkingv1.NetworkPolicy{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np); err != nil {
			t.Fatalf("NetworkPolicy not found: %v", err)
		}
		if len(np.Spec.Ingress) == 0 {
			t.Error("expected ingress rules to be set on NetworkPolicy")
		}
		hasIngress := false
		for _, pt := range np.Spec.PolicyTypes {
			if pt == networkingv1.PolicyTypeIngress {
				hasIngress = true
			}
		}
		if !hasIngress {
			t.Error("expected PolicyTypeIngress to be in PolicyTypes")
		}
	})
}

// --- Group 3: Ingress ---

func TestLanguageAgentController_IngressCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	t.Run("ingress_with_class_name", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()

		reconciler := &LanguageAgentReconciler{
			Client:                  fakeClient,
			Scheme:                  scheme,
			Log:                     logr.Discard(),
			Recorder:                &record.FakeRecorder{},
			RegistryManager:         &mockRegistryManager{},
			DefaultIngressClassName: "nginx",
		}

		ctx := context.Background()
		if err := reconciler.reconcileIngress(ctx, agent, "agent.example.com"); err != nil {
			t.Fatalf("reconcileIngress failed: %v", err)
		}

		ing := &networkingv1.Ingress{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing); err != nil {
			t.Fatalf("expected Ingress to exist: %v", err)
		}
		if ing.Spec.IngressClassName == nil || *ing.Spec.IngressClassName != "nginx" {
			t.Errorf("expected IngressClassName 'nginx', got %v", ing.Spec.IngressClassName)
		}
	})

	t.Run("ingress_without_class_name", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()

		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}

		ctx := context.Background()
		if err := reconciler.reconcileIngress(ctx, agent, "agent.example.com"); err != nil {
			t.Fatalf("reconcileIngress failed: %v", err)
		}

		ing := &networkingv1.Ingress{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing); err != nil {
			t.Fatalf("expected Ingress to exist: %v", err)
		}
		if ing.Spec.IngressClassName != nil {
			t.Errorf("expected no IngressClassName, got %v", *ing.Spec.IngressClassName)
		}
	})

	t.Run("ingress_has_correct_host_and_path", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()

		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}

		ctx := context.Background()
		hostname := "my-agent.example.com"
		if err := reconciler.reconcileIngress(ctx, agent, hostname); err != nil {
			t.Fatalf("reconcileIngress failed: %v", err)
		}

		ing := &networkingv1.Ingress{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing); err != nil {
			t.Fatalf("expected Ingress to exist: %v", err)
		}
		if len(ing.Spec.Rules) == 0 {
			t.Fatal("expected at least one ingress rule")
		}
		if ing.Spec.Rules[0].Host != hostname {
			t.Errorf("expected host %q, got %q", hostname, ing.Spec.Rules[0].Host)
		}
		if ing.Spec.Rules[0].HTTP == nil || len(ing.Spec.Rules[0].HTTP.Paths) == 0 {
			t.Fatal("expected HTTP paths on ingress rule")
		}
		if ing.Spec.Rules[0].HTTP.Paths[0].Path != "/" {
			t.Errorf("expected path '/', got %q", ing.Spec.Rules[0].HTTP.Paths[0].Path)
		}
	})
}

func TestLanguageAgentController_CheckIngressReadiness(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("not_found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "no-ingress", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ready {
			t.Error("expected not ready when ingress not found")
		}
	})

	t.Run("no_lb_ip", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "default"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ing).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "my-ing", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ready {
			t.Error("expected not ready when no LB ingress points")
		}
	})

	t.Run("lb_with_ip", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "default"},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{
						{IP: "10.0.0.1"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ing).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "my-ing", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ready {
			t.Error("expected ready when LB has IP")
		}
	})

	t.Run("lb_with_hostname", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "default"},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{
						{Hostname: "lb.example.com"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ing).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "my-ing", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ready {
			t.Error("expected ready when LB has hostname")
		}
	})
}

// --- Group 4: Resource resolution and persona ---

func TestLanguageAgentController_ResolveTools(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("service_mode_tool", func(t *testing.T) {
		tool := gen.LanguageTool("my-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "my-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 1 {
			t.Fatalf("expected 1 URL, got %d", len(urls))
		}
		expected := "http://my-tool.default.svc.cluster.local:8080"
		if urls[0] != expected {
			t.Errorf("expected %q, got %q", expected, urls[0])
		}
	})

	t.Run("sidecar_mode_tool", func(t *testing.T) {
		tool := gen.LanguageTool("sidecar-tool", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolPort(9090),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "sidecar-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 1 {
			t.Fatalf("expected 1 URL, got %d", len(urls))
		}
		expected := "http://localhost:9090"
		if urls[0] != expected {
			t.Errorf("expected %q, got %q", expected, urls[0])
		}
	})

	t.Run("missing_tool", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "nonexistent"}},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		_, err := r.resolveTools(context.Background(), agent)
		if err == nil {
			t.Error("expected error for missing tool")
		}
	})

	t.Run("custom_port", func(t *testing.T) {
		tool := gen.LanguageTool("port-tool", "default",
			gen.SetToolDeploymentMode("service"),
			gen.SetToolPort(9999),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "port-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "http://port-tool.default.svc.cluster.local:9999"
		if len(urls) != 1 || urls[0] != expected {
			t.Errorf("expected %q, got %v", expected, urls)
		}
	})

	t.Run("disabled_tool_skipped", func(t *testing.T) {
		tool := gen.LanguageTool("disabled-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		disabled := false
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "disabled-tool", Enabled: &disabled}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 0 {
			t.Errorf("expected 0 URLs for disabled tool, got %d: %v", len(urls), urls)
		}
	})

	t.Run("explicitly_enabled_tool_included", func(t *testing.T) {
		tool := gen.LanguageTool("enabled-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		enabled := true
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "enabled-tool", Enabled: &enabled}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 1 {
			t.Errorf("expected 1 URL for explicitly enabled tool, got %d", len(urls))
		}
	})
}

func TestLanguageAgentController_ResolveModels(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("single_model", func(t *testing.T) {
		model := gen.LanguageModel("my-model", "default",
			gen.SetModelName("claude-3-sonnet"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{{Name: "my-model"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(model).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, names, err := r.resolveModels(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// All models in a namespace share the single cluster proxy
		expectedURL := "http://proxy.default.svc.cluster.local:8000"
		if len(urls) != 1 || urls[0] != expectedURL {
			t.Errorf("unexpected URLs: %v", urls)
		}
		if len(names) != 1 || names[0] != "claude-3-sonnet" {
			t.Errorf("unexpected names: %v", names)
		}
	})

	t.Run("multiple_models", func(t *testing.T) {
		m1 := gen.LanguageModel("model-a", "default", gen.SetModelName("gpt-4"))
		m2 := gen.LanguageModel("model-b", "default", gen.SetModelName("claude-3"))
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{
					{Name: "model-a"},
					{Name: "model-b"},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(m1, m2).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		urls, names, err := r.resolveModels(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Multiple models in the same namespace → single shared proxy URL, two model names
		if len(urls) != 1 {
			t.Fatalf("expected 1 proxy URL (deduplicated), got %d: %v", len(urls), urls)
		}
		if len(names) != 2 {
			t.Fatalf("expected 2 names, got %d: %v", len(names), names)
		}
	})

	t.Run("missing_model", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{{Name: "nonexistent"}},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		_, _, err := r.resolveModels(context.Background(), agent)
		if err == nil {
			t.Error("expected error for missing model")
		}
	})
}

func TestLanguageAgentController_ResolveSidecarTools(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("sidecar_tool_added", func(t *testing.T) {
		tool := gen.LanguageTool("my-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolImage("my-tool:v1"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "my-sidecar"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		containers, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		if containers[0].Image != "my-tool:v1" {
			t.Errorf("expected image 'my-tool:v1', got %q", containers[0].Image)
		}
	})

	t.Run("service_mode_skipped", func(t *testing.T) {
		tool := gen.LanguageTool("svc-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "svc-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		containers, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 0 {
			t.Errorf("expected 0 containers for service-mode tool, got %d", len(containers))
		}
	})

	t.Run("default_resources_applied", func(t *testing.T) {
		tool := gen.LanguageTool("def-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "def-sidecar"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		containers, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		if containers[0].Resources.Requests.Cpu().IsZero() {
			t.Error("expected default CPU request to be set on sidecar container")
		}
	})

	t.Run("disabled_sidecar_tool_skipped", func(t *testing.T) {
		tool := gen.LanguageTool("disabled-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolImage("my-tool:v1"),
		)
		disabled := false
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "disabled-sidecar", Enabled: &disabled}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		containers, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 0 {
			t.Errorf("expected 0 containers for disabled sidecar tool, got %d", len(containers))
		}
	})
}

func TestLanguageAgentController_FetchPersona(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("no_refs", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		persona, err := r.fetchPersona(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if persona != nil {
			t.Error("expected nil persona when no refs")
		}
	})

	t.Run("single_ready_persona", func(t *testing.T) {
		p := gen.LanguagePersona("my-persona", "default")
		p.Status.Phase = events.PhaseStatusReady

		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "my-persona",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(p).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		persona, err := r.fetchPersona(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if persona == nil {
			t.Fatal("expected persona to be returned")
		}
	})

	t.Run("persona_not_ready", func(t *testing.T) {
		p := gen.LanguagePersona("pending-persona", "default")
		p.Status.Phase = "Pending"

		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "pending-persona",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(p).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		_, err := r.fetchPersona(context.Background(), agent)
		if err == nil {
			t.Error("expected error when persona not ready")
		}
	})

	t.Run("persona_not_found", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "missing",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		_, err := r.fetchPersona(context.Background(), agent)
		if err == nil {
			t.Error("expected error when persona not found")
		}
	})
}

// --- Group 5: CNI detection ---

func TestLanguageAgentController_DetectNetworkPolicySupport(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tests := []struct {
		name        string
		dsName      string
		wantSupport bool
		wantCNI     string
	}{
		{"cilium_detected", "cilium", true, "cilium"},
		{"calico_detected", "calico-node", true, "calico"},
		{"weave_detected", "weave-net", true, "weave-net"},
		{"antrea_detected", "antrea-agent", true, "antrea"},
		{"flannel_returns_false", "kube-flannel-ds", false, "flannel"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ds := &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.dsName,
					Namespace: "kube-system",
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ds).Build()
			r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

			supported, cni := r.detectNetworkPolicySupport(context.Background())
			if supported != tc.wantSupport {
				t.Errorf("supported: got %v, want %v", supported, tc.wantSupport)
			}
			if cni != tc.wantCNI {
				t.Errorf("CNI: got %q, want %q", cni, tc.wantCNI)
			}
		})
	}

	t.Run("no_cni_returns_false", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		supported, cni := r.detectNetworkPolicySupport(context.Background())
		if supported {
			t.Error("expected not supported when no CNI detected")
		}
		if cni != "unknown" {
			t.Errorf("expected 'unknown', got %q", cni)
		}
	})
}

func TestLanguageAgentController_AgentConfigVolume(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-vol-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "test instructions",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	expectedConfigMapName := GenerateConfigMapName(agent.Name, "agent")

	var foundVolume bool
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		if vol.Name == "agent-config" {
			foundVolume = true
			if vol.ConfigMap == nil {
				t.Fatal("agent-config volume has no ConfigMap source")
			}
			if vol.ConfigMap.Name != expectedConfigMapName {
				t.Errorf("agent-config volume points to %q, want %q", vol.ConfigMap.Name, expectedConfigMapName)
			}
			break
		}
	}
	if !foundVolume {
		t.Error("expected agent-config volume in deployment, not found")
	}

	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("no containers in deployment")
	}
	var foundMount bool
	for _, vm := range containers[0].VolumeMounts {
		if vm.Name == "agent-config" {
			foundMount = true
			if vm.MountPath != "/etc/agent" {
				t.Errorf("agent-config mount path is %q, want /etc/agent", vm.MountPath)
			}
			if !vm.ReadOnly {
				t.Error("agent-config volume mount should be read-only")
			}
			break
		}
	}
	if !foundMount {
		t.Error("expected agent-config volume mount in agent container, not found")
	}
}

func TestLanguageAgentController_AgentConfigMapKeys(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-keys-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "You are a helpful assistant.",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "agent"),
		Namespace: agent.Namespace,
	}, cm); err != nil {
		t.Fatalf("Expected agent ConfigMap to exist: %v", err)
	}

	if _, ok := cm.Data["config.yaml"]; !ok {
		t.Error("ConfigMap missing config.yaml key")
	}
	if _, ok := cm.Data["instructions.txt"]; !ok {
		t.Error("ConfigMap missing instructions.txt key")
	}

	if cm.Data["instructions.txt"] != agent.Spec.Instructions {
		t.Errorf("instructions.txt = %q, want %q", cm.Data["instructions.txt"], agent.Spec.Instructions)
	}

	configYAML := cm.Data["config.yaml"]
	if !strings.Contains(configYAML, agent.Name) {
		t.Errorf("config.yaml does not contain agent name %q: %s", agent.Name, configYAML)
	}
	if !strings.Contains(configYAML, agent.Namespace) {
		t.Errorf("config.yaml does not contain agent namespace %q: %s", agent.Namespace, configYAML)
	}
}

func TestLanguageAgentController_ServiceTypeAndAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		serviceType     corev1.ServiceType
		annotations     map[string]string
		wantServiceType corev1.ServiceType
	}{
		{
			name:            "defaults to ClusterIP when unset",
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
		{
			name:            "NodePort applied",
			serviceType:     corev1.ServiceTypeNodePort,
			wantServiceType: corev1.ServiceTypeNodePort,
		},
		{
			name:            "annotations propagated",
			annotations:     map[string]string{"example.com/ann": "value"},
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			mods := []gen.LanguageAgentModifier{}
			if tt.serviceType != "" {
				mods = append(mods, gen.SetAgentServiceType(tt.serviceType))
			}
			if tt.annotations != nil {
				mods = append(mods, gen.SetAgentServiceAnnotations(tt.annotations))
			}
			agent := gen.LanguageAgent("test-agent", "default", mods...)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), agent).
				WithStatusSubresource(agent).
				Build()

			reconciler := &LanguageAgentReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				Log:             logr.Discard(),
				Recorder:        &record.FakeRecorder{},
				RegistryManager: &mockRegistryManager{},
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			svc := &corev1.Service{}
			if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc); err != nil {
				t.Fatalf("expected service to exist: %v", err)
			}
			if svc.Spec.Type != tt.wantServiceType {
				t.Errorf("service type = %q, want %q", svc.Spec.Type, tt.wantServiceType)
			}
			for k, v := range tt.annotations {
				if svc.Annotations[k] != v {
					t.Errorf("annotation %q = %q, want %q", k, svc.Annotations[k], v)
				}
			}
		})
	}
}

func TestLanguageAgentController_SchedulingFields(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sched-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				NodeSelector: map[string]string{"kubernetes.io/arch": "amd64"},
				Tolerations: []corev1.Toleration{
					{Key: "gpu", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
					{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone", WhenUnsatisfiable: corev1.DoNotSchedule},
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{MatchExpressions: []corev1.NodeSelectorRequirement{
									{Key: "node-type", Operator: corev1.NodeSelectorOpIn, Values: []string{"gpu"}},
								}},
							},
						},
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{{Name: "my-registry-secret"}},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec

	if podSpec.NodeSelector["kubernetes.io/arch"] != "amd64" {
		t.Errorf("Expected NodeSelector kubernetes.io/arch=amd64, got %v", podSpec.NodeSelector)
	}
	if len(podSpec.Tolerations) == 0 || podSpec.Tolerations[0].Key != "gpu" {
		t.Errorf("Expected Toleration gpu, got %v", podSpec.Tolerations)
	}
	if len(podSpec.TopologySpreadConstraints) == 0 || podSpec.TopologySpreadConstraints[0].TopologyKey != "topology.kubernetes.io/zone" {
		t.Errorf("Expected TopologySpreadConstraint, got %v", podSpec.TopologySpreadConstraints)
	}
	if podSpec.Affinity == nil || podSpec.Affinity.NodeAffinity == nil {
		t.Error("Expected Affinity to be set")
	}
	if len(podSpec.ImagePullSecrets) == 0 || podSpec.ImagePullSecrets[0].Name != "my-registry-secret" {
		t.Errorf("Expected ImagePullSecret my-registry-secret, got %v", podSpec.ImagePullSecrets)
	}
}

func TestLanguageAgentController_PodLabelsAndAnnotations(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-labels-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				// "app.kubernetes.io/name" is also set by the operator; operator must take precedence.
				PodLabels: map[string]string{
					"cost-center":            "team-a",
					"app.kubernetes.io/name": "user-override-should-be-ignored",
				},
				PodAnnotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podMeta := deployment.Spec.Template.ObjectMeta

	// User label should be present
	if podMeta.Labels["cost-center"] != "team-a" {
		t.Errorf("Expected pod label cost-center=team-a, got %q", podMeta.Labels["cost-center"])
	}
	// Operator label must not be overridden by user; GetCommonLabels sets it to the agent name.
	if podMeta.Labels["app.kubernetes.io/name"] != agent.Name {
		t.Errorf("Operator label app.kubernetes.io/name should be %q, got %q", agent.Name, podMeta.Labels["app.kubernetes.io/name"])
	}
	// Selector labels must remain unchanged (operator labels only)
	selectorLabels := deployment.Spec.Selector.MatchLabels
	if _, hasUserLabel := selectorLabels["cost-center"]; hasUserLabel {
		t.Error("User pod label should not appear in selector MatchLabels")
	}
	// PodAnnotations
	if podMeta.Annotations["prometheus.io/scrape"] != "true" {
		t.Errorf("Expected pod annotation prometheus.io/scrape=true, got %q", podMeta.Annotations["prometheus.io/scrape"])
	}
	if podMeta.Annotations["prometheus.io/port"] != "8080" {
		t.Errorf("Expected pod annotation prometheus.io/port=8080, got %q", podMeta.Annotations["prometheus.io/port"])
	}
}

func TestLanguageAgentController_UserVolumesAndMounts(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-vols-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				Volumes: []corev1.Volume{
					{
						Name: "user-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: "my-secret"},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "user-secret", MountPath: "/etc/user-secret", ReadOnly: true},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	// Operator-managed volumes (tmp, agent-config) must still be present
	volumeNames := map[string]bool{}
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		volumeNames[v.Name] = true
	}
	if !volumeNames["tmp"] {
		t.Error("Expected operator-managed tmp volume to be present")
	}
	if !volumeNames["agent-config"] {
		t.Error("Expected operator-managed agent-config volume to be present")
	}
	if !volumeNames["user-secret"] {
		t.Error("Expected user-supplied user-secret volume to be appended")
	}

	// User mount must also be present
	mountPaths := map[string]string{}
	for _, m := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		mountPaths[m.Name] = m.MountPath
	}
	if mountPaths["user-secret"] != "/etc/user-secret" {
		t.Errorf("Expected user-secret mounted at /etc/user-secret, got %q", mountPaths["user-secret"])
	}
	// Operator mount (tmp) must still be present
	if mountPaths["tmp"] != "/tmp" {
		t.Errorf("Expected operator tmp mount at /tmp, got %q", mountPaths["tmp"])
	}
}

func TestLanguageAgentController_StartupProbe(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-startup-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				StartupProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					FailureThreshold:    30,
					PeriodSeconds:       10,
					InitialDelaySeconds: 5,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.StartupProbe == nil {
		t.Fatal("Expected StartupProbe to be set")
	}
	if container.StartupProbe.FailureThreshold != 30 {
		t.Errorf("Expected StartupProbe.FailureThreshold=30, got %d", container.StartupProbe.FailureThreshold)
	}
	if container.StartupProbe.PeriodSeconds != 10 {
		t.Errorf("Expected StartupProbe.PeriodSeconds=10, got %d", container.StartupProbe.PeriodSeconds)
	}
}

func TestLanguageAgentController_UserPodSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	customUID := int64(2000)
	customGID := int64(3000)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-custom-sec-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  &customUID,
					RunAsGroup: &customGID,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSec := deployment.Spec.Template.Spec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}
	if podSec.RunAsUser == nil || *podSec.RunAsUser != customUID {
		t.Errorf("Expected RunAsUser=%d (user-supplied), got %v", customUID, podSec.RunAsUser)
	}
	if podSec.RunAsGroup == nil || *podSec.RunAsGroup != customGID {
		t.Errorf("Expected RunAsGroup=%d (user-supplied), got %v", customGID, podSec.RunAsGroup)
	}
	// Container-level security context should still be the operator-managed one
	containerSec := deployment.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext should still be set by operator")
	}
	if containerSec.AllowPrivilegeEscalation == nil || *containerSec.AllowPrivilegeEscalation {
		t.Error("Container SecurityContext.AllowPrivilegeEscalation should be false")
	}
}
