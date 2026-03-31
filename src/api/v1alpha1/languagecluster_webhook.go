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

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagecluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageclusters,verbs=create;update,versions=v1alpha1,name=vlanguagecluster.kb.io,admissionReviewVersions=v1

// LanguageClusterWebhook handles validation for LanguageCluster.
//
// +kubebuilder:object:generate=false
type LanguageClusterWebhook struct {
	client.Client
}

var _ webhook.CustomValidator = &LanguageClusterWebhook{}

// ValidateCreate implements webhook.CustomValidator
func (h *LanguageClusterWebhook) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator
func (h *LanguageClusterWebhook) ValidateUpdate(_ context.Context, _ runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator
func (h *LanguageClusterWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// SetupLanguageClusterWebhookWithManager registers the LanguageCluster validating webhook.
func SetupLanguageClusterWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageClusterWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguageCluster{}).
		WithValidator(h).
		Complete()
}
