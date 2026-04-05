package testutil

import (
	"testing"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SetupTestScheme creates a scheme with all required types registered
func SetupTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()

	schemes := []func(*runtime.Scheme) error{
		langopv1alpha1.AddToScheme,
		corev1.AddToScheme,
		appsv1.AddToScheme,
		batchv1.AddToScheme,
		networkingv1.AddToScheme,
		policyv1.AddToScheme,
		autoscalingv2.AddToScheme,
		rbacv1.AddToScheme,
	}

	for _, addScheme := range schemes {
		if err := addScheme(scheme); err != nil {
			t.Fatalf("Failed to add scheme: %v", err)
		}
	}

	return scheme
}
