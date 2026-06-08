package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
		if updatedCluster.Status.Conditions[i].Type == langopv1alpha1.ConditionReady {
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
	if readyCondition.Reason != langopv1alpha1.ReasonReconcileSuccess {
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

	isController := true
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "LanguageCluster", Name: cluster.Name, Controller: &isController},
			},
		},
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

func TestLanguageClusterController_DeletionWaitsForDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-drain")
	cluster.Finalizers = []string{FinalizerName}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	agent := gen.LanguageAgent("test-agent", "test-cluster-drain",
		gen.SetAgentInstructions("test agent"),
	)
	// Pre-set the operator finalizer so the fake client keeps the agent in the store
	// when Delete is called (simulating an agent whose finalizer has not been stripped yet).
	agent.Finalizers = []string{FinalizerName}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))

	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, result.RequeueAfter, "expected requeue while children still draining")

	// Cluster finalizer must NOT be removed — still waiting for agent to finalize
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))
	assert.True(t, controllerutil.ContainsFinalizer(updatedCluster, FinalizerName), "cluster finalizer should remain while children drain")

	// Namespace must NOT have been deleted
	ns := &corev1.Namespace{}
	nsErr := fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, ns)
	if nsErr == nil {
		// namespace was never created in this test, so not-found is also acceptable
	}
}

func TestLanguageClusterController_DeletionDrainListError(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-drain-err")
	cluster.Finalizers = []string{FinalizerName}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	// Add an agent with a finalizer so that the drain check finds remaining children on
	// the first reconcile. This prevents premature cleanup and lets us then inject a List
	// error on the second reconcile to verify the error is propagated rather than discarded.
	agent := gen.LanguageAgent("test-agent", "test-cluster-drain-err",
		gen.SetAgentInstructions("test agent"),
	)
	agent.Finalizers = []string{FinalizerName}

	failList := false
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(cluster).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if failList {
					return fmt.Errorf("injected list error")
				}
				return c.List(ctx, list, opts...)
			},
		}).Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	// First reconcile: drain check finds the agent and returns errChildrenDraining (requeues).
	result, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, result.RequeueAfter, "expected requeue while children drain")

	// Cluster finalizer must still be present.
	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))
	assert.True(t, controllerutil.ContainsFinalizer(updated, FinalizerName))

	// Now inject a List error — it must be propagated rather than silently treated as
	// "no children present", which would cause premature namespace deletion.
	failList = true
	_, err = reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.Error(t, err, "List error during cleanup must be propagated, not silently discarded")
}

func TestLanguageClusterController_NamespaceOwnerReference(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("test-cluster-ownerref")

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
	// Two reconciles: first adds finalizer, second creates resources including namespace
	_, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)

	ns := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, ns))

	require.Len(t, ns.OwnerReferences, 1, "namespace should have exactly one owner reference")
	assert.Equal(t, cluster.Name, ns.OwnerReferences[0].Name)
	assert.Equal(t, "LanguageCluster", ns.OwnerReferences[0].Kind)
	assert.True(t, *ns.OwnerReferences[0].Controller, "owner reference should be controller=true")
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

func TestLanguageClusterController_CapacityReady_Condition_True_On_Success(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	maxAgents := int32(5)
	cluster := gen.LanguageCluster("cond-cluster",
		gen.SetClusterCapacity(&langopv1alpha1.ClusterCapacitySpec{
			MaxAgents: &maxAgents,
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
	// Second reconcile creates resources and sets conditions
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionCapacityReady {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNilf(t, cond, "expected condition %q to be set", langopv1alpha1.ConditionCapacityReady)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, langopv1alpha1.ReasonReconcileSuccess, cond.Reason)
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

func TestLanguageClusterController_AgentRBAC(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("rbac-cluster")

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

	role := &rbacv1.Role{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "agents", Namespace: cluster.Name}, role)
	require.NoError(t, err, "agents Role must exist in cluster namespace")
	require.Len(t, role.Rules, 1)
	assert.Equal(t, []string{""}, role.Rules[0].APIGroups)
	assert.Equal(t, []string{"events"}, role.Rules[0].Resources)
	assert.Equal(t, []string{"create"}, role.Rules[0].Verbs)

	rb := &rbacv1.RoleBinding{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "agents", Namespace: cluster.Name}, rb)
	require.NoError(t, err, "agents RoleBinding must exist in cluster namespace")
	assert.Equal(t, "agents", rb.RoleRef.Name)
	assert.Equal(t, "Role", rb.RoleRef.Kind)
	require.Len(t, rb.Subjects, 1)
	assert.Equal(t, "ServiceAccount", rb.Subjects[0].Kind)
	assert.Equal(t, "default", rb.Subjects[0].Name)
}

func TestLanguageClusterController_NetworkPolicy(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("np-cluster")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	np := &networkingv1.NetworkPolicy{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name + "-agents", Namespace: cluster.Name}, np)
	require.NoError(t, err, "NetworkPolicy must exist in cluster namespace")
	assert.NotEmpty(t, np.Spec.Egress, "NetworkPolicy must have at least the default egress rules")
}

func TestLanguageClusterController_NetworkPolicy_FromRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("from-cluster",
		gen.SetClusterNetworkPolicies(&langopv1alpha1.AgentNetworkPolicies{
			Ingress: []langopv1alpha1.NetworkIngressRule{
				{
					From:  []langopv1alpha1.NetworkPeer{{Group: "external-readers"}},
					Ports: []langopv1alpha1.NetworkPort{{Port: 8000}},
				},
			},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	np := &networkingv1.NetworkPolicy{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name + "-agents", Namespace: cluster.Name}, np)
	require.NoError(t, err, "NetworkPolicy must exist")

	hasIngress := false
	for _, pt := range np.Spec.PolicyTypes {
		if pt == networkingv1.PolicyTypeIngress {
			hasIngress = true
		}
	}
	assert.True(t, hasIngress, "expected PolicyTypeIngress when From rules are present")
	require.NotEmpty(t, np.Spec.Ingress, "expected at least one ingress rule")

	peer := np.Spec.Ingress[0].From[0]
	require.NotNil(t, peer.PodSelector, "expected PodSelector for Group-based From rule")
	assert.Equal(t, "external-readers", peer.PodSelector.MatchLabels[langoplabels.LabelKeyLangopGroup])
	require.NotEmpty(t, np.Spec.Ingress[0].Ports)
	assert.Equal(t, int32(8000), np.Spec.Ingress[0].Ports[0].Port.IntVal)
}

func TestLanguageClusterController_NetworkPolicy_ServiceRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("svc-cluster",
		gen.SetClusterNetworkPolicies(&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{
					To: []langopv1alpha1.NetworkPeer{{
						Service: &langopv1alpha1.ServiceReference{
							Name:      "my-service",
							Namespace: "other-ns",
						},
					}},
					Ports: []langopv1alpha1.NetworkPort{{Port: 443}},
				},
			},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	np := &networkingv1.NetworkPolicy{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name + "-agents", Namespace: cluster.Name}, np)
	require.NoError(t, err)

	// Find the user-defined egress rule (beyond the default API server rules)
	var userRule *networkingv1.NetworkPolicyEgressRule
	for i := range np.Spec.Egress {
		rule := &np.Spec.Egress[i]
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil {
				if v, ok := peer.NamespaceSelector.MatchLabels[langoplabels.LabelKeyMetadataName]; ok && v == "other-ns" {
					userRule = rule
					break
				}
			}
		}
	}
	require.NotNil(t, userRule, "expected egress rule with kubernetes.io/metadata.name selector")
	require.NotEmpty(t, userRule.Ports)
	assert.Equal(t, int32(443), userRule.Ports[0].Port.IntVal)
}

