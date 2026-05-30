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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagemodel,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagemodels,verbs=create;update,versions=v1alpha1,name=vlanguagemodel.kb.io,admissionReviewVersions=v1

// LanguageModelWebhook handles validation for LanguageModel.
//
// +kubebuilder:object:generate=false
type LanguageModelWebhook struct {
	client.Client
	reader client.Reader
}

var _ admission.Validator[*LanguageModel] = &LanguageModelWebhook{}

// ValidateCreate implements admission.Validator
func (h *LanguageModelWebhook) ValidateCreate(ctx context.Context, m *LanguageModel) (admission.Warnings, error) {
	if err := h.validateClusterMembership(ctx, m.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements admission.Validator
func (h *LanguageModelWebhook) ValidateUpdate(ctx context.Context, _, m *LanguageModel) (admission.Warnings, error) {
	if err := h.validateClusterMembership(ctx, m.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements admission.Validator
func (h *LanguageModelWebhook) ValidateDelete(_ context.Context, _ *LanguageModel) (admission.Warnings, error) {
	return nil, nil
}

func (h *LanguageModelWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	r := client.Reader(h.reader)
	if r == nil {
		r = h.Client
	}
	return validateClusterMembership(ctx, r, namespace)
}

// SetupLanguageModelWebhookWithManager registers the LanguageModel validating webhook.
func SetupLanguageModelWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageModelWebhook{Client: mgr.GetClient(), reader: mgr.GetAPIReader()}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageModel{}).
		WithValidator(h).
		Complete()
}
