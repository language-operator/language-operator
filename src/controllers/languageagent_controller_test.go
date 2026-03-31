package controllers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
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
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	// First reconcile adds finalizer and requeues
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile creates resources and updates status
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
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

	// After reconcile, deployment has 0 ready replicas in fake client → Pending
	if updatedAgent.Status.Phase != "Pending" {
		t.Errorf("Expected phase 'Pending', got '%s'", updatedAgent.Status.Phase)
	}

	// Verify Ready condition
	var readyCondition *metav1.Condition
	for i := range updatedAgent.Status.Conditions {
		if updatedAgent.Status.Conditions[i].Type == langopv1alpha1.ConditionReady {
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

	// ObservedGeneration must match the agent's generation after reconcile
	if updatedAgent.Status.ObservedGeneration != updatedAgent.Generation {
		t.Errorf("Expected ObservedGeneration=%d, got %d", updatedAgent.Generation, updatedAgent.Status.ObservedGeneration)
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

// TestLanguageAgentController_DeletionRemovesFinalizer verifies that reconciling an agent
// with a DeletionTimestamp removes the finalizer and cleans up shared RBAC resources.
func TestLanguageAgentController_DeletionRemovesFinalizer(t *testing.T) {
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

	// Pre-create shared RBAC resources that should be deleted when the last agent goes away.
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: "default"}}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: "default"}}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: "default"}}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sa, role, rb).
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

	// Finalizer must be removed so Kubernetes can complete deletion.
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)

	if errors.IsNotFound(err) {
		// Agent was fully deleted — acceptable.
	} else if err != nil {
		t.Fatalf("Unexpected error getting updated agent: %v", err)
	} else {
		for _, finalizer := range updatedAgent.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after reconcile with DeletionTimestamp")
			}
		}
	}

	// Shared RBAC resources must be deleted when the last agent is gone.
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: "default"}, &corev1.ServiceAccount{}); !errors.IsNotFound(err) {
		t.Errorf("Expected ServiceAccount/language-agent to be deleted, got: %v", err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: "default"}, &rbacv1.Role{}); !errors.IsNotFound(err) {
		t.Errorf("Expected Role/language-agent to be deleted, got: %v", err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: "default"}, &rbacv1.RoleBinding{}); !errors.IsNotFound(err) {
		t.Errorf("Expected RoleBinding/language-agent to be deleted, got: %v", err)
	}
}

// TestLanguageAgentController_DeletionKeepsRBACWhenOtherAgentsExist verifies that shared
// RBAC resources are NOT deleted when another LanguageAgent still exists in the namespace.
func TestLanguageAgentController_DeletionKeepsRBACWhenOtherAgentsExist(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	deletingAgent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-being-deleted",
			Namespace: "default",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
			Finalizers: []string{FinalizerName},
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Agent being deleted",
		},
	}
	otherAgent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Other agent still running",
		},
	}

	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: "default"}}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: "default"}}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: "default"}}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deletingAgent, otherAgent, sa, role, rb).
		WithStatusSubresource(deletingAgent).
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
			Name:      deletingAgent.Name,
			Namespace: deletingAgent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Shared RBAC resources must remain because other-agent still exists.
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: "default"}, &corev1.ServiceAccount{}); err != nil {
		t.Errorf("Expected ServiceAccount/language-agent to still exist, got: %v", err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: "default"}, &rbacv1.Role{}); err != nil {
		t.Errorf("Expected Role/language-agent to still exist, got: %v", err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: "default"}, &rbacv1.RoleBinding{}); err != nil {
		t.Errorf("Expected RoleBinding/language-agent to still exist, got: %v", err)
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

	cluster := gen.ReadyCluster("default")
	cluster.UID = "test-cluster-uid-1234"

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
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
	// AGENT_UUID is empty until status is populated; key must still be present
	if _, ok := envMap["AGENT_UUID"]; !ok {
		t.Error("Expected AGENT_UUID env var to be present")
	}
	if envMap["AGENT_CLUSTER_NAME"] != "default" {
		t.Errorf("Expected AGENT_CLUSTER_NAME=default, got %s", envMap["AGENT_CLUSTER_NAME"])
	}
	if envMap["AGENT_CLUSTER_UUID"] != "test-cluster-uid-1234" {
		t.Errorf("Expected AGENT_CLUSTER_UUID=test-cluster-uid-1234, got %s", envMap["AGENT_CLUSTER_UUID"])
	}
	// AGENT_MODE must not be set when ExecutionMode is empty
	if _, ok := envMap["AGENT_MODE"]; ok {
		t.Errorf("Expected AGENT_MODE to be absent when ExecutionMode is empty, but got %s", envMap["AGENT_MODE"])
	}
}