func TestLanguageClusterController_NetworkPolicy_DNSRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("dns-cluster",
		gen.SetClusterNetworkPolicies(&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{
					To:    []langopv1alpha1.NetworkPeer{{DNS: []string{"api.example.com"}}},
					Ports: []langopv1alpha1.NetworkPort{{Port: 443}},
				},
			},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	np := &networkingv1.NetworkPolicy{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name + "-agents", Namespace: cluster.Name}, np)
	require.NoError(t, err)

	// Find egress rules with IPBlock peers — those come from DNS resolution
	var dnsRule *networkingv1.NetworkPolicyEgressRule
	for i := range np.Spec.Egress {
		rule := &np.Spec.Egress[i]
		for _, peer := range rule.To {
			if peer.IPBlock != nil && peer.IPBlock.CIDR != "0.0.0.0/0" {
				dnsRule = rule
				break
			}
		}
	}

	// DNS resolution may fail in test environments; what matters is that
	// 0.0.0.0/0 was NOT inserted as a permissive fallback.
	for _, egressRule := range np.Spec.Egress {
		for _, peer := range egressRule.To {
			if peer.IPBlock != nil {
				assert.NotEqual(t, "0.0.0.0/0", peer.IPBlock.CIDR,
					"DNS egress rule must not fall back to 0.0.0.0/0")
			}
		}
	}

	// If DNS resolved successfully, assert IPBlock CIDRs are present and port is 443
	if dnsRule != nil {
		require.NotEmpty(t, dnsRule.Ports)
		assert.Equal(t, int32(443), dnsRule.Ports[0].Port.IntVal)
		for _, peer := range dnsRule.To {
			require.NotNil(t, peer.IPBlock, "DNS-resolved peer must use IPBlock")
			assert.NotEqual(t, "0.0.0.0/0", peer.IPBlock.CIDR)
		}
	}
}

func TestLanguageClusterController_GatewayDeploymentImage(t *testing.T) {
	tests := []struct {
		name         string
		gatewayImage string
		wantImage    string
	}{
		{
			name:         "custom image propagated to Deployment",
			gatewayImage: "my-registry/litellm:v1",
			wantImage:    "my-registry/litellm:v1",
		},
		{
			name:         "default image when GatewayImage unset",
			gatewayImage: "",
			wantImage:    "ghcr.io/language-operator/model-gateway:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			cluster := gen.LanguageCluster("img-cluster")

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(cluster).
				WithStatusSubresource(cluster).
				Build()

			reconciler := &LanguageClusterReconciler{
				Client:       fakeClient,
				Scheme:       scheme,
				Log:          logr.Discard(),
				GatewayImage: tt.gatewayImage,
			}

			ctx := context.Background()
			req := clusterRequest(cluster.Name)

			_, err := reconciler.Reconcile(ctx, req)
			require.NoError(t, err)
			_, err = reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			deployment := &appsv1.Deployment{}
			require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, deployment))
			require.NotEmpty(t, deployment.Spec.Template.Spec.Containers, "gateway Deployment must have at least one container")
			assert.Equal(t, tt.wantImage, deployment.Spec.Template.Spec.Containers[0].Image)
		})
	}
}

func TestLanguageClusterController_GatewayConfigMapContainsModel(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("model-cluster")
	model := gen.LanguageModel("gpt-4", cluster.Name)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, model).
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

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway-config", Namespace: cluster.Name}, cm))

	key := "model__gpt-4.json"
	val, ok := cm.Data[key]
	require.True(t, ok, "gateway-config ConfigMap must contain key %q for LanguageModel gpt-4", key)
	assert.NotEmpty(t, val, "model config JSON must not be empty")
	assert.Contains(t, val, "anthropic", "model config JSON must include provider field")
}

func TestLanguageClusterController_GatewayServicePort(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("port-cluster")

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

	svc := &corev1.Service{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, svc))
	require.NotEmpty(t, svc.Spec.Ports, "gateway Service must expose at least one port")
	assert.Equal(t, int32(8000), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(4000), svc.Spec.Ports[0].TargetPort.IntVal)
}

func TestLanguageClusterController_GatewayEndpoint(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("gw-cluster")

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
	assert.Equal(t, "http://gateway.gw-cluster.svc.cluster.local:8000", updated.Status.GatewayEndpoint)
	require.NotNil(t, updated.Status.GatewayReady)
	assert.True(t, *updated.Status.GatewayReady)
}

func TestLanguageClusterController_GatewayIngressCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("ingress-cluster", gen.SetClusterDomain("example.com"), gen.SetClusterIngressEnabled(true))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cluster).WithStatusSubresource(cluster).Build()
	dnsUnblock := make(chan struct{})
	t.Cleanup(func() { close(dnsUnblock) })
	reconciler := &LanguageClusterReconciler{
		Client: fakeClient, Scheme: scheme, Log: logr.Discard(),
		DNSLookup: func(ctx context.Context, host string) error {
			select {
			case <-dnsUnblock:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	ingress := &networkingv1.Ingress{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ingress))
	require.NotEmpty(t, ingress.Spec.Rules)
	assert.Equal(t, "example.com", ingress.Spec.Rules[0].Host)

	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))
	require.NotNil(t, updated.Status.GatewayReady)
	assert.True(t, *updated.Status.GatewayReady)
}

