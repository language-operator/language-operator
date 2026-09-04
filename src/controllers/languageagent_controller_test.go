package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
	if updatedAgent.Status.Phase != langopv1alpha1.ReasonPending {
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
	if readyCondition.Reason != langopv1alpha1.ReasonReconcileSuccess {
		t.Errorf("Expected reason 'ReconcileSuccess', got '%s'", readyCondition.Reason)
	}

	// ObservedGeneration must match the agent's generation after reconcile
	if updatedAgent.Status.ObservedGeneration != updatedAgent.Generation {
		t.Errorf("Expected ObservedGeneration=%d, got %d", updatedAgent.Generation, updatedAgent.Status.ObservedGeneration)
	}
}

func TestLanguageAgentController_RunStatusSync(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-status-agent",
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

	// Second reconcile: creates the Workflow.
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("second Reconcile failed: %v", err)
	}

	// Seed the Workflow status to stand in for the Argo workflow controller.
	wf := &wfv1.Workflow{}
	if err := fakeClient.Get(ctx, req.NamespacedName, wf); err != nil {
		t.Fatalf("Expected Workflow to exist after reconcile: %v", err)
	}
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.StartedAt = started
	// The fake client has no status subresource registered for Workflow, so the
	// whole object is written back.
	if err := fakeClient.Update(ctx, wf); err != nil {
		t.Fatalf("Failed to seed Workflow status: %v", err)
	}

	// Third reconcile: should pick the run state up onto the agent status.
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("third Reconcile failed: %v", err)
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updatedAgent); err != nil {
		t.Fatalf("Failed to fetch updated agent: %v", err)
	}

	if updatedAgent.Status.WorkflowTemplateName != agent.Name {
		t.Errorf("expected WorkflowTemplateName=%q, got %q", agent.Name, updatedAgent.Status.WorkflowTemplateName)
	}
	if updatedAgent.Status.ActiveWorkflowName != agent.Name {
		t.Errorf("expected ActiveWorkflowName=%q, got %q", agent.Name, updatedAgent.Status.ActiveWorkflowName)
	}
	if updatedAgent.Status.LastRunPhase != string(wfv1.WorkflowRunning) {
		t.Errorf("expected LastRunPhase=Running, got %q", updatedAgent.Status.LastRunPhase)
	}
	if updatedAgent.Status.LastRunStartedAt == nil {
		t.Error("expected LastRunStartedAt to be set")
	}
	if updatedAgent.Status.LastRunFinishedAt != nil {
		t.Errorf("expected LastRunFinishedAt to be unset for a running agent, got %v", updatedAgent.Status.LastRunFinishedAt)
	}
	if updatedAgent.Status.Phase != events.PhaseStatusRunning {
		t.Errorf("expected Phase=Running, got %q", updatedAgent.Status.Phase)
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

	// Pre-create per-agent RBAC resources that should be deleted on agent deletion.
	saName := GenerateServiceAccountName(agent.Name)
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "default"}}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "default"}}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "default"}}

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

	// Per-agent RBAC resources must be deleted on agent deletion.
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: "default"}, &corev1.ServiceAccount{}); !errors.IsNotFound(err) {
		t.Errorf("Expected ServiceAccount/%s to be deleted, got: %v", saName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: "default"}, &rbacv1.Role{}); !errors.IsNotFound(err) {
		t.Errorf("Expected Role/%s to be deleted, got: %v", saName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: "default"}, &rbacv1.RoleBinding{}); !errors.IsNotFound(err) {
		t.Errorf("Expected RoleBinding/%s to be deleted, got: %v", saName, err)
	}
}