func TestLanguageAgentController_AgentModeInjection(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mode-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	cluster := gen.ReadyCluster("default")
	cluster.UID = "cluster-uid-mode-test"

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
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

	envMap := make(map[string]string)
	for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		envMap[e.Name] = e.Value
	}

	if envMap["AGENT_MODE"] != "autonomous" {
		t.Errorf("Expected AGENT_MODE=autonomous, got %q", envMap["AGENT_MODE"])
	}
	if envMap["AGENT_CLUSTER_UUID"] != "cluster-uid-mode-test" {
		t.Errorf("Expected AGENT_CLUSTER_UUID=cluster-uid-mode-test, got %s", envMap["AGENT_CLUSTER_UUID"])
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

	// Verify namespace-scoped Role created
	role := &rbacv1.Role{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: agent.Namespace}, role); err != nil {
		t.Fatalf("Expected Role 'language-agent' to exist in namespace %s: %v", agent.Namespace, err)
	}
	if len(role.Rules) == 0 {
		t.Errorf("Expected Role to have at least one rule")
	}

	// Verify namespace-scoped RoleBinding created and points to the Role (not the operator ClusterRole)
	rb := &rbacv1.RoleBinding{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: agent.Namespace}, rb); err != nil {
		t.Fatalf("Expected RoleBinding 'language-agent' to exist in namespace %s: %v", agent.Namespace, err)
	}
	if rb.RoleRef.Kind != "Role" {
		t.Errorf("Expected RoleBinding RoleRef.Kind 'Role', got %q", rb.RoleRef.Kind)
	}
	if rb.RoleRef.Name != "language-agent" {
		t.Errorf("Expected RoleBinding RoleRef.Name 'language-agent', got %q", rb.RoleRef.Name)
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

func TestLanguageAgentController_NetworkPolicy_FromRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("from-rule-agent", "default",
		gen.SetAgentNetworkPolicies([]langopv1alpha1.NetworkRule{
			{
				From: &langopv1alpha1.NetworkPeer{
					Group: "monitoring",
				},
				Ports: []langopv1alpha1.NetworkPort{
					{Port: 9090},
				},
			},
		}),
	)

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

	np := &networkingv1.NetworkPolicy{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np); err != nil {
		t.Fatalf("NetworkPolicy not found: %v", err)
	}

	t.Run("from_rule_appended_to_ingress", func(t *testing.T) {
		// Hardcoded rules: trigger, dashboard, agent-to-agent = 3; user From rule = 1; total >= 4
		if len(np.Spec.Ingress) < 4 {
			t.Errorf("expected at least 4 ingress rules (3 default + 1 from spec), got %d", len(np.Spec.Ingress))
		}
		// Last rule should be the user-defined one
		last := np.Spec.Ingress[len(np.Spec.Ingress)-1]
		if len(last.From) == 0 {
			t.Fatal("expected From peers in user-defined ingress rule")
		}
		peer := last.From[0]
		if peer.PodSelector == nil {
			t.Fatal("expected PodSelector for Group-based From rule")
		}
		if peer.PodSelector.MatchLabels[LabelKeyLangopGroup] != "monitoring" {
			t.Errorf("expected langop.io/group=monitoring, got %v", peer.PodSelector.MatchLabels)
		}
		if len(last.Ports) == 0 || last.Ports[0].Port.IntVal != 9090 {
			t.Error("expected port 9090 in user-defined ingress rule")
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
		// All models in a namespace share the single cluster gateway
		expectedURL := "http://gateway.default.svc.cluster.local:8000"
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
		// Multiple models in the same namespace → single shared gateway URL, two model names
		if len(urls) != 1 {
			t.Fatalf("expected 1 gateway URL (deduplicated), got %d: %v", len(urls), urls)
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
		p.Status.Phase = events.PhaseStatusPending

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
	if _, ok := cm.Data["instructions.txt"]; ok {
		t.Error("ConfigMap must not contain instructions.txt key")
	}

	configYAML := cm.Data["config.yaml"]
	if !strings.Contains(configYAML, agent.Name) {
		t.Errorf("config.yaml does not contain agent name %q: %s", agent.Name, configYAML)
	}
	if !strings.Contains(configYAML, agent.Namespace) {
		t.Errorf("config.yaml does not contain agent namespace %q: %s", agent.Namespace, configYAML)
	}
	if !strings.Contains(configYAML, agent.Spec.Instructions) {
		t.Errorf("config.yaml does not contain instructions %q: %s", agent.Spec.Instructions, configYAML)
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

func TestLanguageAgentController_PhasePending(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "phase-pending-agent",
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

	// First reconcile adds finalizer, no status written yet
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile creates Deployment (with 0 ready replicas) → Pending
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent); err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}
	if updatedAgent.Status.Phase != events.PhaseStatusPending {
		t.Errorf("Expected phase %q, got %q", events.PhaseStatusPending, updatedAgent.Status.Phase)
	}
}

func TestLanguageAgentController_PhaseRunning(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "phase-running-agent",
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

	// Two reconciles to get past finalizer and create resources
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Simulate the Deployment becoming available: update with ReadyReplicas=1
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deploy); err != nil {
		t.Fatalf("Deployment not found: %v", err)
	}
	deploy.Status.Replicas = 1
	deploy.Status.ReadyReplicas = 1
	if err := fakeClient.Status().Update(ctx, deploy); err != nil {
		t.Fatalf("Failed to update deployment status: %v", err)
	}

	// Third reconcile reads the updated deployment status
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Third reconcile failed: %v", err)
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent); err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}
	if updatedAgent.Status.Phase != events.PhaseStatusRunning {
		t.Errorf("Expected phase %q, got %q", events.PhaseStatusRunning, updatedAgent.Status.Phase)
	}
}

func TestLanguageAgentController_PhaseFailed(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "phase-failed-agent",
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

	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Simulate pods crashing: Replicas>0 but ReadyReplicas=0 and DeploymentAvailable=False
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deploy); err != nil {
		t.Fatalf("Deployment not found: %v", err)
	}
	deploy.Status.Replicas = 1
	deploy.Status.ReadyReplicas = 0
	deploy.Status.Conditions = []appsv1.DeploymentCondition{
		{
			Type:   appsv1.DeploymentAvailable,
			Status: corev1.ConditionFalse,
			Reason: "MinimumReplicasUnavailable",
		},
	}
	if err := fakeClient.Status().Update(ctx, deploy); err != nil {
		t.Fatalf("Failed to update deployment status: %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Third reconcile failed: %v", err)
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent); err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}
	if updatedAgent.Status.Phase != events.PhaseStatusFailed {
		t.Errorf("Expected phase %q, got %q", events.PhaseStatusFailed, updatedAgent.Status.Phase)
	}
}

