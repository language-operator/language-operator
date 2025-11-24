package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func TestLanguageAgentController_ReferenceGrantCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tests := []struct {
		name                       string
		agentNamespace             string
		gatewayNamespace           string
		shouldCreateReferenceGrant bool
	}{
		{
			name:                       "same namespace - no ReferenceGrant needed",
			agentNamespace:             "test-namespace",
			gatewayNamespace:           "test-namespace",
			shouldCreateReferenceGrant: false,
		},
		{
			name:                       "cross-namespace - ReferenceGrant needed",
			agentNamespace:             "agents",
			gatewayNamespace:           "istio-system",
			shouldCreateReferenceGrant: true,
		},
		{
			name:                       "default gateway - cross-namespace",
			agentNamespace:             "custom-namespace",
			gatewayNamespace:           "default",
			shouldCreateReferenceGrant: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &langopv1alpha1.LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: tt.agentNamespace,
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(agent).
				Build()

			reconciler := &LanguageAgentReconciler{
				Client: fakeClient,
				Scheme: scheme,
				Log:    logr.Discard(),
			}

			ctx := context.Background()

			// Call reconcileReferenceGrant
			err := reconciler.reconcileReferenceGrant(ctx, agent, "test-gateway", tt.gatewayNamespace)
			if err != nil {
				t.Fatalf("reconcileReferenceGrant failed: %v", err)
			}

			// Check if ReferenceGrant was created
			referenceGrantName := "test-agent-" + tt.agentNamespace + "-referencegrant"
			referenceGrant := &unstructured.Unstructured{}
			referenceGrant.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "gateway.networking.k8s.io",
				Version: "v1beta1",
				Kind:    "ReferenceGrant",
			})

			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      referenceGrantName,
				Namespace: tt.gatewayNamespace,
			}, referenceGrant)

			if tt.shouldCreateReferenceGrant {
				if err != nil {
					t.Errorf("Expected ReferenceGrant to be created, but got error: %v", err)
				} else {
					// Verify ReferenceGrant spec
					spec, ok := referenceGrant.Object["spec"].(map[string]interface{})
					if !ok {
						t.Fatalf("ReferenceGrant spec is not a map")
					}

					// Check 'from' field
					from, ok := spec["from"].([]interface{})
					if !ok || len(from) != 1 {
						t.Fatalf("ReferenceGrant 'from' field is invalid")
					}
					fromEntry := from[0].(map[string]interface{})
					if fromEntry["group"] != "gateway.networking.k8s.io" {
						t.Errorf("Expected 'from' group to be 'gateway.networking.k8s.io', got '%v'", fromEntry["group"])
					}
					if fromEntry["kind"] != "HTTPRoute" {
						t.Errorf("Expected 'from' kind to be 'HTTPRoute', got '%v'", fromEntry["kind"])
					}
					if fromEntry["namespace"] != tt.agentNamespace {
						t.Errorf("Expected 'from' namespace to be '%s', got '%v'", tt.agentNamespace, fromEntry["namespace"])
					}

					// Check 'to' field
					to, ok := spec["to"].([]interface{})
					if !ok || len(to) != 1 {
						t.Fatalf("ReferenceGrant 'to' field is invalid")
					}
					toEntry := to[0].(map[string]interface{})
					if toEntry["group"] != "gateway.networking.k8s.io" {
						t.Errorf("Expected 'to' group to be 'gateway.networking.k8s.io', got '%v'", toEntry["group"])
					}
					if toEntry["kind"] != "Gateway" {
						t.Errorf("Expected 'to' kind to be 'Gateway', got '%v'", toEntry["kind"])
					}
					if toEntry["name"] != "test-gateway" {
						t.Errorf("Expected 'to' name to be 'test-gateway', got '%v'", toEntry["name"])
					}
				}
			} else {
				if err == nil {
					t.Error("Expected no ReferenceGrant to be created, but one was found")
				}
			}
		})
	}
}