// TestLanguageAgentController_DeletionCleansPerAgentRBACLeavesOtherAgentsIntact verifies that
// deleting an agent removes only its own per-agent RBAC and leaves the other agent's RBAC untouched.
func TestLanguageAgentController_DeletionCleansPerAgentRBACLeavesOtherAgentsIntact(t *testing.T) {
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

	deletingSAName := GenerateServiceAccountName(deletingAgent.Name)
	otherSAName := GenerateServiceAccountName(otherAgent.Name)

	// Pre-create per-agent RBAC for both agents.
	deletingSA := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: deletingSAName, Namespace: "default"}}
	deletingRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: deletingSAName, Namespace: "default"}}
	deletingRB := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: deletingSAName, Namespace: "default"}}
	otherSA := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: otherSAName, Namespace: "default"}}
	otherRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: otherSAName, Namespace: "default"}}
	otherRB := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: otherSAName, Namespace: "default"}}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deletingAgent, otherAgent, deletingSA, deletingRole, deletingRB, otherSA, otherRole, otherRB).
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

	// Deleting agent's per-agent RBAC must be cleaned up.
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: deletingSAName, Namespace: "default"}, &corev1.ServiceAccount{}); !errors.IsNotFound(err) {
		t.Errorf("Expected ServiceAccount/%s to be deleted, got: %v", deletingSAName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: deletingSAName, Namespace: "default"}, &rbacv1.Role{}); !errors.IsNotFound(err) {
		t.Errorf("Expected Role/%s to be deleted, got: %v", deletingSAName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: deletingSAName, Namespace: "default"}, &rbacv1.RoleBinding{}); !errors.IsNotFound(err) {
		t.Errorf("Expected RoleBinding/%s to be deleted, got: %v", deletingSAName, err)
	}

	// Other agent's per-agent RBAC must remain untouched.
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: otherSAName, Namespace: "default"}, &corev1.ServiceAccount{}); err != nil {
		t.Errorf("Expected ServiceAccount/%s to still exist, got: %v", otherSAName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: otherSAName, Namespace: "default"}, &rbacv1.Role{}); err != nil {
		t.Errorf("Expected Role/%s to still exist, got: %v", otherSAName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: otherSAName, Namespace: "default"}, &rbacv1.RoleBinding{}); err != nil {
		t.Errorf("Expected RoleBinding/%s to still exist, got: %v", otherSAName, err)
	}
}

// TestLanguageAgentController_DeletionCleansPerAgentRBACAlwaysDeletesOwn verifies that an
// agent's own per-agent RBAC is always cleaned up on deletion, even when other agents are
// also being deleted concurrently.
func TestLanguageAgentController_DeletionCleansPerAgentRBACAlwaysDeletesOwn(t *testing.T) {
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
	otherDeletingAgent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-agent-also-deleting",
			Namespace: "default",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
			Finalizers: []string{FinalizerName},
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Other agent also being deleted",
		},
	}

	saName := GenerateServiceAccountName(deletingAgent.Name)
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "default"}}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "default"}}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "default"}}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deletingAgent, otherDeletingAgent, sa, role, rb).
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

	// The deleting agent's per-agent RBAC must always be cleaned up.
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: "default"}, &corev1.ServiceAccount{}); !errors.IsNotFound(err) {
		t.Errorf("Expected ServiceAccount/%s to be deleted, got: %v", saName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: "default"}, &rbacv1.Role{}); !errors.IsNotFound(err) {
		t.Errorf("Expected Role/%s to be deleted, got: %v", saName, err)
	}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: "default"}, &rbacv1.RoleBinding{}); !errors.IsNotFound(err) {
		t.Errorf("Expected RoleBinding/%s to be deleted, got: %v", saName, err)
	}
}

