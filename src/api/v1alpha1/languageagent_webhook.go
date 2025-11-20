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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-langop-io-v1alpha1-languageagent,mutating=true,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagents,verbs=create;update,versions=v1alpha1,name=mlanguageagent.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languageagent,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagents,verbs=create;update,versions=v1alpha1,name=vlanguageagent.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &LanguageAgent{}
var _ webhook.Validator = &LanguageAgent{}

// Default implements webhook.Defaulter
func (a *LanguageAgent) Default() {
	// Default workspace to enabled if not specified
	if a.Spec.Workspace == nil {
		a.Spec.Workspace = &WorkspaceSpec{
			Enabled:    true,
			Size:       "10Gi",
			AccessMode: "ReadWriteOnce",
			MountPath:  "/workspace",
		}
	}
}

// ValidateCreate implements webhook.Validator
func (a *LanguageAgent) ValidateCreate() (admission.Warnings, error) {
	warnings := admission.Warnings{}

	// Basic validation
	if err := a.validateSpec(); err != nil {
		return warnings, err
	}

	// Note: Cost validation is performed at the controller level
	// (admission webhooks run before cost estimation is possible)

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator
func (a *LanguageAgent) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	warnings := admission.Warnings{}

	// Basic validation
	if err := a.validateSpec(); err != nil {
		return warnings, err
	}

	// Note: Cost validation is performed at the controller level
	// (admission webhooks run before cost estimation is possible)

	return warnings, nil
}

// ValidateDelete implements webhook.Validator
func (a *LanguageAgent) ValidateDelete() (admission.Warnings, error) {
	// No validation needed on delete
	return nil, nil
}

// validateSpec performs basic spec validation
func (a *LanguageAgent) validateSpec() error {
	// Instructions are required
	if a.Spec.Instructions == "" {
		return fmt.Errorf("spec.instructions is required")
	}

	// Validate safety config if present
	if a.Spec.SafetyConfig != nil {
		if a.Spec.SafetyConfig.MaxCostPerExecution != nil && *a.Spec.SafetyConfig.MaxCostPerExecution < 0 {
			return fmt.Errorf("spec.safetyConfig.maxCostPerExecution must be non-negative")
		}
	}

	// Validate rate limits if present
	if a.Spec.RateLimits != nil {
		if a.Spec.RateLimits.RequestsPerMinute != nil && *a.Spec.RateLimits.RequestsPerMinute < 0 {
			return fmt.Errorf("spec.rateLimits.requestsPerMinute must be non-negative")
		}
		if a.Spec.RateLimits.TokensPerMinute != nil && *a.Spec.RateLimits.TokensPerMinute < 0 {
			return fmt.Errorf("spec.rateLimits.tokensPerMinute must be non-negative")
		}
		if a.Spec.RateLimits.ToolCallsPerMinute != nil && *a.Spec.RateLimits.ToolCallsPerMinute < 0 {
			return fmt.Errorf("spec.rateLimits.toolCallsPerMinute must be non-negative")
		}
	}

	return nil
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (a *LanguageAgent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(a).
		Complete()
}
