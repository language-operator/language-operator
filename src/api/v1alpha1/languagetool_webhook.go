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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-langop-io-v1alpha1-languagetool,mutating=true,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagetools,verbs=create;update,versions=v1alpha1,name=mlanguagetool.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagetool,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagetools,verbs=create;update,versions=v1alpha1,name=vlanguagetool.kb.io,admissionReviewVersions=v1

// LanguageToolWebhook handles defaulting and validation for LanguageTool.
//
// +kubebuilder:object:generate=false
type LanguageToolWebhook struct {
	client.Client
}

var _ webhook.CustomDefaulter = &LanguageToolWebhook{}
var _ webhook.CustomValidator = &LanguageToolWebhook{}

// Default implements webhook.CustomDefaulter
func (h *LanguageToolWebhook) Default(ctx context.Context, obj runtime.Object) error {
	t := obj.(*LanguageTool)

	if t.Spec.Deployment.Resources.Requests == nil && t.Spec.Deployment.Resources.Limits == nil {
		t.Spec.Deployment.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator
func (h *LanguageToolWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	t := obj.(*LanguageTool)
	if err := h.validateClusterMembership(ctx, t.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator
func (h *LanguageToolWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	t := obj.(*LanguageTool)
	if err := h.validateClusterMembership(ctx, t.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator
func (h *LanguageToolWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (h *LanguageToolWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	return validateClusterMembership(ctx, h.Client, namespace)
}

// SetupLanguageToolWebhookWithManager registers the LanguageTool mutating and validating webhooks.
func SetupLanguageToolWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageToolWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguageTool{}).
		WithDefaulter(h).
		WithValidator(h).
		Complete()
}
