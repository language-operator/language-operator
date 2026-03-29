package controllers

import (
	"context"
	"testing"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockRegistryManager is a mock implementation of RegistryManager for testing
type mockRegistryManager struct {
	registries []string
}

func (m *mockRegistryManager) GetRegistries() []string {
	if m.registries == nil {
		return []string{"docker.io", "gcr.io", "ghcr.io"}
	}
	return m.registries
}

func TestLanguageToolController_SidecarMode(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-sidecar-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolDeploymentMode("sidecar"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      tool.Name,
			Namespace: tool.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no deployment was created for sidecar mode
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, deployment)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected deployment to not exist for sidecar mode, but got: %v", err)
	}

	// Verify no service was created for sidecar mode
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, service)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected service to not exist for sidecar mode, but got: %v", err)
	}
}

func TestLanguageToolController_ServiceMode(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-service-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolDeploymentMode("service"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      tool.Name,
			Namespace: tool.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify deployment was created for service mode
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected deployment to exist for service mode, but got error: %v", err)
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != "test:latest" {
		t.Errorf("Expected image 'test:latest', got '%s'", deployment.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify service was created for service mode
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, service)
	if err != nil {
		t.Fatalf("Expected service to exist for service mode, but got error: %v", err)
	}
}

func TestLanguageToolController_StatusPhases(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tests := []struct {
		name              string
		deploymentStatus  *appsv1.DeploymentStatus
		expectedPhase     string
		expectedCondition metav1.ConditionStatus
		expectedReason    string
	}{
		{
			name: "Pending - no replicas yet",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       0,
				AvailableReplicas:   0,
				UpdatedReplicas:     1,
				UnavailableReplicas: 0,
			},
			expectedPhase:     "Pending",
			expectedCondition: metav1.ConditionFalse,
			expectedReason:    "Pending",
		},
		{
			name: "Failed - pods exist but none ready (CrashLoopBackOff)",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       0,
				AvailableReplicas:   0,
				UpdatedReplicas:     1,
				UnavailableReplicas: 1,
			},
			expectedPhase:     "Failed",
			expectedCondition: metav1.ConditionFalse,
			expectedReason:    "PodsNotReady",
		},
		{
			name: "Running - at least one pod ready",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UpdatedReplicas:     1,
				UnavailableReplicas: 0,
			},
			expectedPhase:     "Running",
			expectedCondition: metav1.ConditionTrue,
			expectedReason:    "ReconcileSuccess",
		},
		{
			name: "Updating - not all replicas updated",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UpdatedReplicas:     0,
				UnavailableReplicas: 0,
			},
			expectedPhase:     "Updating",
			expectedCondition: metav1.ConditionFalse,
			expectedReason:    "Updating",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := gen.LanguageTool("test-tool", "default",
				gen.SetToolImage("test:latest"),
				gen.SetToolDeploymentMode("service"),
				gen.SetToolPort(8080),
			)
			tool.Generation = 1

			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tool.Name,
					Namespace: tool.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: func() *int32 { r := int32(1); return &r }(),
				},
				Status: *tt.deploymentStatus,
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), tool, deployment).
				WithStatusSubresource(tool).
				Build()

			reconciler := &LanguageToolReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				RegistryManager: &mockRegistryManager{},
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tool.Name,
					Namespace: tool.Namespace,
				},
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			// Fetch updated tool to check status
			updatedTool := &langopv1alpha1.LanguageTool{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      tool.Name,
				Namespace: tool.Namespace,
			}, updatedTool)
			if err != nil {
				t.Fatalf("Failed to fetch updated tool: %v", err)
			}

			// Verify phase
			if updatedTool.Status.Phase != tt.expectedPhase {
				t.Errorf("Expected phase '%s', got '%s'", tt.expectedPhase, updatedTool.Status.Phase)
			}

			// Verify Ready condition
			var readyCondition *metav1.Condition
			for i := range updatedTool.Status.Conditions {
				if updatedTool.Status.Conditions[i].Type == "Ready" {
					readyCondition = &updatedTool.Status.Conditions[i]
					break
				}
			}
			if readyCondition == nil {
				t.Fatalf("Ready condition not found")
			}
			if readyCondition.Status != tt.expectedCondition {
				t.Errorf("Expected condition status '%s', got '%s'", tt.expectedCondition, readyCondition.Status)
			}
			if readyCondition.Reason != tt.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tt.expectedReason, readyCondition.Reason)
			}

			// Verify replica counts are copied from deployment
			if tt.expectedPhase != "Pending" {
				if updatedTool.Status.ReadyReplicas != tt.deploymentStatus.ReadyReplicas {
					t.Errorf("Expected ReadyReplicas %d, got %d", tt.deploymentStatus.ReadyReplicas, updatedTool.Status.ReadyReplicas)
				}
			}
		})
	}
}

func TestLanguageToolController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-tool",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found tool, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found tool")
	}
}

func TestLanguageToolController_ServiceTypeAndAnnotations(t *testing.T) {
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
			annotations:     map[string]string{"example.com/foo": "bar"},
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			mods := []gen.LanguageToolModifier{
				gen.SetToolDeploymentMode("service"),
				gen.SetToolPort(8080),
			}
			if tt.serviceType != "" {
				mods = append(mods, gen.SetToolServiceType(tt.serviceType))
			}
			if tt.annotations != nil {
				mods = append(mods, gen.SetToolServiceAnnotations(tt.annotations))
			}
			tool := gen.LanguageTool("test-tool", "default", mods...)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), tool).
				WithStatusSubresource(tool).
				Build()

			reconciler := &LanguageToolReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				RegistryManager: &mockRegistryManager{},
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace},
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			svc := &corev1.Service{}
			if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, svc); err != nil {
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