func TestLanguageAgentController_UUIDAssignmentRaceCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid-agent",
			Namespace: "default",
			UID:       "aaaabbbb-cccc-dddd-eeee-ffffgggghhhh",
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
			UID:        "11112222-3333-4444-5555-666677778888",
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

	// Verify WorkflowTemplate created
	tmpl := &wfv1.WorkflowTemplate{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, tmpl); err != nil {
		t.Fatalf("Expected WorkflowTemplate to exist: %v", err)
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

	// Simulate the Workflow starting up.
	wf := &wfv1.Workflow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, wf); err != nil {
		t.Fatalf("Workflow not found: %v", err)
	}
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.StartedAt = metav1.Now()
	if err := fakeClient.Update(ctx, wf); err != nil {
		t.Fatalf("Failed to update Workflow status: %v", err)
	}

	// Third reconcile reads the updated Workflow status
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

	// Simulate the agent process dying for good: the Workflow ends in Failed.
	wf := &wfv1.Workflow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, wf); err != nil {
		t.Fatalf("Workflow not found: %v", err)
	}
	wf.Status.Phase = wfv1.WorkflowFailed
	wf.Status.StartedAt = metav1.Now()
	wf.Status.FinishedAt = metav1.Now()
	if err := fakeClient.Update(ctx, wf); err != nil {
		t.Fatalf("Failed to update Workflow status: %v", err)
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

func TestLanguageAgentController_PhaseErrorReportedAsFailed(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "phase-error-agent",
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

	// Argo reports infrastructure failures as Error rather than Failed; to someone
	// looking at an agent the distinction is not meaningful, so both read as Failed.
	wf := &wfv1.Workflow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, wf); err != nil {
		t.Fatalf("Workflow not found: %v", err)
	}
	wf.Status.Phase = wfv1.WorkflowError
	wf.Status.StartedAt = metav1.Now()
	if err := fakeClient.Update(ctx, wf); err != nil {
		t.Fatalf("Failed to update Workflow status: %v", err)
	}

	// Third reconcile reads the updated Workflow status
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
	if regCond.Reason != langopv1alpha1.ReasonRegistryNotAllowed {
		t.Errorf("Expected RegistryValidated reason %q, got %q", langopv1alpha1.ReasonRegistryNotAllowed, regCond.Reason)
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
			name: langopv1alpha1.ReasonConfigMapError,
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
			condReason:  langopv1alpha1.ReasonConfigMapError,
		},
		{
			name: langopv1alpha1.ReasonPVCError,
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
			condReason:  langopv1alpha1.ReasonPVCError,
		},
		{
			name: langopv1alpha1.ReasonServiceError,
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
			condReason:  langopv1alpha1.ReasonServiceError,
		},
		{
			name: langopv1alpha1.ReasonServiceAccountError,
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
			condReason:  langopv1alpha1.ReasonServiceAccountError,
		},
		{
			name: langopv1alpha1.ReasonWorkflowError,
			buildAgent: func() *langopv1alpha1.LanguageAgent {
				a := gen.LanguageAgent("wf-err-agent", "default")
				a.Finalizers = []string{FinalizerName}
				return a
			},
			failCreate:  func(obj client.Object) bool { _, ok := obj.(*wfv1.WorkflowTemplate); return ok },
			failErrMsg:  "injected workflow template error",
			expectError: true,
			condType:    langopv1alpha1.ConditionReady,
			condStatus:  metav1.ConditionFalse,
			condReason:  langopv1alpha1.ReasonWorkflowError,
		},
		{
			name: langopv1alpha1.ReasonNetworkPolicyError,
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
			condReason:       langopv1alpha1.ReasonNetworkPolicyError,
		},
		{
			name: langopv1alpha1.ReasonNetworkPolicyTimeout,
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
			condReason:       langopv1alpha1.ReasonNetworkPolicyTimeout,
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

func TestLanguageAgentController_DegradedPhase(t *testing.T) {
	// Verify that a running agent (Workflow phase Running) whose NetworkPolicy timed out
	// reports Degraded rather than Running.
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("degraded-agent", "default")
	agent.Finalizers = []string{FinalizerName}

	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*networkingv1.NetworkPolicy); ok {
					return fmt.Errorf("context deadline exceeded: timeout waiting for network policy")
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
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	// First reconcile: NP times out but reconcile continues; the Workflow is created.
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Seed the Workflow status to simulate the agent running.
	wf := &wfv1.Workflow{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, wf))
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.StartedAt = metav1.Now()
	require.NoError(t, fakeClient.Update(ctx, wf))

	// Second reconcile: NP still times out; the agent is running → Degraded phase.
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, updated))

	assert.Equal(t, events.PhaseStatusDegraded, updated.Status.Phase, "expected Degraded when agent is running but NetworkPolicy timed out")

	var npCond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionNetworkPolicyReady {
			npCond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, npCond, "expected ConditionNetworkPolicyReady to be set")
	assert.Equal(t, metav1.ConditionFalse, npCond.Status)
	assert.Equal(t, langopv1alpha1.ReasonNetworkPolicyTimeout, npCond.Reason)
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

	r := &LanguageAgentReconciler{Client: fakeClient,
		Scheme: scheme, Log: logr.Discard()}

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