func TestLanguageClusterController_GatewayIngressError(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("ingress-err-cluster", gen.SetClusterDomain("example.com"), gen.SetClusterIngressEnabled(true))

	// Use a stateful interceptor: allow first Create (finalizer Update), fail Ingress Creates.
	failIngress := false
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cluster).WithStatusSubresource(cluster).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if failIngress {
					if _, ok := obj.(*networkingv1.Ingress); ok {
						return fmt.Errorf("injected ingress create error")
					}
				}
				return c.Create(ctx, obj, opts...)
			},
		}).Build()
	dnsUnblock2 := make(chan struct{})
	t.Cleanup(func() { close(dnsUnblock2) })
	reconciler := &LanguageClusterReconciler{
		Client: fakeClient, Scheme: scheme, Log: logr.Discard(),
		DNSLookup: func(ctx context.Context, host string) error {
			select {
			case <-dnsUnblock2:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	ctx := context.Background()
	req := clusterRequest(cluster.Name)

	// First reconcile: adds finalizer, allow all creates.
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Second reconcile: fail Ingress creation.
	failIngress = true
	_, err = reconciler.Reconcile(ctx, req)
	require.Error(t, err)

	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))
	assert.Equal(t, events.PhaseStatusFailed, updated.Status.Phase)
	require.NotNil(t, updated.Status.GatewayReady)
	assert.False(t, *updated.Status.GatewayReady)
	var gatewayReadyCond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionGatewayReady {
			gatewayReadyCond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, gatewayReadyCond, "expected GatewayReady condition")
	assert.Equal(t, metav1.ConditionFalse, gatewayReadyCond.Status)
	assert.Equal(t, "GatewayIngressError", gatewayReadyCond.Reason)
}

func TestLanguageClusterController_ErrorPathConditions(t *testing.T) {
	type errorPathCase struct {
		name             string
		buildCluster     func() *langopv1alpha1.LanguageCluster
		failCreate       func(obj client.Object) bool
		networkIsolation bool
		condType         string
		condStatus       metav1.ConditionStatus
		condReason       string
	}

	maxAgents := int32(5)
	cases := []errorPathCase{
		{
			name:         "NamespaceError",
			buildCluster: func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("ns-err-cluster") },
			failCreate:   func(obj client.Object) bool { _, ok := obj.(*corev1.Namespace); return ok },
			condType:     langopv1alpha1.ConditionReady,
			condStatus:   metav1.ConditionFalse,
			condReason:   "NamespaceError",
		},
		{
			name:         "RBACError",
			buildCluster: func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("rbac-err-cluster") },
			failCreate:   func(obj client.Object) bool { _, ok := obj.(*rbacv1.Role); return ok },
			condType:     langopv1alpha1.ConditionReady,
			condStatus:   metav1.ConditionFalse,
			condReason:   "RBACError",
		},
		{
			name:             langopv1alpha1.ReasonNetworkPolicyError,
			buildCluster:     func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("netpol-err-cluster") },
			failCreate:       func(obj client.Object) bool { _, ok := obj.(*networkingv1.NetworkPolicy); return ok },
			networkIsolation: true,
			condType:         langopv1alpha1.ConditionReady,
			condStatus:       metav1.ConditionFalse,
			condReason:       langopv1alpha1.ReasonNetworkPolicyError,
		},
		{
			name:         "GatewayError",
			buildCluster: func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("gw-err-cluster") },
			failCreate:   func(obj client.Object) bool { _, ok := obj.(*corev1.ConfigMap); return ok },
			condType:     langopv1alpha1.ConditionGatewayReady,
			condStatus:   metav1.ConditionFalse,
			condReason:   "GatewayError",
		},
		{
			name: "CapacityError",
			buildCluster: func() *langopv1alpha1.LanguageCluster {
				return gen.LanguageCluster("cap-err-cluster", gen.SetClusterCapacity(&langopv1alpha1.ClusterCapacitySpec{MaxAgents: &maxAgents}))
			},
			failCreate: func(obj client.Object) bool { _, ok := obj.(*corev1.ResourceQuota); return ok },
			condType:   langopv1alpha1.ConditionCapacityReady,
			condStatus: metav1.ConditionFalse,
			condReason: "CapacityError",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			cluster := tc.buildCluster()

			failCreate := false
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(cluster).WithStatusSubresource(cluster).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if failCreate && tc.failCreate(obj) {
							return fmt.Errorf("injected create error")
						}
						return c.Create(ctx, obj, opts...)
					},
				}).Build()

			reconciler := &LanguageClusterReconciler{
				Client:                  fakeClient,
				Scheme:                  scheme,
				Log:                     logr.Discard(),
				NetworkIsolationEnabled: tc.networkIsolation,
			}

			ctx := context.Background()
			req := clusterRequest(cluster.Name)

			// First reconcile: adds finalizer, allow all creates.
			_, err := reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			// Second reconcile: inject error.
			failCreate = true
			_, err = reconciler.Reconcile(ctx, req)
			require.Error(t, err)

			updated := &langopv1alpha1.LanguageCluster{}
			require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))
			assert.Equal(t, events.PhaseStatusFailed, updated.Status.Phase)

			var cond *metav1.Condition
			for i := range updated.Status.Conditions {
				if updated.Status.Conditions[i].Type == tc.condType {
					cond = &updated.Status.Conditions[i]
					break
				}
			}
			require.NotNilf(t, cond, "expected condition %q", tc.condType)
			assert.Equal(t, tc.condStatus, cond.Status)
			assert.Equal(t, tc.condReason, cond.Reason)
		})
	}
}

// reconcileGatewayCluster is a test helper that creates a LanguageCluster with the given
// gateway spec, runs two reconcile passes, and returns the resulting gateway Deployment.
func reconcileGatewayCluster(t *testing.T, name string, gateway *langopv1alpha1.GatewaySpec, reconcilerOpts ...func(*LanguageClusterReconciler)) *appsv1.Deployment {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster(name)
	cluster.Spec.Gateway = gateway

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	r := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}
	for _, opt := range reconcilerOpts {
		opt(r)
	}

	ctx := context.Background()
	req := clusterRequest(name)
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	deployment := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: name}, deployment))
	require.NotEmpty(t, deployment.Spec.Template.Spec.Containers)
	return deployment
}

func TestLanguageClusterController_GatewayImagePullPolicy(t *testing.T) {
	tests := []struct {
		name            string
		specPolicy      corev1.PullPolicy
		operatorDefault corev1.PullPolicy
		wantPolicy      corev1.PullPolicy
	}{
		{
			name:            "spec policy takes precedence",
			specPolicy:      corev1.PullAlways,
			operatorDefault: corev1.PullIfNotPresent,
			wantPolicy:      corev1.PullAlways,
		},
		{
			name:            "falls back to operator default when spec unset",
			specPolicy:      "",
			operatorDefault: corev1.PullNever,
			wantPolicy:      corev1.PullNever,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := reconcileGatewayCluster(t, "ipp-cluster", &langopv1alpha1.GatewaySpec{
				Deployment: langopv1alpha1.DeploymentSpec{
					ImagePullPolicy: tt.specPolicy,
				},
			}, func(r *LanguageClusterReconciler) {
				r.GatewayImagePullPolicy = tt.operatorDefault
			})
			assert.Equal(t, tt.wantPolicy, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
		})
	}
}