func TestLanguageAgentController_PhaseFailedOnEarlyExit(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Image uses ghcr.io but registry whitelist only allows docker.io → validation fails
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "early-exit-agent",
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
		// Whitelist excludes ghcr.io → registry validation will fail
		RegistryManager: &mockRegistryManager{registries: []string{"docker.io"}},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	// Reconcile: finalizer added then registry validation fires immediately → early exit with error
	_, err := reconciler.Reconcile(ctx, req)
	if err == nil {
		t.Fatal("Expected reconcile error from registry validation, got nil")
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent); err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}
	if updatedAgent.Status.Phase != events.PhaseStatusFailed {
		t.Errorf("Expected phase %q after early-exit error, got %q", events.PhaseStatusFailed, updatedAgent.Status.Phase)
	}

	var regCond *metav1.Condition
	for i := range updatedAgent.Status.Conditions {
		if updatedAgent.Status.Conditions[i].Type == langopv1alpha1.ConditionRegistryValidated {
			regCond = &updatedAgent.Status.Conditions[i]
			break
		}
	}
	if regCond == nil {
		t.Fatal("Expected RegistryValidated condition to be set")
	}
	if regCond.Status != metav1.ConditionFalse {
		t.Errorf("Expected RegistryValidated status %q, got %q", metav1.ConditionFalse, regCond.Status)
	}
	if regCond.Reason != "RegistryNotAllowed" {
		t.Errorf("Expected RegistryValidated reason %q, got %q", "RegistryNotAllowed", regCond.Reason)
	}
}

