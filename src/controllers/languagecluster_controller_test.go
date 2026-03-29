package controllers

import (
	"context"
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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func clusterRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: name}}
}

func TestLanguageClusterController_BasicReconciliation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	// First reconcile adds finalizer and requeues
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	// Second reconcile should set status
	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue after status update")
	}

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))

	if updatedCluster.Status.Phase != events.PhaseStatusReady {
		t.Errorf("Expected phase %q, got '%s'", events.PhaseStatusReady, updatedCluster.Status.Phase)
	}
	if !controllerutil.ContainsFinalizer(updatedCluster, FinalizerName) {
		t.Error("Expected finalizer to be added")
	}
	if updatedCluster.Status.GatewayReady == nil {
		t.Error("Expected GatewayReady to be set (non-nil) after successful reconcile")
	} else if !*updatedCluster.Status.GatewayReady {
		t.Error("Expected GatewayReady=true after successful reconcile")
	}

	// Namespace should be created
	ns := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, ns))
}

func TestLanguageClusterController_ReadyCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-condition")
	cluster.Generation = 5

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))

	var readyCondition *metav1.Condition
	for i := range updatedCluster.Status.Conditions {
		if updatedCluster.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedCluster.Status.Conditions[i]
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
	if readyCondition.Message != "LanguageCluster is ready" {
		t.Errorf("Expected message 'LanguageCluster is ready', got '%s'", readyCondition.Message)
	}
	if readyCondition.ObservedGeneration != cluster.Generation {
		t.Errorf("Expected ObservedGeneration %d, got %d", cluster.Generation, readyCondition.ObservedGeneration)
	}
}

func TestLanguageClusterController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, clusterRequest("non-existent-cluster"))

	if err != nil {
		t.Errorf("Expected no error for not found cluster, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found cluster")
	}
}

func TestLanguageClusterController_MultipleReconciles(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-multiple")
	cluster.Generation = 2

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))

	if updatedCluster.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready' after multiple reconciles, got '%s'", updatedCluster.Status.Phase)
	}
}

func TestLanguageClusterController_Finalizer(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-finalizer")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))
	if !controllerutil.ContainsFinalizer(updatedCluster, FinalizerName) {
		t.Error("Expected finalizer to be added after first reconcile")
	}

	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)
}

func TestLanguageClusterController_DeletionWithoutDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-deletion")
	cluster.Finalizers = []string{FinalizerName}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: cluster.Name},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, ns).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster)
	if err == nil {
		for _, finalizer := range updatedCluster.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}

	// Namespace should be deleted
	deletedNs := &corev1.Namespace{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, deletedNs)
	require.True(t, errors.IsNotFound(err), "Expected namespace to be deleted")
}

func TestLanguageClusterController_DeletionWithDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-with-deps")
	cluster.Finalizers = []string{FinalizerName}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	agent := gen.LanguageAgent("test-agent", "test-cluster-with-deps",
		gen.SetAgentInstructions("test agent"),
	)

	tool := gen.LanguageTool("test-tool", "test-cluster-with-deps",
		gen.SetToolType("shell"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent, tool).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent)
	if err == nil {
		if updatedAgent.DeletionTimestamp.IsZero() {
			t.Error("Expected agent to be marked for deletion")
		}
	}

	updatedTool := &langopv1alpha1.LanguageTool{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, updatedTool)
	if err == nil {
		if updatedTool.DeletionTimestamp.IsZero() {
			t.Error("Expected tool to be marked for deletion")
		}
	}

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster)
	if err == nil {
		for _, finalizer := range updatedCluster.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}
}

func TestLanguageClusterController_CapacityQuota_Created(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	maxAgents := int32(5)
	maxCPU := resource.MustParse("4")
	maxMemory := resource.MustParse("8Gi")
	cluster := gen.LanguageCluster("quota-cluster",
		gen.SetClusterCapacity(&langopv1alpha1.ClusterCapacitySpec{
			MaxAgents: &maxAgents,
			MaxCPU:    &maxCPU,
			MaxMemory: &maxMemory,
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	// Second reconcile creates resources
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	quota := &corev1.ResourceQuota{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "langop-quota", Namespace: cluster.Name}, quota)
	require.NoError(t, err, "expected ResourceQuota to be created")

	gotAgents := quota.Spec.Hard["count/languageagents.langop.io"]
	gotCPU := quota.Spec.Hard[corev1.ResourceLimitsCPU]
	gotMemory := quota.Spec.Hard[corev1.ResourceLimitsMemory]
	assert.Equal(t, "5", gotAgents.String())
	assert.Equal(t, "4", gotCPU.String())
	assert.Equal(t, "8Gi", gotMemory.String())
}

func TestLanguageClusterController_CapacityQuota_Absent_When_SpecUnset(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("no-quota-cluster")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	quota := &corev1.ResourceQuota{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "langop-quota", Namespace: cluster.Name}, quota)
	assert.True(t, errors.IsNotFound(err), "expected no ResourceQuota when spec.capacity is unset")
}

func TestLanguageClusterController_CapacityStatus_WrittenWhenEmpty(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Cluster with no sub-resources — all counts should be zero, but status.capacity must be written.
	cluster := gen.LanguageCluster("empty-cluster")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))

	require.NotNil(t, updated.Status.Capacity, "status.capacity must be written even when cluster has no resources")
	assert.Equal(t, int32(0), updated.Status.Capacity.AgentCount)
	assert.Equal(t, int32(0), updated.Status.Capacity.ModelCount)
	assert.Equal(t, int32(0), updated.Status.Capacity.ToolCount)
	assert.Equal(t, int32(0), updated.Status.Capacity.PersonaCount)
}

