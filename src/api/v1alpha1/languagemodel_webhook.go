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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-langop-io-v1alpha1-languagemodel,mutating=true,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagemodels,verbs=create;update,versions=v1alpha1,name=mlanguagemodel.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagemodel,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagemodels,verbs=create;update,versions=v1alpha1,name=vlanguagemodel.kb.io,admissionReviewVersions=v1

// LanguageModelWebhook handles defaulting and validation for LanguageModel.
//
// +kubebuilder:object:generate=false
type LanguageModelWebhook struct {
	client.Client
}

var _ webhook.CustomDefaulter = &LanguageModelWebhook{}
var _ webhook.CustomValidator = &LanguageModelWebhook{}

// Default implements webhook.CustomDefaulter
func (h *LanguageModelWebhook) Default(_ context.Context, obj runtime.Object) error {
	m := obj.(*LanguageModel)
	if m.Spec.ClusterRef == "" {
		m.Spec.ClusterRef = m.Namespace
	}
	return nil
}

// ValidateCreate implements webhook.CustomValidator
func (h *LanguageModelWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	m := obj.(*LanguageModel)
	if err := h.validateClusterMembership(ctx, m.Spec.ClusterRef, m.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator
func (h *LanguageModelWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	m := obj.(*LanguageModel)
	if err := h.validateClusterMembership(ctx, m.Spec.ClusterRef, m.Namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator
func (h *LanguageModelWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (h *LanguageModelWebhook) validateClusterMembership(ctx context.Context, clusterRef, namespace string) error {
	cluster := &LanguageCluster{}
	if err := h.Get(ctx, types.NamespacedName{Name: clusterRef}, cluster); err != nil {
		return fmt.Errorf("namespace %q is not managed by a LanguageCluster: no cluster %q exists", namespace, clusterRef)
	}
	return nil
}

// SetupLanguageModelWebhookWithManager registers the LanguageModel mutating and validating webhooks.
func SetupLanguageModelWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageModelWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguageModel{}).
		WithDefaulter(h).
		WithValidator(h).
		Complete()
}
