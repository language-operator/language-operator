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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagepersona,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagepersonas,verbs=create;update,versions=v1alpha1,name=vlanguagepersona.kb.io,admissionReviewVersions=v1

// LanguagePersonaWebhook handles validation for LanguagePersona.
//
// +kubebuilder:object:generate=false
type LanguagePersonaWebhook struct {
	client.Client
}

var _ webhook.CustomValidator = &LanguagePersonaWebhook{}

// ValidateCreate implements webhook.CustomValidator
func (h *LanguagePersonaWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	p := obj.(*LanguagePersona)
	if err := h.validateClusterMembership(ctx, p.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator
func (h *LanguagePersonaWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	p := obj.(*LanguagePersona)
	if err := h.validateClusterMembership(ctx, p.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator
func (h *LanguagePersonaWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (h *LanguagePersonaWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	return validateClusterMembership(ctx, h.Client, namespace)
}

// SetupLanguagePersonaWebhookWithManager registers the LanguagePersona validating webhook.
func SetupLanguagePersonaWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguagePersonaWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguagePersona{}).
		WithValidator(h).
		Complete()
}
