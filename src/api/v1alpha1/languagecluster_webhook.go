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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagecluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageclusters,verbs=create;update,versions=v1alpha1,name=vlanguagecluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &LanguageCluster{}

// ValidateCreate implements webhook.Validator
func (c *LanguageCluster) ValidateCreate() (admission.Warnings, error) {
	return nil, c.validate()
}

// ValidateUpdate implements webhook.Validator
func (c *LanguageCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return nil, c.validate()
}

// ValidateDelete implements webhook.Validator
func (c *LanguageCluster) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (c *LanguageCluster) validate() error {
	// No validation needed for LanguageCluster
	// (LanguageCluster now only manages namespace creation)
	return nil
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (c *LanguageCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}