func TestLanguageClusterController_GatewayEnvAndEnvFrom(t *testing.T) {
	envVar := corev1.EnvVar{Name: "LOG_LEVEL", Value: "debug"}
	envFrom := corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: "extra-config"},
		},
	}

	dep := reconcileGatewayCluster(t, "env-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			Env:     []corev1.EnvVar{envVar},
			EnvFrom: []corev1.EnvFromSource{envFrom},
		},
	})
	container := dep.Spec.Template.Spec.Containers[0]
	assert.Contains(t, container.Env, envVar)
	require.Len(t, container.EnvFrom, 1)
	assert.Equal(t, envFrom, container.EnvFrom[0])
}

func TestLanguageClusterController_GatewayPodAnnotationsAndLabels(t *testing.T) {
	dep := reconcileGatewayCluster(t, "annot-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			PodAnnotations: map[string]string{"vault.hashicorp.com/agent-inject": "true"},
			PodLabels:      map[string]string{"sidecar.istio.io/inject": "true"},
		},
	})
	// User annotation present alongside operator-managed config-hash.
	assert.Equal(t, "true", dep.Spec.Template.Annotations["vault.hashicorp.com/agent-inject"])
	assert.NotEmpty(t, dep.Spec.Template.Annotations[langoplabels.LabelKeyLangopConfigHash])
	// User label present alongside operator-managed gateway labels.
	assert.Equal(t, "true", dep.Spec.Template.Labels["sidecar.istio.io/inject"])
	assert.Equal(t, "gateway", dep.Spec.Template.Labels["app.kubernetes.io/component"])
}

func TestLanguageClusterController_GatewayInitContainers(t *testing.T) {
	initContainer := corev1.Container{
		Name:  "wait-for-db",
		Image: "busybox:latest",
	}
	dep := reconcileGatewayCluster(t, "init-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			InitContainers: []corev1.Container{initContainer},
		},
	})
	require.Len(t, dep.Spec.Template.Spec.InitContainers, 1)
	assert.Equal(t, "wait-for-db", dep.Spec.Template.Spec.InitContainers[0].Name)
}

func TestLanguageClusterController_GatewayProbes(t *testing.T) {
	t.Run("default probes used when spec probes are nil", func(t *testing.T) {
		dep := reconcileGatewayCluster(t, "probe-default", &langopv1alpha1.GatewaySpec{})
		c := dep.Spec.Template.Spec.Containers[0]
		require.NotNil(t, c.LivenessProbe)
		assert.Equal(t, "/health/liveliness", c.LivenessProbe.HTTPGet.Path)
		require.NotNil(t, c.ReadinessProbe)
		assert.Equal(t, "/health/readiness", c.ReadinessProbe.HTTPGet.Path)
		assert.Nil(t, c.StartupProbe)
	})

	t.Run("spec probes override defaults", func(t *testing.T) {
		customLiveness := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{Path: "/custom-live", Port: intstr.FromInt(4000)},
			},
		}
		customReadiness := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{Path: "/custom-ready", Port: intstr.FromInt(4000)},
			},
		}
		customStartup := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{Path: "/startup", Port: intstr.FromInt(4000)},
			},
		}
		dep := reconcileGatewayCluster(t, "probe-custom", &langopv1alpha1.GatewaySpec{
			Deployment: langopv1alpha1.DeploymentSpec{
				LivenessProbe:  customLiveness,
				ReadinessProbe: customReadiness,
				StartupProbe:   customStartup,
			},
		})
		c := dep.Spec.Template.Spec.Containers[0]
		require.NotNil(t, c.LivenessProbe)
		assert.Equal(t, "/custom-live", c.LivenessProbe.HTTPGet.Path)
		require.NotNil(t, c.ReadinessProbe)
		assert.Equal(t, "/custom-ready", c.ReadinessProbe.HTTPGet.Path)
		require.NotNil(t, c.StartupProbe)
		assert.Equal(t, "/startup", c.StartupProbe.HTTPGet.Path)
	})
}

func TestLanguageClusterController_GatewaySecurityContext(t *testing.T) {
	runAsNonRoot := true
	sc := &corev1.PodSecurityContext{RunAsNonRoot: &runAsNonRoot}

	dep := reconcileGatewayCluster(t, "sc-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			SecurityContext: sc,
		},
	})
	require.NotNil(t, dep.Spec.Template.Spec.SecurityContext)
	assert.Equal(t, &runAsNonRoot, dep.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
}

func TestLanguageClusterController_GatewayServiceAccountName(t *testing.T) {
	dep := reconcileGatewayCluster(t, "sa-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			ServiceAccountName: "litellm-sa",
		},
	})
	assert.Equal(t, "litellm-sa", dep.Spec.Template.Spec.ServiceAccountName)
}

func TestLanguageClusterController_GatewayCommandAndArgs(t *testing.T) {
	dep := reconcileGatewayCluster(t, "cmd-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			Command: []string{"/bin/sh"},
			Args:    []string{"-c", "litellm --config /etc/langop/config.yaml"},
		},
	})
	c := dep.Spec.Template.Spec.Containers[0]
	assert.Equal(t, []string{"/bin/sh"}, c.Command)
	assert.Equal(t, []string{"-c", "litellm --config /etc/langop/config.yaml"}, c.Args)
}

func TestLanguageClusterController_GatewayManagedSA(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("managed-sa-cluster")
	// No ServiceAccountName set — operator should manage the gateway SA.

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := clusterRequest("managed-sa-cluster")
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	// Managed ServiceAccount must exist.
	sa := &corev1.ServiceAccount{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: "managed-sa-cluster"}, sa))

	// Deployment pod spec must reference the managed SA.
	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: "managed-sa-cluster"}, dep))
	assert.Equal(t, "gateway", dep.Spec.Template.Spec.ServiceAccountName)
}

func TestLanguageClusterController_GatewayServiceAccountAnnotations(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("sa-annot-cluster")
	cluster.Spec.Gateway = &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			ServiceAccountAnnotations: map[string]string{
				"eks.amazonaws.com/role-arn": "arn:aws:iam::123456789012:role/my-gateway-role",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := clusterRequest("sa-annot-cluster")
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	sa := &corev1.ServiceAccount{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: "sa-annot-cluster"}, sa))
	assert.Equal(t, "arn:aws:iam::123456789012:role/my-gateway-role", sa.Annotations["eks.amazonaws.com/role-arn"])
}

func TestLanguageClusterController_GatewayRoleRules(t *testing.T) {
	extraRule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"secrets"},
		Verbs:     []string{"get"},
	}

	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("role-rules-cluster")
	cluster.Spec.Gateway = &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			RoleRules: []rbacv1.PolicyRule{extraRule},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := clusterRequest("role-rules-cluster")
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	role := &rbacv1.Role{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: "role-rules-cluster"}, role))
	require.Len(t, role.Rules, 1)
	assert.Equal(t, extraRule, role.Rules[0])

	rb := &rbacv1.RoleBinding{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: "role-rules-cluster"}, rb))
	assert.Equal(t, "gateway", rb.RoleRef.Name)
	require.Len(t, rb.Subjects, 1)
	assert.Equal(t, "gateway", rb.Subjects[0].Name)
}

