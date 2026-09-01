package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

func TestLanguageAgentController_WorkspaceStorageClass_DefaultFromReconciler(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sc-default-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
	)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).Build()
	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                &record.FakeRecorder{},
		RegistryManager:         &mockRegistryManager{},
		DefaultStorageClassName: "local-path",
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pvc := &corev1.PersistentVolumeClaim{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name + "-workspace", Namespace: agent.Namespace}, pvc))
	require.NotNil(t, pvc.Spec.StorageClassName)
	assert.Equal(t, "local-path", *pvc.Spec.StorageClassName)
}

func TestLanguageAgentController_WorkspaceStorageClass_PerAgentOverridesDefault(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sc-override-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceStorageClass("fast-ssd"),
	)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).Build()
	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                &record.FakeRecorder{},
		RegistryManager:         &mockRegistryManager{},
		DefaultStorageClassName: "local-path",
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

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
	require.NotEmpty(t, podSpec.Containers)
	var found bool
	for _, vm := range podSpec.Containers[0].VolumeMounts {
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

func TestLanguageAgentController_WorkspaceRetain_PVCOrphaned(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("retain-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceRetain(true),
	)
	agent.Finalizers = []string{FinalizerName}
	agent.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}

	// Pre-create the PVC with an owner reference pointing at the agent.
	pvcName := agent.Name + "-workspace"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "langop.io/v1alpha1",
					Kind:       "LanguageAgent",
					Name:       agent.Name,
					UID:        agent.UID,
					Controller: func() *bool { b := true; return &b }(),
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, pvc).
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
	require.NoError(t, err)

	// PVC must still exist (not garbage-collected) with no owner references.
	got := &corev1.PersistentVolumeClaim{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: "default"}, got))
	assert.Empty(t, got.OwnerReferences, "expected PVC owner references to be stripped when retain=true")
}

func TestLanguageAgentController_WorkspaceRetain_False_PVCKeepsOwnerRef(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("no-retain-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
	)

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

	// Two reconciles: first adds finalizer, second creates resources.
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pvc := &corev1.PersistentVolumeClaim{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name + "-workspace", Namespace: "default"}, pvc))
	assert.NotEmpty(t, pvc.OwnerReferences, "expected PVC to retain owner reference when retain=false")
}
