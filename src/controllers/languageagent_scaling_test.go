package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func TestLanguageAgentController_PDB_CreatedForMultiReplica(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("multi-agent", "default", gen.SetAgentReplicas(3))
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pdb := &policyv1.PodDisruptionBudget{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pdb))
	require.NotNil(t, pdb.Spec.MinAvailable)
	assert.Equal(t, int32(1), pdb.Spec.MinAvailable.IntVal)
	assert.Equal(t, GetCommonLabels(agent.Name, "LanguageAgent"), pdb.Spec.Selector.MatchLabels)
}

func TestLanguageAgentController_PDB_NotCreatedForSingleReplica(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("single-agent", "default")
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pdb := &policyv1.PodDisruptionBudget{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pdb)
	assert.True(t, errors.IsNotFound(err), "PDB must not exist for a single-replica agent")
}

func TestLanguageAgentController_PDB_DeletedWhenReplicasReducedToOne(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("scale-agent", "default", gen.SetAgentReplicas(3))
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	// First pair of reconciles: creates PDB for replicas=3.
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pdb := &policyv1.PodDisruptionBudget{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pdb), "PDB should exist after multi-replica reconcile")

	// Scale down to 1 replica.
	current := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, current))
	one := int32(1)
	current.Spec.Deployment.Replicas = &one
	require.NoError(t, fakeClient.Update(ctx, current))

	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pdb)
	assert.True(t, errors.IsNotFound(err), "PDB must be deleted after scaling down to 1 replica")
}

func TestLanguageAgentController_HPA_CreatedWhenAutoscalingSet(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("hpa-agent", "default", gen.SetAgentAutoscaling(2, 10))
	cluster := gen.ReadyCluster("default")
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, hpa))
	require.NotNil(t, hpa.Spec.MinReplicas)
	assert.Equal(t, int32(2), *hpa.Spec.MinReplicas)
	assert.Equal(t, int32(10), hpa.Spec.MaxReplicas)
	assert.Equal(t, "Deployment", hpa.Spec.ScaleTargetRef.Kind)
	assert.Equal(t, agent.Name, hpa.Spec.ScaleTargetRef.Name)
	// Default CPU metric injected when no metrics specified
	require.Len(t, hpa.Spec.Metrics, 1)
	assert.Equal(t, autoscalingv2.ResourceMetricSourceType, hpa.Spec.Metrics[0].Type)
	assert.Equal(t, corev1.ResourceCPU, hpa.Spec.Metrics[0].Resource.Name)
	assert.Equal(t, int32(80), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
}

func TestLanguageAgentController_HPA_NotCreatedWhenAutoscalingNil(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("no-hpa-agent", "default")
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, hpa)
	assert.True(t, errors.IsNotFound(err), "HPA must not exist when autoscaling is not configured")
}

func TestLanguageAgentController_HPA_DeletedWhenAutoscalingRemoved(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("rm-hpa-agent", "default", gen.SetAgentAutoscaling(2, 5))
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, hpa), "HPA should exist after autoscaling reconcile")

	// Remove autoscaling from the agent spec
	current := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, current))
	current.Spec.Deployment.Autoscaling = nil
	require.NoError(t, fakeClient.Update(ctx, current))

	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, hpa)
	assert.True(t, errors.IsNotFound(err), "HPA must be deleted when autoscaling is removed")
}

func TestLanguageAgentController_HPA_CustomMetricsPreserved(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	avgUtil := int32(60)
	agent := gen.LanguageAgent("custom-hpa-agent", "default")
	agent.Spec.Deployment.Autoscaling = &langopv1alpha1.AutoscalingSpec{
		MinReplicas: func() *int32 { v := int32(3); return &v }(),
		MaxReplicas: 20,
		Metrics: []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceMemory,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &avgUtil,
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, hpa))
	// Custom metrics must be preserved — no CPU default should be added
	require.Len(t, hpa.Spec.Metrics, 1)
	assert.Equal(t, corev1.ResourceMemory, hpa.Spec.Metrics[0].Resource.Name)
	assert.Equal(t, int32(60), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
}