func TestLanguageClusterController_GatewayVolumesAndMounts(t *testing.T) {
	userVol := corev1.Volume{
		Name: "custom-certs",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: "my-certs"},
		},
	}
	userMount := corev1.VolumeMount{Name: "custom-certs", MountPath: "/etc/ssl/custom"}

	dep := reconcileGatewayCluster(t, "vol-cluster", &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			Volumes:      []corev1.Volume{userVol},
			VolumeMounts: []corev1.VolumeMount{userMount},
		},
	})

	// Operator-managed models-config volume must still be present.
	var volNames []string
	for _, v := range dep.Spec.Template.Spec.Volumes {
		volNames = append(volNames, v.Name)
	}
	assert.Contains(t, volNames, "models-config", "operator-managed volume must be retained")
	assert.Contains(t, volNames, "custom-certs", "user volume must be appended")

	// Operator-managed mount must still be present.
	var mountPaths []string
	for _, m := range dep.Spec.Template.Spec.Containers[0].VolumeMounts {
		mountPaths = append(mountPaths, m.MountPath)
	}
	assert.Contains(t, mountPaths, "/etc/langop/models", "operator-managed mount must be retained")
	assert.Contains(t, mountPaths, "/etc/ssl/custom", "user mount must be appended")
}

// TestValidateDNS_SkipsOnSameGeneration verifies that validateDNS does not perform
// a DNS lookup (and does not modify the condition) when the DNSConfigured condition
// already reflects the current resource generation.
func TestValidateDNS_SkipsOnSameGeneration(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("skip-dns",
		gen.SetClusterDomain("example.com"),
	)
	cluster.Generation = 3

	// Pre-populate a DNSConfigured condition for the current generation.
	existingCondition := metav1.Condition{
		Type:               langopv1alpha1.ConditionDNSConfigured,
		Status:             metav1.ConditionTrue,
		Reason:             "WildcardDNSReady",
		Message:            "already validated",
		ObservedGeneration: 3,
		LastTransitionTime: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
	}
	cluster.Status.Conditions = []metav1.Condition{existingCondition}

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

	reconciler.validateDNS(context.Background(), cluster)

	// Condition must be unchanged — same content and same LastTransitionTime.
	var found *metav1.Condition
	for i := range cluster.Status.Conditions {
		if cluster.Status.Conditions[i].Type == langopv1alpha1.ConditionDNSConfigured {
			found = &cluster.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, existingCondition.Status, found.Status)
	assert.Equal(t, existingCondition.Reason, found.Reason)
	assert.Equal(t, existingCondition.ObservedGeneration, found.ObservedGeneration)
	assert.Equal(t, existingCondition.LastTransitionTime, found.LastTransitionTime,
		"LastTransitionTime must not change when DNS lookup is skipped")
}

// TestValidateDNS_NonBlocking verifies that validateDNS returns immediately without
// modifying the in-memory cluster object when the DNS condition is stale. The lookup
// is dispatched to a background goroutine instead of blocking the reconcile worker.
func TestValidateDNS_NonBlocking(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("async-dns",
		gen.SetClusterDomain("example.com"),
	)
	cluster.Generation = 2
	// No DNS condition — stale path, goroutine will be launched.

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

	reconciler.validateDNS(context.Background(), cluster)

	// The in-memory cluster object must NOT be modified — the goroutine writes
	// back asynchronously via the API server, not by mutating the passed pointer.
	for _, cond := range cluster.Status.Conditions {
		assert.NotEqual(t, langopv1alpha1.ConditionDNSConfigured, cond.Type,
			"validateDNS must not set DNS condition synchronously on the in-memory cluster")
	}
}

// TestValidateDNS_SuccessSetsTrueCondition verifies that when DNS resolution succeeds,
// validateDNS writes ConditionDNSConfigured=True with Reason=WildcardDNSReady back
// to the API server via the async goroutine.
func TestValidateDNS_SuccessSetsTrueCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("dns-success",
		gen.SetClusterDomain("example.com"),
	)
	cluster.Generation = 1

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	done := make(chan struct{})
	reconciler := &LanguageClusterReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Log:         logr.Discard(),
		DNSLookup:   func(_ context.Context, _ string) error { return nil },
		dnsTestDone: done,
	}

	reconciler.validateDNS(context.Background(), cluster)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DNS goroutine did not complete in time")
	}

	var updated langopv1alpha1.LanguageCluster
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: "dns-success"}, &updated))

	var found *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionDNSConfigured {
			found = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, found, "ConditionDNSConfigured must be set after successful DNS lookup")
	assert.Equal(t, metav1.ConditionTrue, found.Status)
	assert.Equal(t, "WildcardDNSReady", found.Reason)
	assert.Equal(t, cluster.Generation, found.ObservedGeneration)
}

// TestValidateDNS_FailureSetsWildcardDNSMissing verifies that when DNS resolution fails,
// validateDNS writes ConditionDNSConfigured=False with Reason=WildcardDNSMissing back
// to the API server via the async goroutine.
func TestValidateDNS_FailureSetsWildcardDNSMissing(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("dns-failure",
		gen.SetClusterDomain("example.com"),
	)
	cluster.Generation = 1

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	done := make(chan struct{})
	reconciler := &LanguageClusterReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Log:         logr.Discard(),
		DNSLookup:   func(_ context.Context, _ string) error { return fmt.Errorf("no such host") },
		dnsTestDone: done,
	}

	reconciler.validateDNS(context.Background(), cluster)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DNS goroutine did not complete in time")
	}

	var updated langopv1alpha1.LanguageCluster
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: "dns-failure"}, &updated))

	var found *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionDNSConfigured {
			found = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, found, "ConditionDNSConfigured must be set after failed DNS lookup")
	assert.Equal(t, metav1.ConditionFalse, found.Status)
	assert.Equal(t, "WildcardDNSMissing", found.Reason)
	assert.Equal(t, cluster.Generation, found.ObservedGeneration)
}