func TestLanguageClusterController_CapacityStatus_Counts(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("status-cluster")
	agent1 := gen.LanguageAgent("agent-one", "status-cluster")
	agent2 := gen.LanguageAgent("agent-two", "status-cluster")
	model := gen.LanguageModel("model-one", "status-cluster")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent1, agent2, model).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))

	require.NotNil(t, updated.Status.Capacity)
	assert.Equal(t, int32(2), updated.Status.Capacity.AgentCount)
	assert.Equal(t, int32(1), updated.Status.Capacity.ModelCount)
	assert.Equal(t, int32(0), updated.Status.Capacity.ToolCount)
	assert.Equal(t, int32(0), updated.Status.Capacity.PersonaCount)
}

func TestLanguageClusterController_GatewayServiceTypeAndAnnotations(t *testing.T) {
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
			name:            "LoadBalancer applied",
			serviceType:     corev1.ServiceTypeLoadBalancer,
			wantServiceType: corev1.ServiceTypeLoadBalancer,
		},
		{
			name:            "annotations propagated",
			annotations:     map[string]string{"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"},
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			mods := []gen.LanguageClusterModifier{}
			if tt.serviceType != "" {
				mods = append(mods, gen.SetClusterGatewayServiceType(tt.serviceType))
			}
			if tt.annotations != nil {
				mods = append(mods, gen.SetClusterGatewayServiceAnnotations(tt.annotations))
			}
			cluster := gen.LanguageCluster("test-cluster", mods...)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(cluster).
				WithStatusSubresource(cluster).
				Build()

			reconciler := &LanguageClusterReconciler{
				Client: fakeClient,
				Scheme: scheme,
				Log:    logr.Discard(),
			}

			ctx := context.Background()
			req := clusterRequest(cluster.Name)

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, req)
			require.NoError(t, err)
			// Second reconcile creates resources
			_, err = reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			svc := &corev1.Service{}
			if err := fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, svc); err != nil {
				t.Fatalf("expected gateway service to exist: %v", err)
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

func TestLanguageClusterController_GatewaySchedulingFields(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster")
	cluster.Spec.Gateway = &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			NodeSelector: map[string]string{"cloud.google.com/gke-nodepool": "proxy-pool"},
			Tolerations: []corev1.Toleration{
				{Key: "dedicated", Value: "proxy", Operator: corev1.TolerationOpEqual, Effect: corev1.TaintEffectNoSchedule},
			},
			TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
				{MaxSkew: 1, TopologyKey: "kubernetes.io/hostname", WhenUnsatisfiable: corev1.DoNotSchedule},
			},
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								TopologyKey: "kubernetes.io/hostname",
							},
						},
					},
				},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "proxy-registry-secret"}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	// Second reconcile creates resources
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, deployment); err != nil {
		t.Fatalf("expected gateway deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec

	if podSpec.NodeSelector["cloud.google.com/gke-nodepool"] != "proxy-pool" {
		t.Errorf("Expected NodeSelector gke-nodepool=proxy-pool, got %v", podSpec.NodeSelector)
	}
	if len(podSpec.Tolerations) == 0 || podSpec.Tolerations[0].Key != "dedicated" {
		t.Errorf("Expected Toleration dedicated, got %v", podSpec.Tolerations)
	}
	if len(podSpec.TopologySpreadConstraints) == 0 || podSpec.TopologySpreadConstraints[0].TopologyKey != "kubernetes.io/hostname" {
		t.Errorf("Expected TopologySpreadConstraint, got %v", podSpec.TopologySpreadConstraints)
	}
	if podSpec.Affinity == nil || podSpec.Affinity.PodAntiAffinity == nil {
		t.Error("Expected Affinity.PodAntiAffinity to be set")
	}
	if len(podSpec.ImagePullSecrets) == 0 || podSpec.ImagePullSecrets[0].Name != "proxy-registry-secret" {
		t.Errorf("Expected ImagePullSecret proxy-registry-secret, got %v", podSpec.ImagePullSecrets)
	}
}
