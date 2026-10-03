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
	"sort"
	"strings"

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
	return h.sharedModelNameWarnings(ctx, m), nil
}

// ValidateUpdate implements admission.Validator
func (h *LanguageModelWebhook) ValidateUpdate(ctx context.Context, _, m *LanguageModel) (admission.Warnings, error) {
	if err := h.validateClusterMembership(ctx, m.Namespace); err != nil {
		return nil, err
	}
	return h.sharedModelNameWarnings(ctx, m), nil
}

// sharedModelNameWarnings warns when another LanguageModel in the cluster has the
// same modelName. Agents call the gateway by modelName, so the gateway serves them
// as one model and load-balances across both. That is how to spread a model over
// several endpoints, so it is allowed; but it also happens by accident, so say so.
// A failed lookup adds no warning rather than blocking the request.
func (h *LanguageModelWebhook) sharedModelNameWarnings(ctx context.Context, m *LanguageModel) admission.Warnings {
	if m.Spec.ModelName == "" {
		return nil
	}
	models := &LanguageModelList{}
	if err := h.listReader().List(ctx, models, client.InNamespace(m.Namespace)); err != nil {
		return nil
	}
	var others []string
	for _, other := range models.Items {
		if other.Name != m.Name && other.Spec.ModelName == m.Spec.ModelName {
			others = append(others, other.Name)
		}
	}
	if len(others) == 0 {
		return nil
	}
	sort.Strings(others)
	return admission.Warnings{fmt.Sprintf(
		"LanguageModel(s) %s already use modelName %q: the gateway load-balances requests for %q across all of them",
		strings.Join(others, ", "), m.Spec.ModelName, m.Spec.ModelName)}
}

func (h *LanguageModelWebhook) listReader() client.Reader {
	if h.reader != nil {
		return h.reader
	}
	return h.Client
}

// ValidateDelete implements admission.Validator
func (h *LanguageModelWebhook) ValidateDelete(_ context.Context, _ *LanguageModel) (admission.Warnings, error) {
	return nil, nil
}

func (h *LanguageModelWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	return validateClusterMembership(ctx, h.listReader(), namespace)
}

// SetupLanguageModelWebhookWithManager registers the LanguageModel validating webhook.
func SetupLanguageModelWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageModelWebhook{Client: mgr.GetClient(), reader: mgr.GetAPIReader()}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageModel{}).
		WithValidator(h).
		Complete()
}