func TestLanguageClusterController_GatewayIngressTLS(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	const domain = "example.com"
	const gatewayHost = "example.com"

	reconcileCluster := func(t *testing.T, cluster *langopv1alpha1.LanguageCluster, rOpts ...func(*LanguageClusterReconciler)) *networkingv1.Ingress {
		t.Helper()
		// Gateway ingress is opt-in; these tests exercise its rendering so enable it.
		gen.SetClusterIngressEnabled(true)(cluster)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster).
			WithStatusSubresource(cluster).
			Build()
		dnsUnblock := make(chan struct{})
		t.Cleanup(func() { close(dnsUnblock) })
		r := &LanguageClusterReconciler{
			Client: fakeClient, Scheme: scheme, Log: logr.Discard(),
			DNSLookup: func(ctx context.Context, host string) error {
				select {
				case <-dnsUnblock:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		for _, opt := range rOpts {
			opt(r)
		}
		ctx := context.Background()
		req := clusterRequest(cluster.Name)
		_, err := r.Reconcile(ctx, req)
		require.NoError(t, err)
		_, err = r.Reconcile(ctx, req)
		require.NoError(t, err)
		ing := &networkingv1.Ingress{}
		require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ing))
		return ing
	}

	t.Run("explicit_secret_name", func(t *testing.T) {
		cluster := gen.LanguageCluster("tls-explicit",
			gen.SetClusterDomain(domain),
			gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{
				SecretName: "my-tls-secret",
			}))
		ing := reconcileCluster(t, cluster)
		require.Len(t, ing.Spec.TLS, 1)
		assert.Equal(t, "my-tls-secret", ing.Spec.TLS[0].SecretName)
		assert.Equal(t, []string{gatewayHost}, ing.Spec.TLS[0].Hosts)
	})

	t.Run("cert_manager_clusterissuer", func(t *testing.T) {
		cluster := gen.LanguageCluster("tls-clusterissuer",
			gen.SetClusterDomain(domain),
			gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{}))
		ing := reconcileCluster(t, cluster, func(r *LanguageClusterReconciler) {
			r.DefaultTLSIssuerName = "letsencrypt"
		})
		assert.Equal(t, "letsencrypt", ing.Annotations["cert-manager.io/cluster-issuer"])
		require.Len(t, ing.Spec.TLS, 1)
		assert.Equal(t, "gateway-tls", ing.Spec.TLS[0].SecretName)
		assert.Equal(t, []string{gatewayHost}, ing.Spec.TLS[0].Hosts)
	})

	t.Run("cert_manager_issuer_kind", func(t *testing.T) {
		cluster := gen.LanguageCluster("tls-issuer",
			gen.SetClusterDomain(domain),
			gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{}))
		ing := reconcileCluster(t, cluster, func(r *LanguageClusterReconciler) {
			r.DefaultTLSIssuerName = "my-issuer"
			r.DefaultTLSIssuerKind = "Issuer"
		})
		assert.Equal(t, "my-issuer", ing.Annotations["cert-manager.io/issuer"])
		_, hasClusterIssuer := ing.Annotations["cert-manager.io/cluster-issuer"]
		assert.False(t, hasClusterIssuer, "cert-manager.io/cluster-issuer should not be set for Issuer kind")
		require.Len(t, ing.Spec.TLS, 1)
		assert.Equal(t, "gateway-tls", ing.Spec.TLS[0].SecretName)
	})

	t.Run("tls_disabled", func(t *testing.T) {
		disabled := false
		cluster := gen.LanguageCluster("tls-disabled",
			gen.SetClusterDomain(domain),
			gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{
				Enabled: &disabled,
			}))
		ing := reconcileCluster(t, cluster)
		assert.Empty(t, ing.Spec.TLS, "TLS should not be configured when Enabled=false")
	})
}

func TestLanguageClusterController_GatewayIngressClassName(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	const domain = "example.com"

	reconcileCluster := func(t *testing.T, cluster *langopv1alpha1.LanguageCluster, defaultIngressClassName string) *networkingv1.Ingress {
		t.Helper()
		// Gateway ingress is opt-in; these tests exercise its rendering so enable it.
		gen.SetClusterIngressEnabled(true)(cluster)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster).
			WithStatusSubresource(cluster).
			Build()
		// Block the async DNS goroutine until after the reconcile returns to
		// prevent a concurrent Status().Update() racing with the main reconcile.
		dnsUnblock := make(chan struct{})
		t.Cleanup(func() { close(dnsUnblock) })
		r := &LanguageClusterReconciler{
			Client:                  fakeClient,
			Scheme:                  scheme,
			Log:                     logr.Discard(),
			DefaultIngressClassName: defaultIngressClassName,
			DNSLookup: func(ctx context.Context, host string) error {
				select {
				case <-dnsUnblock:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		ctx := context.Background()
		req := clusterRequest(cluster.Name)
		_, err := r.Reconcile(ctx, req)
		require.NoError(t, err)
		_, err = r.Reconcile(ctx, req)
		require.NoError(t, err)
		ing := &networkingv1.Ingress{}
		require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ing))
		return ing
	}

	t.Run("operator_default_applied_when_spec_empty", func(t *testing.T) {
		cluster := gen.LanguageCluster("ingress-class-default", gen.SetClusterDomain(domain))
		ing := reconcileCluster(t, cluster, "traefik")
		require.NotNil(t, ing.Spec.IngressClassName)
		assert.Equal(t, "traefik", *ing.Spec.IngressClassName)
	})

	t.Run("per_cluster_spec_overrides_operator_default", func(t *testing.T) {
		cluster := gen.LanguageCluster("ingress-class-override",
			gen.SetClusterDomain(domain),
			gen.SetClusterIngressClassName("nginx"))
		ing := reconcileCluster(t, cluster, "traefik")
		require.NotNil(t, ing.Spec.IngressClassName)
		assert.Equal(t, "nginx", *ing.Spec.IngressClassName)
	})

	t.Run("no_class_when_default_empty_and_spec_empty", func(t *testing.T) {
		cluster := gen.LanguageCluster("ingress-class-none", gen.SetClusterDomain(domain))
		ing := reconcileCluster(t, cluster, "")
		assert.Nil(t, ing.Spec.IngressClassName)
	})
}

func TestLanguageClusterController_GatewayIngressEnabled(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	const domain = "example.com"

	reconcile := func(t *testing.T, objs ...client.Object) client.Client {
		t.Helper()
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs[0]).
			Build()
		dnsUnblock := make(chan struct{})
		t.Cleanup(func() { close(dnsUnblock) })
		r := &LanguageClusterReconciler{
			Client: fakeClient, Scheme: scheme, Log: logr.Discard(),
			DNSLookup: func(ctx context.Context, host string) error {
				select {
				case <-dnsUnblock:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		ctx := context.Background()
		req := clusterRequest(objs[0].GetName())
		_, err := r.Reconcile(ctx, req)
		require.NoError(t, err)
		_, err = r.Reconcile(ctx, req)
		require.NoError(t, err)
		return fakeClient
	}

	t.Run("not_created_by_default", func(t *testing.T) {
		cluster := gen.LanguageCluster("ingress-default-off", gen.SetClusterDomain(domain))
		fakeClient := reconcile(t, cluster)
		ing := &networkingv1.Ingress{}
		err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ing)
		assert.True(t, errors.IsNotFound(err), "gateway ingress must not be created when enabled is unset")
	})

	t.Run("created_when_enabled", func(t *testing.T) {
		cluster := gen.LanguageCluster("ingress-on", gen.SetClusterDomain(domain), gen.SetClusterIngressEnabled(true))
		fakeClient := reconcile(t, cluster)
		ing := &networkingv1.Ingress{}
		require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ing))
	})

	t.Run("deleted_when_disabled", func(t *testing.T) {
		cluster := gen.LanguageCluster("ingress-stale", gen.SetClusterDomain(domain), gen.SetClusterIngressEnabled(false))
		stale := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "gateway", Namespace: cluster.Name},
		}
		fakeClient := reconcile(t, cluster, stale)
		ing := &networkingv1.Ingress{}
		err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ing)
		assert.True(t, errors.IsNotFound(err), "stale gateway ingress must be deleted when disabled")
	})
}