func TestLanguageAgentController_ManagedResources(t *testing.T) {
	reconcileAgent := func(t *testing.T, agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster, networkIsolation bool) *langopv1alpha1.LanguageAgent {
		t.Helper()
		scheme := testutil.SetupTestScheme(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, agent).
			WithStatusSubresource(agent).
			Build()
		recorder := record.NewFakeRecorder(10)
		reconciler := &LanguageAgentReconciler{
			Client:                  fakeClient,
			Scheme:                  scheme,
			Log:                     logr.Discard(),
			Recorder:                recorder,
			EventManager:            events.NewEventManager(recorder),
			RegistryManager:         &mockRegistryManager{},
			NetworkIsolationEnabled: networkIsolation,
		}
		ctx := context.Background()
		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
		// First reconcile adds finalizer; second creates resources and writes status.
		_, err := reconciler.Reconcile(ctx, req)
		require.NoError(t, err)
		_, err = reconciler.Reconcile(ctx, req)
		require.NoError(t, err)
		updated := &langopv1alpha1.LanguageAgent{}
		require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, updated))
		return updated
	}

	hasMR := func(resources []langopv1alpha1.ManagedResource, kind, name string) bool {
		for _, r := range resources {
			if r.Kind == kind && r.Name == name {
				return true
			}
		}
		return false
	}

	t.Run("baseline_no_workspace_no_domain_no_creds", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "base-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:     "ghcr.io/language-operator/agent:latest",
				Workspace: &langopv1alpha1.WorkspaceSpec{Enabled: func() *bool { b := false; return &b }()},
			},
		}
		cluster := gen.ReadyCluster("default")
		updated := reconcileAgent(t, agent, cluster, false)
		mr := updated.Status.ManagedResources

		assert.NotEmpty(t, mr)
		assert.True(t, hasMR(mr, "WorkflowTemplate", "base-agent"), "WorkflowTemplate must be present")
		assert.True(t, hasMR(mr, "Workflow", "base-agent"), "Workflow must be present for a service agent")
		assert.True(t, hasMR(mr, "Service", "base-agent"), "Service must be present")
		assert.True(t, hasMR(mr, "ConfigMap", GenerateConfigMapName("base-agent", "agent")), "ConfigMap must be present")
		assert.True(t, hasMR(mr, "ServiceAccount", GenerateServiceAccountName("base-agent")), "ServiceAccount must be present")
		assert.True(t, hasMR(mr, "Role", GenerateServiceAccountName("base-agent")), "Role must be present")
		assert.True(t, hasMR(mr, "RoleBinding", GenerateServiceAccountName("base-agent")), "RoleBinding must be present")

		assert.False(t, hasMR(mr, "PersistentVolumeClaim", GeneratePVCName("base-agent")), "PVC must not be present when workspace disabled")
		assert.False(t, hasMR(mr, "Secret", "base-agent-runtime"), "Secret must not be present without creds")
		assert.False(t, hasMR(mr, "Ingress", "base-agent"), "Ingress must not be present without domain")
		assert.False(t, hasMR(mr, "NetworkPolicy", "base-agent"), "NetworkPolicy must not be present without isolation")
	})

	t.Run("workspace_enabled", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "ws-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:     "ghcr.io/language-operator/agent:latest",
				Workspace: &langopv1alpha1.WorkspaceSpec{Enabled: func() *bool { b := true; return &b }()},
			},
		}
		updated := reconcileAgent(t, agent, gen.ReadyCluster("default"), false)
		assert.True(t, hasMR(updated.Status.ManagedResources, "PersistentVolumeClaim", GeneratePVCName("ws-agent")))
	})

	t.Run("workspace_nil_defaults_to_enabled", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "ws-nil-agent", Namespace: "default"},
			Spec:       langopv1alpha1.LanguageAgentSpec{Image: "ghcr.io/language-operator/agent:latest"},
		}
		updated := reconcileAgent(t, agent, gen.ReadyCluster("default"), false)
		assert.True(t, hasMR(updated.Status.ManagedResources, "PersistentVolumeClaim", GeneratePVCName("ws-nil-agent")))
	})

	t.Run("cluster_domain_adds_ingress", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "ing-agent", Namespace: "test-cluster"},
			Spec:       langopv1alpha1.LanguageAgentSpec{Image: "ghcr.io/language-operator/agent:latest"},
		}
		cluster := gen.ReadyCluster("test-cluster", gen.SetClusterDomain("agents.example.com"))
		updated := reconcileAgent(t, agent, cluster, false)
		assert.True(t, hasMR(updated.Status.ManagedResources, "Ingress", "ing-agent"))
	})

	t.Run("network_isolation_adds_networkpolicy", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "np-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:     "ghcr.io/language-operator/agent:latest",
				Workspace: &langopv1alpha1.WorkspaceSpec{Enabled: func() *bool { b := false; return &b }()},
			},
		}
		updated := reconcileAgent(t, agent, gen.ReadyCluster("default"), true)
		assert.True(t, hasMR(updated.Status.ManagedResources, "NetworkPolicy", "np-agent"))
	})

	t.Run("custom_service_account_omits_rbac", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "custom-sa-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
				Deployment: langopv1alpha1.DeploymentSpec{
					ServiceAccountName: "my-sa",
				},
			},
		}
		updated := reconcileAgent(t, agent, gen.ReadyCluster("default"), false)
		mr := updated.Status.ManagedResources
		assert.False(t, hasMR(mr, "ServiceAccount", "language-agent"))
		assert.False(t, hasMR(mr, "Role", "language-agent"))
		assert.False(t, hasMR(mr, "RoleBinding", "language-agent"))
	})

	t.Run("scheduled_task_adds_cronworkflow", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "cron-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
				Execution: langopv1alpha1.ExecutionSpec{
					Mode:     langopv1alpha1.ExecutionModeTask,
					Schedule: "*/5 * * * *",
				},
			},
		}
		mr := reconcileAgent(t, agent, gen.ReadyCluster("default"), false).Status.ManagedResources
		assert.True(t, hasMR(mr, "CronWorkflow", "cron-agent"))
		assert.True(t, hasMR(mr, "WorkflowTemplate", "cron-agent"))
		// A task agent is not addressable, so it gets neither a Workflow nor a Service.
		assert.False(t, hasMR(mr, "Workflow", "cron-agent"))
		assert.False(t, hasMR(mr, "Service", "cron-agent"))
	})

	t.Run("unscheduled_task_is_template_only", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "manual-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:     "ghcr.io/language-operator/agent:latest",
				Execution: langopv1alpha1.ExecutionSpec{Mode: langopv1alpha1.ExecutionModeTask},
			},
		}
		mr := reconcileAgent(t, agent, gen.ReadyCluster("default"), false).Status.ManagedResources
		assert.True(t, hasMR(mr, "WorkflowTemplate", "manual-agent"))
		assert.False(t, hasMR(mr, "CronWorkflow", "manual-agent"))
		assert.False(t, hasMR(mr, "Workflow", "manual-agent"))
	})

}