func TestLanguageAgentController_ObservedGenerationSetOnError(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Image uses ghcr.io but registry whitelist only allows docker.io → validation fails
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "gen-error-agent",
			Namespace:  "default",
			Generation: 3,
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
		RegistryManager: &mockRegistryManager{registries: []string{"docker.io"}},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	if err == nil {
		t.Fatal("Expected reconcile error from registry validation, got nil")
	}

	updated := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated); err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}
	if updated.Status.ObservedGeneration != agent.Generation {
		t.Errorf("Expected ObservedGeneration %d on error path, got %d", agent.Generation, updated.Status.ObservedGeneration)
	}
	if updated.Status.Phase != events.PhaseStatusFailed {
		t.Errorf("Expected phase %q, got %q", events.PhaseStatusFailed, updated.Status.Phase)
	}
}

// parseAgentConfigMap reconciles the agent once and returns the parsed config.yaml from the agent ConfigMap.
func parseAgentConfigMap(t *testing.T, scheme *runtime.Scheme, objects ...client.Object) agentConfigYAML {
	t.Helper()

	// The agent must be the last object passed so we can extract its name/namespace.
	agentObj := objects[len(objects)-1]
	agent := agentObj.(*langopv1alpha1.LanguageAgent)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
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
	require.NoError(t, err, "Reconcile failed")

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "agent"),
		Namespace: agent.Namespace,
	}, cm), "agent ConfigMap not found")

	var cfg agentConfigYAML
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &cfg), "failed to parse config.yaml")

	return cfg
}

func TestLanguageAgentController_ConfigMapContent(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("model_section", func(t *testing.T) {
		model := gen.LanguageModel("claude-model", "default",
			gen.SetModelProvider("anthropic"),
			gen.SetModelName("claude-sonnet-4-5"),
		)
		agent := gen.LanguageAgent("model-agent", "default",
			gen.SetAgentInstructions("do the thing"),
		)
		agent.Spec.Models = []langopv1alpha1.ModelReference{
			{Name: "claude-model", Role: "primary"},
		}

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), model, agent)

		assert.Equal(t, "do the thing", cfg.Instructions)
		require.Contains(t, cfg.Models, "claude-model", "config.yaml missing model entry")
		m := cfg.Models["claude-model"]
		assert.Equal(t, "http://gateway.default.svc.cluster.local:8000", m.Endpoint)
		assert.Equal(t, "anthropic", m.Provider)
		assert.Equal(t, "claude-sonnet-4-5", m.Model)
		assert.Equal(t, "primary", m.Role)
	})

	t.Run("tool_service_mode", func(t *testing.T) {
		tool := gen.LanguageTool("search-tool", "default") // service mode by default, port 0 → 8080
		agent := gen.LanguageAgent("tool-agent", "default")
		agent.Spec.Tools = []langopv1alpha1.ToolReference{{Name: "search-tool"}}

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), tool, agent)

		require.Contains(t, cfg.Tools, "search-tool", "config.yaml missing tool entry")
		tool_cfg := cfg.Tools["search-tool"]
		assert.Equal(t, "http://search-tool.default.svc.cluster.local:8080", tool_cfg.Endpoint)
		assert.Equal(t, "mcp", tool_cfg.Protocol)
	})

	t.Run("tool_sidecar_mode", func(t *testing.T) {
		tool := gen.LanguageTool("sidecar-tool", "default",
			gen.SetToolDeploymentMode("sidecar"),
		)
		agent := gen.LanguageAgent("sidecar-agent", "default")
		agent.Spec.Tools = []langopv1alpha1.ToolReference{{Name: "sidecar-tool"}}

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), tool, agent)

		require.Contains(t, cfg.Tools, "sidecar-tool", "config.yaml missing sidecar tool entry")
		assert.Equal(t, "http://localhost:8080", cfg.Tools["sidecar-tool"].Endpoint)
		assert.Equal(t, "mcp", cfg.Tools["sidecar-tool"].Protocol)
	})

	t.Run("persona_section", func(t *testing.T) {
		persona := gen.LanguagePersona("my-persona", "default",
			gen.SetPersonaTone("professional"),
			gen.SetPersonaPersonality("concise and precise"),
		)
		persona.Status.Phase = events.PhaseStatusReady

		agent := gen.LanguageAgent("persona-agent", "default")
		agent.Spec.Persona = "my-persona"

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), persona, agent)

		require.Len(t, cfg.Personas, 1, "config.yaml should contain exactly one persona")
		p := cfg.Personas[0]
		assert.Equal(t, "my-persona", p.Name)
		assert.Equal(t, "professional", p.Tone)
		assert.Equal(t, "concise and precise", p.Personality)
	})
}

