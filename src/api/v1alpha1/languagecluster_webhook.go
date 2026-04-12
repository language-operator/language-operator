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
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/util/validation"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagecluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageclusters,verbs=create;update,versions=v1alpha1,name=vlanguagecluster.kb.io,admissionReviewVersions=v1

// LanguageClusterWebhook handles validation for LanguageCluster.
// LanguageCluster is cluster-scoped, so no cluster-membership check is needed.
//
// +kubebuilder:object:generate=false
type LanguageClusterWebhook struct{}

var _ admission.Validator[*LanguageCluster] = &LanguageClusterWebhook{}

// ValidateCreate implements admission.Validator
func (h *LanguageClusterWebhook) ValidateCreate(_ context.Context, lc *LanguageCluster) (admission.Warnings, error) {
	return nil, lc.validateSpec()
}

// ValidateUpdate implements admission.Validator
func (h *LanguageClusterWebhook) ValidateUpdate(_ context.Context, _, lc *LanguageCluster) (admission.Warnings, error) {
	if lc.DeletionTimestamp != nil {
		return nil, nil
	}
	return nil, lc.validateSpec()
}

// ValidateDelete implements admission.Validator
func (h *LanguageClusterWebhook) ValidateDelete(_ context.Context, _ *LanguageCluster) (admission.Warnings, error) {
	return nil, nil
}

// validateSpec performs pure spec validation (no API calls).
func (lc *LanguageCluster) validateSpec() error {
	if lc.Spec.Domain != "" {
		if errs := validation.IsDNS1123Subdomain(lc.Spec.Domain); len(errs) > 0 {
			return fmt.Errorf("spec.domain: %q is not a valid DNS subdomain: %s", lc.Spec.Domain, strings.Join(errs, "; "))
		}
	}
	return nil
}

// SetupLanguageClusterWebhookWithManager registers the LanguageCluster validating webhook.
func SetupLanguageClusterWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageClusterWebhook{}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageCluster{}).
		WithValidator(h).
		Complete()
}
