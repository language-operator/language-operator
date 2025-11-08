package controllers

import (
	"context"
	"testing"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupLanguageModelTestScheme(t *testing.T) *runtime.Scheme {
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
	return scheme
}

func TestLanguageModelController_ConfigMapCreation(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	model := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "anthropic",
			ModelName: "claude-3-5-sonnet-20241022",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile creates resources
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify ConfigMap was created
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(model.Name, "model"),
		Namespace: model.Namespace,
	}, cm)
	if err != nil {
		t.Fatalf("Expected ConfigMap to exist, but got error: %v", err)
	}

	// Verify ConfigMap contains model configuration
	if cm.Data["provider"] != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", cm.Data["provider"])
	}
	if cm.Data["modelName"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected modelName 'claude-3-5-sonnet-20241022', got '%s'", cm.Data["modelName"])
	}
	if cm.Data["model.json"] == "" {
		t.Error("Expected model.json to contain serialized spec")
	}
}

func TestLanguageModelController_DeploymentAndServiceCreation(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	model := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model-deployment",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "openai",
			ModelName: "gpt-4",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer, second creates resources
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify Deployment configuration
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Name != "proxy" {
		t.Errorf("Expected container name 'proxy', got '%s'", container.Name)
	}
	if container.Image != "git.theryans.io/language-operator/model:latest" {
		t.Errorf("Expected image 'git.theryans.io/language-operator/model:latest', got '%s'", container.Image)
	}

	// Verify Service was created
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, service)
	if err != nil {
		t.Fatalf("Expected Service to exist, but got error: %v", err)
	}

	// Verify Service configuration
	if len(service.Spec.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(service.Spec.Ports))
	}
	if service.Spec.Ports[0].Port != 8000 {
		t.Errorf("Expected port 8000, got %d", service.Spec.Ports[0].Port)
	}
}

func TestLanguageModelController_StatusUpdates(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	model := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-model-status",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "anthropic",
			ModelName: "claude-3-5-sonnet-20241022",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer, second creates resources
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Fetch updated model to check status
	updatedModel := &langopv1alpha1.LanguageModel{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, updatedModel)
	if err != nil {
		t.Fatalf("Failed to fetch updated model: %v", err)
	}

	// Verify status phase
	if updatedModel.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedModel.Status.Phase)
	}

	// Verify ObservedGeneration
	if updatedModel.Status.ObservedGeneration != model.Generation {
		t.Errorf("Expected ObservedGeneration %d, got %d", model.Generation, updatedModel.Status.ObservedGeneration)
	}

	// Verify Ready condition
	var readyCondition *metav1.Condition
	for i := range updatedModel.Status.Conditions {
		if updatedModel.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedModel.Status.Conditions[i]
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
}

func TestLanguageModelController_APIKeySecretMount(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	model := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model-secret",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "openai",
			ModelName: "gpt-4",
			APIKeySecretRef: &langopv1alpha1.SecretReference{
				Name: "openai-api-key",
				Key:  "api-key",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer, second creates resources
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify Deployment has secret volume
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Check for secrets volume
	foundSecretsVolume := false
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		if vol.Name == "secrets" {
			foundSecretsVolume = true
			if vol.Secret == nil {
				t.Error("Expected secrets volume to use Secret source")
			} else if vol.Secret.SecretName != "openai-api-key" {
				t.Errorf("Expected secret name 'openai-api-key', got '%s'", vol.Secret.SecretName)
			}
			break
		}
	}
	if !foundSecretsVolume {
		t.Error("Expected secrets volume to be mounted")
	}

	// Check for secrets volume mount
	foundSecretsMount := false
	for _, mount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		if mount.Name == "secrets" {
			foundSecretsMount = true
			if mount.MountPath != "/etc/secrets" {
				t.Errorf("Expected mount path '/etc/secrets', got '%s'", mount.MountPath)
			}
			if !mount.ReadOnly {
				t.Error("Expected secrets mount to be read-only")
			}
			break
		}
	}
	if !foundSecretsMount {
		t.Error("Expected secrets volume mount on container")
	}
}

func TestLanguageModelController_NotFoundHandling(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-model",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found model, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found model")
	}
}

func TestLanguageModelController_NetworkPolicyCreation(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	model := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model-netpol",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "anthropic",
			ModelName: "claude-3-5-sonnet-20241022",
			Egress: []langopv1alpha1.NetworkRule{
				{
					Description: "Allow Anthropic API",
					To: &langopv1alpha1.NetworkPeer{
						DNS: []string{"api.anthropic.com"},
					},
					Ports: []langopv1alpha1.NetworkPort{
						{
							Port:     443,
							Protocol: "TCP",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer, second creates resources
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify NetworkPolicy was created
	netpol := &networkingv1.NetworkPolicy{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, netpol)
	if err != nil {
		t.Fatalf("Expected NetworkPolicy to exist, but got error: %v", err)
	}

	// Verify NetworkPolicy has both Ingress and Egress rules
	foundIngress := false
	foundEgress := false
	for _, policyType := range netpol.Spec.PolicyTypes {
		if policyType == networkingv1.PolicyTypeIngress {
			foundIngress = true
		}
		if policyType == networkingv1.PolicyTypeEgress {
			foundEgress = true
		}
	}
	if !foundIngress {
		t.Error("Expected NetworkPolicy to have Ingress policy type")
	}
	if !foundEgress {
		t.Error("Expected NetworkPolicy to have Egress policy type")
	}
}

func TestLanguageModelController_Finalizer(t *testing.T) {
	scheme := setupLanguageModelTestScheme(t)

	model := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model-finalizer",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "openai",
			ModelName: "gpt-4",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	// Should requeue for finalizer
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	// Fetch model to verify finalizer
	updatedModel := &langopv1alpha1.LanguageModel{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, updatedModel)
	if err != nil {
		t.Fatalf("Failed to fetch updated model: %v", err)
	}

	// Verify finalizer was added
	if !HasFinalizer(updatedModel) {
		t.Error("Expected finalizer to be added after first reconcile")
	}
}
