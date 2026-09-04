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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

	// Verify per-agent ServiceAccount created with expected name.
	saName := GenerateServiceAccountName(agent.Name)
	sa := &corev1.ServiceAccount{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: agent.Namespace}, sa); err != nil {
		t.Fatalf("Expected ServiceAccount %q to exist in namespace %s: %v", saName, agent.Namespace, err)
	}

	// Verify namespace-scoped Role created with default rules.
	role := &rbacv1.Role{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: agent.Namespace}, role); err != nil {
		t.Fatalf("Expected Role %q to exist in namespace %s: %v", saName, agent.Namespace, err)
	}
	if len(role.Rules) == 0 {
		t.Errorf("Expected Role to have at least one rule")
	}

	// Verify namespace-scoped RoleBinding created and points to the Role (not the operator ClusterRole).
	rb := &rbacv1.RoleBinding{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: agent.Namespace}, rb); err != nil {
		t.Fatalf("Expected RoleBinding %q to exist in namespace %s: %v", saName, agent.Namespace, err)
	}
	if rb.RoleRef.Kind != "Role" {
		t.Errorf("Expected RoleBinding RoleRef.Kind 'Role', got %q", rb.RoleRef.Kind)
	}
	if rb.RoleRef.Name != saName {
		t.Errorf("Expected RoleBinding RoleRef.Name %q, got %q", saName, rb.RoleRef.Name)
	}
}

func TestLanguageAgentController_CustomServiceAccount(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("custom-sa-agent", "default",
		gen.SetAgentServiceAccountName("custom-sa"))

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

	// First reconcile adds finalizer.
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Second reconcile creates resources.
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// No operator-managed RBAC resources should be created when a custom SA is specified.
	defaultSAName := GenerateServiceAccountName(agent.Name)
	err = fakeClient.Get(ctx, types.NamespacedName{Name: defaultSAName, Namespace: agent.Namespace}, &corev1.ServiceAccount{})
	assert.True(t, errors.IsNotFound(err), "expected no ServiceAccount %q, got: %v", defaultSAName, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: defaultSAName, Namespace: agent.Namespace}, &rbacv1.Role{})
	assert.True(t, errors.IsNotFound(err), "expected no Role %q, got: %v", defaultSAName, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: defaultSAName, Namespace: agent.Namespace}, &rbacv1.RoleBinding{})
	assert.True(t, errors.IsNotFound(err), "expected no RoleBinding %q, got: %v", defaultSAName, err)

	// The Workflow pods must run as the custom service account.
	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
	assert.Equal(t, "custom-sa", podSpec.ServiceAccountName)
}

func TestLanguageAgentController_ServiceAccountAnnotations(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "annotated-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				ServiceAccountAnnotations: map[string]string{
					"eks.amazonaws.com/role-arn": "arn:aws:iam::123456789012:role/my-role",
					"iam.gke.io/service-account": "my-gsa@my-project.iam.gserviceaccount.com",
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
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	saName := GenerateServiceAccountName(agent.Name)
	sa := &corev1.ServiceAccount{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: agent.Namespace}, sa))

	assert.Equal(t, "arn:aws:iam::123456789012:role/my-role", sa.Annotations["eks.amazonaws.com/role-arn"])
	assert.Equal(t, "my-gsa@my-project.iam.gserviceaccount.com", sa.Annotations["iam.gke.io/service-account"])
}

func TestLanguageAgentController_RoleRules(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-rules-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				RoleRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"get"},
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
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	saName := GenerateServiceAccountName(agent.Name)
	role := &rbacv1.Role{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: agent.Namespace}, role))

	// Default rules (configmaps, pods) must still be present.
	var hasConfigMaps, hasPods, hasSecrets bool
	for _, rule := range role.Rules {
		for _, res := range rule.Resources {
			switch res {
			case "configmaps":
				hasConfigMaps = true
			case "pods":
				hasPods = true
			case "secrets":
				hasSecrets = true
			}
		}
	}
	assert.True(t, hasConfigMaps, "expected default configmaps rule in Role")
	assert.True(t, hasPods, "expected default pods rule in Role")
	assert.True(t, hasSecrets, "expected custom secrets rule to be appended to Role")
}