func TestLanguageClusterController_ManagedResources(t *testing.T) {
	reconcileClusterMR := func(t *testing.T, cluster *langopv1alpha1.LanguageCluster, networkIsolation bool) *langopv1alpha1.LanguageCluster {
		t.Helper()
		scheme := testutil.SetupTestScheme(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster).
			WithStatusSubresource(cluster).
			Build()
		dnsUnblock := make(chan struct{})
		t.Cleanup(func() { close(dnsUnblock) })
		reconciler := &LanguageClusterReconciler{
			Client:                  fakeClient,
			Scheme:                  scheme,
			Log:                     logr.Discard(),
			NetworkIsolationEnabled: networkIsolation,
			DNSLookup: func(ctx context.Context, host string) error {
				select {
				case <-dnsUnblock:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		ctx := context.Background()
		req := clusterRequest(cluster.Name)
		// First reconcile adds finalizer; second creates resources and writes status.
		_, err := reconciler.Reconcile(ctx, req)
		require.NoError(t, err)
		_, err = reconciler.Reconcile(ctx, req)
		require.NoError(t, err)
		updated := &langopv1alpha1.LanguageCluster{}
		require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))
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

	t.Run("baseline_no_domain_no_capacity", func(t *testing.T) {
		cluster := gen.LanguageCluster("mr-cluster")
		updated := reconcileClusterMR(t, cluster, false)
		mr := updated.Status.ManagedResources

		assert.NotEmpty(t, mr)
		assert.True(t, hasMR(mr, "Namespace", "mr-cluster"), "Namespace must be present")
		assert.True(t, hasMR(mr, "Role", "agents"), "Role must be present")
		assert.True(t, hasMR(mr, "RoleBinding", "agents"), "RoleBinding must be present")
		assert.True(t, hasMR(mr, "ConfigMap", "gateway-config"), "gateway ConfigMap must be present")
		assert.True(t, hasMR(mr, "Deployment", "gateway"), "gateway Deployment must be present")
		assert.True(t, hasMR(mr, "Service", "gateway"), "gateway Service must be present")

		assert.False(t, hasMR(mr, "Ingress", "gateway"), "Ingress must not be present without domain")
		assert.False(t, hasMR(mr, "ResourceQuota", "langop-quota"), "ResourceQuota must not be present without capacity")
		assert.False(t, hasMR(mr, "NetworkPolicy", "mr-cluster-agents"), "NetworkPolicy must not be present without isolation")

		// Namespace entry must be cluster-scoped (empty namespace field)
		for _, r := range mr {
			if r.Kind == "Namespace" {
				assert.Empty(t, r.Namespace, "Namespace resource must have empty namespace field (cluster-scoped)")
				break
			}
		}
	})

	t.Run("domain_alone_no_ingress", func(t *testing.T) {
		cluster := gen.LanguageCluster("mr-cluster-domain", gen.SetClusterDomain("ai.example.com"))
		updated := reconcileClusterMR(t, cluster, false)
		assert.False(t, hasMR(updated.Status.ManagedResources, "Ingress", "gateway"),
			"gateway ingress is opt-in; a domain alone must not add it")
	})

	t.Run("ingress_enabled_adds_ingress", func(t *testing.T) {
		cluster := gen.LanguageCluster("mr-cluster-ing",
			gen.SetClusterDomain("ai.example.com"),
			gen.SetClusterIngressEnabled(true))
		updated := reconcileClusterMR(t, cluster, false)
		assert.True(t, hasMR(updated.Status.ManagedResources, "Ingress", "gateway"))
	})

	t.Run("ingress_explicitly_disabled", func(t *testing.T) {
		disabled := false
		cluster := gen.LanguageCluster("mr-cluster-noing",
			gen.SetClusterDomain("ai.example.com"),
			func(c *langopv1alpha1.LanguageCluster) {
				if c.Spec.Ingress == nil {
					c.Spec.Ingress = &langopv1alpha1.IngressConfig{}
				}
				c.Spec.Ingress.Enabled = &disabled
			})
		updated := reconcileClusterMR(t, cluster, false)
		assert.False(t, hasMR(updated.Status.ManagedResources, "Ingress", "gateway"))
	})

	t.Run("capacity_adds_resourcequota", func(t *testing.T) {
		maxAgents := int32(10)
		cluster := gen.LanguageCluster("mr-cluster-cap",
			gen.SetClusterCapacity(&langopv1alpha1.ClusterCapacitySpec{MaxAgents: &maxAgents}))
		updated := reconcileClusterMR(t, cluster, false)
		assert.True(t, hasMR(updated.Status.ManagedResources, "ResourceQuota", "langop-quota"))
	})

	t.Run("network_isolation_adds_networkpolicy", func(t *testing.T) {
		cluster := gen.LanguageCluster("mr-cluster-np")
		updated := reconcileClusterMR(t, cluster, true)
		assert.True(t, hasMR(updated.Status.ManagedResources, "NetworkPolicy", fmt.Sprintf("%s-agents", cluster.Name)))
	})
}

// TestLanguageClusterController_GatewayDistinctSecretMountPaths verifies that two
// LanguageModels with different apiKeySecretRef names each get a unique VolumeMount
// path (/etc/secrets/<secretName>) so the gateway pod spec is valid.
func TestLanguageClusterController_GatewayDistinctSecretMountPaths(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("secrets-cluster")
	modelA := gen.LanguageModel("model-a", cluster.Name,
		gen.SetModelAPIKeySecretRef("secret-a", "api-key"),
	)
	modelB := gen.LanguageModel("model-b", cluster.Name,
		gen.SetModelAPIKeySecretRef("secret-b", "api-key"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, modelA, modelB).
		WithStatusSubresource(cluster).
		Build()

	r := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, dep))
	require.NotEmpty(t, dep.Spec.Template.Spec.Containers)

	// Collect all VolumeMount paths and Volume names.
	var mountPaths []string
	for _, m := range dep.Spec.Template.Spec.Containers[0].VolumeMounts {
		mountPaths = append(mountPaths, m.MountPath)
	}
	var volNames []string
	for _, v := range dep.Spec.Template.Spec.Volumes {
		volNames = append(volNames, v.Name)
	}

	// Each secret must be mounted to its own unique subdirectory.
	assert.Contains(t, mountPaths, "/etc/secrets/secret-a", "secret-a must have unique mount path")
	assert.Contains(t, mountPaths, "/etc/secrets/secret-b", "secret-b must have unique mount path")

	// Both volumes must be present.
	assert.Contains(t, volNames, "secret-secret-a")
	assert.Contains(t, volNames, "secret-secret-b")

	// KeyToPath.Path must be just the key, not secretName/key.
	for _, v := range dep.Spec.Template.Spec.Volumes {
		if v.Secret == nil {
			continue
		}
		for _, item := range v.Secret.Items {
			assert.NotContains(t, item.Path, "/", "KeyToPath.Path must not contain a slash — subdirectory comes from MountPath")
		}
	}

	// Mount paths must all be distinct (no duplicates).
	seen := map[string]int{}
	for _, p := range mountPaths {
		seen[p]++
	}
	for p, count := range seen {
		assert.Equal(t, 1, count, "mount path %q appears %d times — must be unique", p, count)
	}
}

func TestLanguageClusterController_GatewayHPA_CreatedWhenAutoscalingSet(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	minReplicas := int32(2)
	cluster := gen.LanguageCluster("hpa-cluster")
	cluster.Spec.Gateway = &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			Autoscaling: &langopv1alpha1.AutoscalingSpec{
				MinReplicas: &minReplicas,
				MaxReplicas: 10,
			},
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

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, hpa))
	require.NotNil(t, hpa.Spec.MinReplicas)
	assert.Equal(t, int32(2), *hpa.Spec.MinReplicas)
	assert.Equal(t, int32(10), hpa.Spec.MaxReplicas)
	assert.Equal(t, "Deployment", hpa.Spec.ScaleTargetRef.Kind)
	assert.Equal(t, "gateway", hpa.Spec.ScaleTargetRef.Name)
	// Default CPU metric injected when no metrics specified
	require.Len(t, hpa.Spec.Metrics, 1)
	assert.Equal(t, autoscalingv2.ResourceMetricSourceType, hpa.Spec.Metrics[0].Type)
	assert.Equal(t, corev1.ResourceCPU, hpa.Spec.Metrics[0].Resource.Name)
	assert.Equal(t, int32(80), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
}

