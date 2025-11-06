package controllers

import (
	"context"
	"testing"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLanguageToolController_SidecarMode(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := langopv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add langop scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add core scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add apps scheme: %v", err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add networking scheme: %v", err)
	}

	tool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sidecar-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Type:           "mcp",
			Image:          "test:latest",
			DeploymentMode: "sidecar",
			Port:           8080,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client: fakeClient,
		Scheme: scheme,
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
	scheme := runtime.NewScheme()
	if err := langopv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add langop scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add core scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add apps scheme: %v", err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add networking scheme: %v", err)
	}

	tool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Type:           "mcp",
			Image:          "test:latest",
			DeploymentMode: "service",
			Port:           8080,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client: fakeClient,
		Scheme: scheme,
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
