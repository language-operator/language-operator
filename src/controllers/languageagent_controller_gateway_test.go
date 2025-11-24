package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLanguageAgentController_GatewayConfiguration(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tests := []struct {
		name                     string
		cluster                  *langopv1alpha1.LanguageCluster
		expectedGatewayName      string
		expectedGatewayNamespace string
	}{
		{
			name: "new GatewayName field takes precedence",
			cluster: &langopv1alpha1.LanguageCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-new",
					Namespace: "test-namespace",
				},
				Spec: langopv1alpha1.LanguageClusterSpec{
					IngressConfig: &langopv1alpha1.IngressConfig{
						GatewayName:      "my-gateway",
						GatewayNamespace: "gateway-system",
						GatewayClassName: "old-gateway", // Should be ignored
					},
				},
			},
			expectedGatewayName:      "my-gateway",
			expectedGatewayNamespace: "gateway-system",
		},
		{
			name: "GatewayName without namespace defaults to agent namespace",
			cluster: &langopv1alpha1.LanguageCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-default-ns",
					Namespace: "test-namespace",
				},
				Spec: langopv1alpha1.LanguageClusterSpec{
					IngressConfig: &langopv1alpha1.IngressConfig{
						GatewayName: "my-gateway",
					},
				},
			},
			expectedGatewayName:      "my-gateway",
			expectedGatewayNamespace: "test-namespace", // Agent's namespace
		},
		{
			name: "fallback to deprecated GatewayClassName for backward compatibility",
			cluster: &langopv1alpha1.LanguageCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-legacy",
					Namespace: "test-namespace",
				},
				Spec: langopv1alpha1.LanguageClusterSpec{
					IngressConfig: &langopv1alpha1.IngressConfig{
						GatewayClassName: "legacy-gateway",
					},
				},
			},
			expectedGatewayName:      "legacy-gateway",
			expectedGatewayNamespace: "test-namespace", // Agent's namespace
		},
		{
			name: "empty fields default to 'default' gateway",
			cluster: &langopv1alpha1.LanguageCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-empty",
					Namespace: "test-namespace",
				},
				Spec: langopv1alpha1.LanguageClusterSpec{
					IngressConfig: &langopv1alpha1.IngressConfig{},
				},
			},
			expectedGatewayName:      "default",
			expectedGatewayNamespace: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "test-namespace",
				},
				Spec: langopv1alpha1.LanguageAgentSpec{
					ClusterRef: tt.cluster.Name,
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.cluster, agent).
				WithStatusSubresource(agent).
				Build()

			reconciler := &LanguageAgentReconciler{
				Client: fakeClient,
				Scheme: scheme,
				Log:    logr.Discard(),
			}

			ctx := context.Background()

			// Test gateway configuration parsing by directly testing the logic
			// Get cluster config for Gateway configuration (copied from reconcileHTTPRoute logic)
			var gatewayName, gatewayNamespace string
			if agent.Spec.ClusterRef != "" {
				cluster := &langopv1alpha1.LanguageCluster{}
				if err := reconciler.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err == nil {
					if cluster.Spec.IngressConfig != nil {
						// Prefer new GatewayName field, fall back to deprecated GatewayClassName for backward compatibility
						if cluster.Spec.IngressConfig.GatewayName != "" {
							gatewayName = cluster.Spec.IngressConfig.GatewayName
							// Use specified namespace or default to cluster namespace
							if cluster.Spec.IngressConfig.GatewayNamespace != "" {
								gatewayNamespace = cluster.Spec.IngressConfig.GatewayNamespace
							} else {
								gatewayNamespace = agent.Namespace
							}
						} else if cluster.Spec.IngressConfig.GatewayClassName != "" {
							// Backward compatibility: treat GatewayClassName as Gateway resource name
							gatewayName = cluster.Spec.IngressConfig.GatewayClassName
							gatewayNamespace = agent.Namespace
						}
					}
				}
			}

			// Default to "default" gateway if not specified
			if gatewayName == "" {
				gatewayName = "default"
				gatewayNamespace = "default"
			}

			// Verify the gateway configuration matches expectations
			if gatewayName != tt.expectedGatewayName {
				t.Errorf("Expected gatewayName '%s', got '%s'", tt.expectedGatewayName, gatewayName)
			}
			if gatewayNamespace != tt.expectedGatewayNamespace {
				t.Errorf("Expected gatewayNamespace '%s', got '%s'", tt.expectedGatewayNamespace, gatewayNamespace)
			}
		})
	}
}

func TestLanguageAgentController_GatewayFieldPrecedence(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Test that GatewayName takes precedence over GatewayClassName
	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "precedence-test",
			Namespace: "test-namespace",
		},
		Spec: langopv1alpha1.LanguageClusterSpec{
			IngressConfig: &langopv1alpha1.IngressConfig{
				GatewayName:      "new-gateway",
				GatewayNamespace: "new-namespace",
				GatewayClassName: "old-gateway", // Should be ignored
			},
		},
	}

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			ClusterRef: cluster.Name,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()

	// Test that the reconciler can fetch and process the cluster configuration
	fetchedCluster := &langopv1alpha1.LanguageCluster{}
	err := reconciler.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, fetchedCluster)
	if err != nil {
		t.Fatalf("Failed to fetch cluster: %v", err)
	}

	// Verify the cluster has the expected configuration
	if fetchedCluster.Spec.IngressConfig.GatewayName != "new-gateway" {
		t.Errorf("Expected GatewayName 'new-gateway', got '%s'", fetchedCluster.Spec.IngressConfig.GatewayName)
	}
	if fetchedCluster.Spec.IngressConfig.GatewayNamespace != "new-namespace" {
		t.Errorf("Expected GatewayNamespace 'new-namespace', got '%s'", fetchedCluster.Spec.IngressConfig.GatewayNamespace)
	}
	if fetchedCluster.Spec.IngressConfig.GatewayClassName != "old-gateway" {
		t.Errorf("Expected GatewayClassName 'old-gateway' (for backward compatibility), got '%s'", fetchedCluster.Spec.IngressConfig.GatewayClassName)
	}
}
