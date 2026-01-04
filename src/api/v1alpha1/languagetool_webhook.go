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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-langop-io-v1alpha1-languagetool,mutating=true,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagetools,verbs=create;update,versions=v1alpha1,name=mlanguagetool.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagetool,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languagetools,verbs=create;update,versions=v1alpha1,name=vlanguagetool.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &LanguageTool{}
var _ webhook.Validator = &LanguageTool{}

// Default implements webhook.Defaulter
func (t *LanguageTool) Default() {
	// Default resources if not specified
	if t.Spec.Resources.Requests == nil && t.Spec.Resources.Limits == nil {
		t.Spec.Resources = corev1.ResourceRequirements{
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
}

// ValidateCreate implements webhook.Validator
func (t *LanguageTool) ValidateCreate() (admission.Warnings, error) {
	return t.validateTool()
}

// ValidateUpdate implements webhook.Validator
func (t *LanguageTool) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return t.validateTool()
}

// ValidateDelete implements webhook.Validator
func (t *LanguageTool) ValidateDelete() (admission.Warnings, error) {
	// No validation needed for delete
	return nil, nil
}

// validateTool performs common validation for LanguageTool
func (t *LanguageTool) validateTool() (admission.Warnings, error) {
	var warnings admission.Warnings

	// Basic spec validation is handled by kubebuilder annotations
	// Add any custom validation logic here if needed

	return warnings, nil
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (t *LanguageTool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(t).
		Complete()
}