func TestLanguageAgentController_WorkspaceStorageClass(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sc-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceStorageClass("fast-ssd"),
	)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pvc := &corev1.PersistentVolumeClaim{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name + "-workspace", Namespace: agent.Namespace}, pvc))
	require.NotNil(t, pvc.Spec.StorageClassName)
	assert.Equal(t, "fast-ssd", *pvc.Spec.StorageClassName)
}

func TestLanguageAgentController_WorkspaceMountPath(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("mp-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceMountPath("/data"),
	)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	deployment := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment))
	require.NotEmpty(t, deployment.Spec.Template.Spec.Containers)
	var found bool
	for _, vm := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		if vm.Name == "workspace" {
			assert.Equal(t, "/data", vm.MountPath)
			found = true
			break
		}
	}
	assert.True(t, found, "expected workspace volume mount in container")
}

func TestLanguageAgentController_WorkspaceAccessMode(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("am-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceAccessMode(corev1.ReadWriteMany),
	)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pvc := &corev1.PersistentVolumeClaim{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name + "-workspace", Namespace: agent.Namespace}, pvc))
	require.NotEmpty(t, pvc.Spec.AccessModes)
	assert.Equal(t, corev1.ReadWriteMany, pvc.Spec.AccessModes[0])
}

func TestLanguageAgentController_ErrorPathConditions(t *testing.T) {
	type errorPathCase struct {
		name             string
		buildAgent       func() *langopv1alpha1.LanguageAgent
		failCreate       func(obj client.Object) bool
		failErrMsg       string
		networkIsolation bool
		expectError      bool
		condType         string
		condStatus       metav1.ConditionStatus
		condReason       string
	}

	cases := []errorPathCase{
		{
			name: "ConfigMapError",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("cm-err-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:  func(obj client.Object) bool { _, ok := obj.(*corev1.ConfigMap); return ok },
			failErrMsg:  "injected configmap error",
			expectError: true,
			condType:    langopv1alpha1.ConditionReady,
			condStatus:  metav1.ConditionFalse,
			condReason:  "ConfigMapError",
		},
		{
			name: "PVCError",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("pvc-err-agent", "default", gen.SetAgentWorkspace("5Gi"))
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:  func(obj client.Object) bool { _, ok := obj.(*corev1.PersistentVolumeClaim); return ok },
			failErrMsg:  "injected pvc error",
			expectError: true,
			condType:    langopv1alpha1.ConditionReady,
			condStatus:  metav1.ConditionFalse,
			condReason:  "PVCError",
		},
		{
			name: "ServiceError",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("svc-err-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:  func(obj client.Object) bool { _, ok := obj.(*corev1.Service); return ok },
			failErrMsg:  "injected service error",
			expectError: true,
			condType:    langopv1alpha1.ConditionReady,
			condStatus:  metav1.ConditionFalse,
			condReason:  "ServiceError",
		},
		{
			name: "ServiceAccountError",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("sa-err-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:  func(obj client.Object) bool { _, ok := obj.(*corev1.ServiceAccount); return ok },
			failErrMsg:  "injected serviceaccount error",
			expectError: true,
			condType:    langopv1alpha1.ConditionReady,
			condStatus:  metav1.ConditionFalse,
			condReason:  "ServiceAccountError",
		},
		{
			name: "DeploymentError",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("dep-err-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:  func(obj client.Object) bool { _, ok := obj.(*appsv1.Deployment); return ok },
			failErrMsg:  "injected deployment error",
			expectError: true,
			condType:    langopv1alpha1.ConditionReady,
			condStatus:  metav1.ConditionFalse,
			condReason:  "DeploymentError",
		},
		{
			name: "NetworkPolicyError",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("np-err-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:       func(obj client.Object) bool { _, ok := obj.(*networkingv1.NetworkPolicy); return ok },
			failErrMsg:       "injected networkpolicy error",
			networkIsolation: true,
			expectError:      true,
			condType:         langopv1alpha1.ConditionReady,
			condStatus:       metav1.ConditionFalse,
			condReason:       "NetworkPolicyError",
		},
		{
			name: "NetworkPolicyTimeout",
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("np-timeout-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:       func(obj client.Object) bool { _, ok := obj.(*networkingv1.NetworkPolicy); return ok },
			failErrMsg:       "context deadline exceeded: timeout waiting for network policy",
			networkIsolation: true,
			expectError:      false, // degraded mode — reconcile continues without error
			condType:         langopv1alpha1.ConditionNetworkPolicyReady,
			condStatus:       metav1.ConditionFalse,
			condReason:       "NetworkPolicyTimeout",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			agent := tc.buildAgent()
			recorder := record.NewFakeRecorder(10)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), agent).
				WithStatusSubresource(agent).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if tc.failCreate(obj) {
							return fmt.Errorf("%s", tc.failErrMsg)
						}
						return c.Create(ctx, obj, opts...)
					},
				}).Build()

			reconciler := &LanguageAgentReconciler{
				Client:                  fakeClient,
				Scheme:                  scheme,
				Log:                     logr.Discard(),
				Recorder:                recorder,
				EventManager:            events.NewEventManager(recorder),
				RegistryManager:         &mockRegistryManager{},
				NetworkIsolationEnabled: tc.networkIsolation,
			}

			ctx := context.Background()
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

			// Agent already has finalizer — single reconcile hits resource creation.
			_, err := reconciler.Reconcile(ctx, req)
			if tc.expectError {
				require.Error(t, err, "expected reconcile to return error")
			} else {
				require.NoError(t, err, "expected reconcile to succeed in degraded mode")
			}

			updated := &langopv1alpha1.LanguageAgent{}
			require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

			if tc.expectError {
				assert.Equal(t, events.PhaseStatusFailed, updated.Status.Phase)
			}

			var cond *metav1.Condition
			for i := range updated.Status.Conditions {
				if updated.Status.Conditions[i].Type == tc.condType {
					cond = &updated.Status.Conditions[i]
					break
				}
			}
			require.NotNil(t, cond, "expected condition %q to be set", tc.condType)
			assert.Equal(t, tc.condStatus, cond.Status)
			assert.Equal(t, tc.condReason, cond.Reason)
		})
	}
}