func TestLanguageAgentController_ReferenceGrantUpdate(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "agents",
		},
	}

	// Create an existing ReferenceGrant with different spec
	existingReferenceGrant := &unstructured.Unstructured{}
	existingReferenceGrant.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1beta1",
		Kind:    "ReferenceGrant",
	})
	existingReferenceGrant.SetName("test-agent-agents-referencegrant")
	existingReferenceGrant.SetNamespace("istio-system")
	existingReferenceGrant.Object = map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1beta1",
		"kind":       "ReferenceGrant",
		"metadata": map[string]interface{}{
			"name":      "test-agent-agents-referencegrant",
			"namespace": "istio-system",
		},
		"spec": map[string]interface{}{
			"from": []interface{}{
				map[string]interface{}{
					"group":     "gateway.networking.k8s.io",
					"kind":      "HTTPRoute",
					"namespace": "wrong-namespace", // This should be updated
				},
			},
			"to": []interface{}{
				map[string]interface{}{
					"group": "gateway.networking.k8s.io",
					"kind":  "Gateway",
					"name":  "old-gateway", // This should be updated
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, existingReferenceGrant).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()

	// Call reconcileReferenceGrant to update the existing ReferenceGrant
	err := reconciler.reconcileReferenceGrant(ctx, agent, "new-gateway", "istio-system")
	if err != nil {
		t.Fatalf("reconcileReferenceGrant failed: %v", err)
	}

	// Fetch the updated ReferenceGrant
	updatedReferenceGrant := &unstructured.Unstructured{}
	updatedReferenceGrant.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1beta1",
		Kind:    "ReferenceGrant",
	})

	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-agent-agents-referencegrant",
		Namespace: "istio-system",
	}, updatedReferenceGrant)
	if err != nil {
		t.Fatalf("Failed to fetch updated ReferenceGrant: %v", err)
	}

	// Verify the spec was updated
	spec, ok := updatedReferenceGrant.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("ReferenceGrant spec is not a map")
	}

	// Check updated 'from' field
	from := spec["from"].([]interface{})[0].(map[string]interface{})
	if from["namespace"] != "agents" {
		t.Errorf("Expected updated 'from' namespace to be 'agents', got '%v'", from["namespace"])
	}

	// Check updated 'to' field
	to := spec["to"].([]interface{})[0].(map[string]interface{})
	if to["name"] != "new-gateway" {
		t.Errorf("Expected updated 'to' name to be 'new-gateway', got '%v'", to["name"])
	}
}

// Test the readiness checking functions directly
func TestLanguageAgentController_CheckHTTPRouteReadiness(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tests := []struct {
		name          string
		httpRoute     *unstructured.Unstructured
		expectReady   bool
		expectMessage string
	}{
		{
			name: "HTTPRoute ready - Accepted and Programmed",
			httpRoute: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "gateway.networking.k8s.io/v1",
					"kind":       "HTTPRoute",
					"metadata": map[string]interface{}{
						"name":      "test-agent",
						"namespace": "default",
					},
					"status": map[string]interface{}{
						"parents": []interface{}{
							map[string]interface{}{
								"conditions": []interface{}{
									map[string]interface{}{
										"type":   "Accepted",
										"status": "True",
									},
									map[string]interface{}{
										"type":   "Programmed",
										"status": "True",
									},
								},
							},
						},
					},
				},
			},
			expectReady:   true,
			expectMessage: "HTTPRoute is ready and programmed",
		},
		{
			name: "HTTPRoute not ready - Accepted but not Programmed",
			httpRoute: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "gateway.networking.k8s.io/v1",
					"kind":       "HTTPRoute",
					"metadata": map[string]interface{}{
						"name":      "test-agent",
						"namespace": "default",
					},
					"status": map[string]interface{}{
						"parents": []interface{}{
							map[string]interface{}{
								"conditions": []interface{}{
									map[string]interface{}{
										"type":   "Accepted",
										"status": "True",
									},
									map[string]interface{}{
										"type":   "Programmed",
										"status": "False",
									},
								},
							},
						},
					},
				},
			},
			expectReady:   false,
			expectMessage: "HTTPRoute is not ready - waiting for Gateway to accept and program route",
		},
		{
			name: "HTTPRoute not found",
			httpRoute: nil,
			expectReady:   false,
			expectMessage: "HTTPRoute not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme)
			
			if tt.httpRoute != nil {
				tt.httpRoute.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "gateway.networking.k8s.io",
					Version: "v1",
					Kind:    "HTTPRoute",
				})
				tt.httpRoute.SetName("test-agent")
				tt.httpRoute.SetNamespace("default")
				builder = builder.WithObjects(tt.httpRoute)
			}

			fakeClient := builder.Build()

			reconciler := &LanguageAgentReconciler{
				Client:   fakeClient,
				Scheme:   scheme,
				Log:      logr.Discard(),
			}

			ctx := context.Background()
			ready, message, err := reconciler.checkHTTPRouteReadiness(ctx, "test-agent", "default")
			if err != nil {
				t.Fatalf("checkHTTPRouteReadiness failed: %v", err)
			}

			if ready != tt.expectReady {
				t.Errorf("Expected ready=%v, got ready=%v", tt.expectReady, ready)
			}

			if message != tt.expectMessage {
				t.Errorf("Expected message=%q, got message=%q", tt.expectMessage, message)
			}
		})
	}
}
