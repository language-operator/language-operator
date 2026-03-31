/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-langop-io-v1alpha1-languageagent,mutating=true,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagents,verbs=create;update,versions=v1alpha1,name=mlanguageagent.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languageagent,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagents,verbs=create;update,versions=v1alpha1,name=vlanguageagent.kb.io,admissionReviewVersions=v1

// LanguageAgentWebhook handles defaulting and validation for LanguageAgent.
// It holds a client so it can verify LanguageCluster membership at admission time.
//
// +kubebuilder:object:generate=false
type LanguageAgentWebhook struct {
	client.Client
}

var _ webhook.CustomDefaulter = &LanguageAgentWebhook{}
var _ webhook.CustomValidator = &LanguageAgentWebhook{}

// Default implements webhook.CustomDefaulter
func (h *LanguageAgentWebhook) Default(ctx context.Context, obj runtime.Object) error {
	a := obj.(*LanguageAgent)

	// Default workspace
	if a.Spec.Workspace == nil {
		enabled := true
		a.Spec.Workspace = &WorkspaceSpec{
			Enabled:    &enabled,
			Size:       "10Gi",
			AccessMode: "ReadWriteOnce",
			MountPath:  "/workspace",
		}
	}

	// Default port to 8080
	if a.Spec.Port == nil {
		port := int32(8080)
		a.Spec.Port = &port
	}

	// Default resources
	if a.Spec.Deployment.Resources.Requests == nil && a.Spec.Deployment.Resources.Limits == nil {
		a.Spec.Deployment.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		}
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator
func (h *LanguageAgentWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	a := obj.(*LanguageAgent)
	if err := h.validateClusterMembership(ctx, a.Namespace); err != nil {
		return nil, err
	}
	return nil, a.validateSpec()
}

// ValidateUpdate implements webhook.CustomValidator
func (h *LanguageAgentWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	a := obj.(*LanguageAgent)
	if err := h.validateClusterMembership(ctx, a.Namespace); err != nil {
		return nil, err
	}
	return nil, a.validateSpec()
}

// ValidateDelete implements webhook.CustomValidator
func (h *LanguageAgentWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validateClusterMembership verifies a LanguageCluster exists for this namespace.
func (h *LanguageAgentWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	cluster := &LanguageCluster{}
	err := h.Get(ctx, types.NamespacedName{Name: namespace}, cluster)
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("namespace %q is not managed by a LanguageCluster: no cluster %q exists", namespace, namespace)
	}
	return fmt.Errorf("failed to check LanguageCluster for namespace %q: %w", namespace, err)
}

// validateSpec performs pure spec validation (no API calls)
func (a *LanguageAgent) validateSpec() error {
	if len(a.Spec.Models) > 0 {
		if err := a.validateModelReferences(); err != nil {
			return fmt.Errorf("spec.models: %w", err)
		}
	}

	if a.Spec.Workspace != nil && (a.Spec.Workspace.Enabled == nil || *a.Spec.Workspace.Enabled) {
		if err := validateWorkspaceSize(a.Spec.Workspace.Size); err != nil {
			return fmt.Errorf("spec.workspace.size: %w", err)
		}
	}

	return nil
}

func validateWorkspaceSize(size string) error {
	if size == "" {
		return fmt.Errorf("cannot be empty, PersistentVolumeClaims require explicit storage size (e.g., \"10Gi\", \"1.5Ti\")")
	}
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("invalid format %q, expected Kubernetes quantity format (e.g., \"10Gi\", \"1.5Ti\")", size)
	}
	if quantity.IsZero() {
		return fmt.Errorf("cannot be zero, PersistentVolumeClaims require non-zero storage")
	}
	if quantity.Sign() < 0 {
		return fmt.Errorf("cannot be negative, got: %s", size)
	}
	return nil
}

func (a *LanguageAgent) validateModelReferences() error {
	if len(a.Spec.Models) == 0 {
		return nil
	}
	primaryCount := 0
	for i, modelRef := range a.Spec.Models {
		if modelRef.Name == "" {
			return fmt.Errorf("models[%d].name cannot be empty", i)
		}
		if modelRef.Role == "primary" || modelRef.Role == "" {
			primaryCount++
		}
		if modelRef.Priority != nil && *modelRef.Priority < 0 {
			return fmt.Errorf("models[%d].priority cannot be negative", i)
		}
	}
	if primaryCount == 0 {
		return fmt.Errorf("at least one model must have role 'primary'")
	}
	return nil
}

// SetupWebhookWithManager registers the LanguageAgent mutating and validating webhooks.
func SetupLanguageAgentWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageAgentWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguageAgent{}).
		WithDefaulter(h).
		WithValidator(h).
		Complete()
}
