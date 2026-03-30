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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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
		gen.SetClusterNetworkPolicies([]langopv1alpha1.NetworkRule{
			{
				From: &langopv1alpha1.NetworkPeer{
					Group: "external-readers",
				},
				Ports: []langopv1alpha1.NetworkPort{
					{Port: 8000},
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
	assert.Equal(t, "external-readers", peer.PodSelector.MatchLabels[LabelKeyLangopGroup])
	require.NotEmpty(t, np.Spec.Ingress[0].Ports)
	assert.Equal(t, int32(8000), np.Spec.Ingress[0].Ports[0].Port.IntVal)
}

func TestLanguageClusterController_NetworkPolicy_ServiceRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("svc-cluster",
		gen.SetClusterNetworkPolicies([]langopv1alpha1.NetworkRule{
			{
				To: &langopv1alpha1.NetworkPeer{
					Service: &langopv1alpha1.ServiceReference{
						Name:      "my-service",
						Namespace: "other-ns",
					},
				},
				Ports: []langopv1alpha1.NetworkPort{{Port: 443}},
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
				if v, ok := peer.NamespaceSelector.MatchLabels[LabelKeyMetadataName]; ok && v == "other-ns" {
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
		gen.SetClusterNetworkPolicies([]langopv1alpha1.NetworkRule{
			{
				To: &langopv1alpha1.NetworkPeer{
					DNS: []string{"api.example.com"},
				},
				Ports: []langopv1alpha1.NetworkPort{{Port: 443}},
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
			wantImage:    "ghcr.io/language-operator/model:latest",
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
	cluster := gen.LanguageCluster("ingress-cluster", gen.SetClusterDomain("example.com"))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cluster).WithStatusSubresource(cluster).Build()
	reconciler := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	ctx := context.Background()
	req := clusterRequest(cluster.Name)
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	ingress := &networkingv1.Ingress{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: "gateway", Namespace: cluster.Name}, ingress))
	require.NotEmpty(t, ingress.Spec.Rules)
	assert.Equal(t, "gateway.example.com", ingress.Spec.Rules[0].Host)

	updated := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updated))
	require.NotNil(t, updated.Status.GatewayReady)
	assert.True(t, *updated.Status.GatewayReady)
}

func TestLanguageClusterController_GatewayIngressError(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("ingress-err-cluster", gen.SetClusterDomain("example.com"))

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
	reconciler := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
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
		if updated.Status.Conditions[i].Type == "GatewayReady" {
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
			condType:     "Ready",
			condStatus:   metav1.ConditionFalse,
			condReason:   "NamespaceError",
		},
		{
			name:         "RBACError",
			buildCluster: func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("rbac-err-cluster") },
			failCreate:   func(obj client.Object) bool { _, ok := obj.(*rbacv1.Role); return ok },
			condType:     "Ready",
			condStatus:   metav1.ConditionFalse,
			condReason:   "RBACError",
		},
		{
			name:             "NetworkPolicyError",
			buildCluster:     func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("netpol-err-cluster") },
			failCreate:       func(obj client.Object) bool { _, ok := obj.(*networkingv1.NetworkPolicy); return ok },
			networkIsolation: true,
			condType:         "Ready",
			condStatus:       metav1.ConditionFalse,
			condReason:       "NetworkPolicyError",
		},
		{
			name:         "GatewayError",
			buildCluster: func() *langopv1alpha1.LanguageCluster { return gen.LanguageCluster("gw-err-cluster") },
			failCreate:   func(obj client.Object) bool { _, ok := obj.(*corev1.ConfigMap); return ok },
			condType:     "GatewayReady",
			condStatus:   metav1.ConditionFalse,
			condReason:   "GatewayError",
		},
		{
			name: "CapacityError",
			buildCluster: func() *langopv1alpha1.LanguageCluster {
				return gen.LanguageCluster("cap-err-cluster", gen.SetClusterCapacity(&langopv1alpha1.ClusterCapacitySpec{MaxAgents: &maxAgents}))
			},
			failCreate: func(obj client.Object) bool { _, ok := obj.(*corev1.ResourceQuota); return ok },
			condType:   "CapacityReady",
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
	assert.NotEmpty(t, dep.Spec.Template.Annotations[LabelKeyLangopConfigHash])
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