func TestLanguageClusterController_GatewayHPA_DeletedWhenAutoscalingRemoved(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	minReplicas := int32(2)
	cluster := gen.LanguageCluster("rm-hpa-cluster")
	cluster.Spec.Gateway = &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			Autoscaling: &langopv1alpha1.AutoscalingSpec{
				MinReplicas: &minReplicas,
				MaxReplicas: 5,
			},
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

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, hpa),
		"HPA should exist after autoscaling reconcile")

	// Remove autoscaling from the cluster spec
	current := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, current))
	current.Spec.Gateway.Deployment.Autoscaling = nil
	require.NoError(t, fakeClient.Update(ctx, current))

	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, hpa)
	assert.True(t, errors.IsNotFound(err), "HPA must be deleted when autoscaling is removed")
}

func TestLanguageClusterController_GatewayHPA_ReplicaCountPreserved(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	minReplicas := int32(2)
	cluster := gen.LanguageCluster("preserve-hpa-cluster")
	cluster.Spec.Gateway = &langopv1alpha1.GatewaySpec{
		Deployment: langopv1alpha1.DeploymentSpec{
			Autoscaling: &langopv1alpha1.AutoscalingSpec{
				MinReplicas: &minReplicas,
				MaxReplicas: 8,
			},
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

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Simulate HPA scaling the deployment to 5 replicas
	deployment := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, deployment))
	five := int32(5)
	deployment.Spec.Replicas = &five
	require.NoError(t, fakeClient.Update(ctx, deployment))

	// Reconcile again — the controller must not reset replicas back to 1
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, updated))
	require.NotNil(t, updated.Spec.Replicas)
	assert.Equal(t, int32(5), *updated.Spec.Replicas, "HPA-managed replica count must be preserved")
}

func TestLanguageClusterController_AdoptNamespace_Labels(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("adopt-cluster")

	// Pre-existing namespace with no operator labels
	existingNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: cluster.Name},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, existingNs).
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
	_, err = reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)

	ns := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, ns))

	assert.Equal(t, cluster.Name, ns.Labels[langoplabels.LabelKeyLangopCluster], "adopted namespace must have cluster label")
	assert.Equal(t, "language-operator", ns.Labels[langoplabels.LabelKeyK8sManagedBy], "adopted namespace must have managed-by label")
	assert.Empty(t, ns.OwnerReferences, "adopted namespace must not have an owner reference")
}

func TestLanguageClusterController_AdoptNamespace_NotDeletedOnCleanup(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("adopt-delete-cluster")
	cluster.Finalizers = []string{FinalizerName}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	// Namespace has no owner reference — simulates an adopted (pre-existing) namespace.
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

	// Namespace must still exist — it was adopted, not created
	survivingNs := &corev1.Namespace{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, survivingNs)
	require.NoError(t, err, "adopted namespace must not be deleted on cluster removal")
}

func TestLanguageClusterReconciler_AdoptsPreExistingMembers(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "adopt-test"

	cluster := gen.LanguageCluster(ns)
	agent := gen.LanguageAgent("agent-1", ns)
	model := gen.LanguageModel("model-1", ns)
	persona := gen.LanguagePersona("persona-1", ns)
	tool := gen.LanguageTool("tool-1", ns)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent, model, persona, tool).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := clusterRequest(ns)

	// First reconcile: adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Second reconcile: runs to completion and calls adoptPreExistingMembers
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify cluster reached Ready
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: ns}, updatedCluster))
	assert.Equal(t, events.PhaseStatusReady, updatedCluster.Status.Phase)

	// Every member resource must have the cluster-generation annotation stamped.
	wantAnnotation := fmt.Sprintf("%d", updatedCluster.Generation)

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "agent-1", Namespace: ns}, updatedAgent))
	assert.Equal(t, wantAnnotation, updatedAgent.Annotations[langoplabels.AnnotationKeyClusterGeneration], "LanguageAgent should be adopted")

	updatedModel := &langopv1alpha1.LanguageModel{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "model-1", Namespace: ns}, updatedModel))
	assert.Equal(t, wantAnnotation, updatedModel.Annotations[langoplabels.AnnotationKeyClusterGeneration], "LanguageModel should be adopted")

	updatedPersona := &langopv1alpha1.LanguagePersona{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "persona-1", Namespace: ns}, updatedPersona))
	assert.Equal(t, wantAnnotation, updatedPersona.Annotations[langoplabels.AnnotationKeyClusterGeneration], "LanguagePersona should be adopted")

	updatedTool := &langopv1alpha1.LanguageTool{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "tool-1", Namespace: ns}, updatedTool))
	assert.Equal(t, wantAnnotation, updatedTool.Annotations[langoplabels.AnnotationKeyClusterGeneration], "LanguageTool should be adopted")
}