func TestLanguageAgentController_EnqueueAgentsInNamespace(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent1 := gen.LanguageAgent("agent-1", "ns-a")
	agent2 := gen.LanguageAgent("agent-2", "ns-a")
	agentOther := gen.LanguageAgent("agent-other", "ns-b")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent1, agent2, agentOther).
		Build()

	r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	// A LanguageTool in ns-a should enqueue agent-1 and agent-2 only.
	tool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{Name: "my-tool", Namespace: "ns-a"},
	}

	reqs := r.enqueueAgentsInNamespace()(context.Background(), tool)
	require.Len(t, reqs, 2)
	names := make([]string, len(reqs))
	for i, req := range reqs {
		assert.Equal(t, "ns-a", req.Namespace)
		names[i] = req.Name
	}
	assert.ElementsMatch(t, []string{"agent-1", "agent-2"}, names)
}

func TestLanguageAgentController_ConditionNetworkPolicyEnforced_Supported(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Seed a Cilium DaemonSet so detectNetworkPolicySupport returns (true, "cilium").
	ciliumDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cilium",
			Namespace: "kube-system",
		},
	}

	agent := gen.LanguageAgent("np-enforced-agent", "default")
	agent.Finalizers = []string{FinalizerName}

	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent, ciliumDS).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                recorder,
		EventManager:            events.NewEventManager(recorder),
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
		NetworkPolicyTimeout:    30 * time.Second,
		NetworkPolicyRetries:    3,
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionNetworkPolicyEnforced {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "expected ConditionNetworkPolicyEnforced to be set")
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, "Enforced", cond.Reason)
	assert.Contains(t, cond.Message, "cilium")
}

func TestLanguageAgentController_ConditionNetworkPolicyEnforced_NotSupported(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// No CNI DaemonSet present — detectNetworkPolicySupport returns (false, "unknown").
	agent := gen.LanguageAgent("np-not-enforced-agent", "default")
	agent.Finalizers = []string{FinalizerName}

	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                recorder,
		EventManager:            events.NewEventManager(recorder),
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
		NetworkPolicyTimeout:    30 * time.Second,
		NetworkPolicyRetries:    3,
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionNetworkPolicyEnforced {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "expected ConditionNetworkPolicyEnforced to be set")
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, "CNINotSupported", cond.Reason)
}
