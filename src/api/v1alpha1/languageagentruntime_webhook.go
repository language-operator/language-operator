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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languageagentruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagentruntimes,verbs=create;update,versions=v1alpha1,name=vlanguageagentruntime.kb.io,admissionReviewVersions=v1

// LanguageAgentRuntimeWebhook handles validation for LanguageAgentRuntime.
// LanguageAgentRuntime is cluster-scoped, so no cluster-membership check is needed.
//
// +kubebuilder:object:generate=false
type LanguageAgentRuntimeWebhook struct{}

var _ admission.Validator[*LanguageAgentRuntime] = &LanguageAgentRuntimeWebhook{}

// ValidateCreate implements admission.Validator
func (h *LanguageAgentRuntimeWebhook) ValidateCreate(_ context.Context, rt *LanguageAgentRuntime) (admission.Warnings, error) {
	return nil, rt.validateSpec()
}

// ValidateUpdate implements admission.Validator
func (h *LanguageAgentRuntimeWebhook) ValidateUpdate(_ context.Context, _, rt *LanguageAgentRuntime) (admission.Warnings, error) {
	if rt.DeletionTimestamp != nil {
		return nil, nil
	}
	return nil, rt.validateSpec()
}

// ValidateDelete implements admission.Validator
func (h *LanguageAgentRuntimeWebhook) ValidateDelete(_ context.Context, _ *LanguageAgentRuntime) (admission.Warnings, error) {
	return nil, nil
}

// validateSpec performs pure spec validation (no API calls).
func (rt *LanguageAgentRuntime) validateSpec() error {
	if rt.Spec.Workspace != nil {
		if err := validateWorkspaceSize(rt.Spec.Workspace.Size); err != nil {
			return fmt.Errorf("spec.workspace.size: %w", err)
		}
	}

	if len(rt.Spec.Ports) > 0 {
		if err := validateAgentPorts(rt.Spec.Ports); err != nil {
			return fmt.Errorf("spec.ports: %w", err)
		}
	}

	return nil
}

// SetupLanguageAgentRuntimeWebhookWithManager registers the LanguageAgentRuntime validating webhook.
func SetupLanguageAgentRuntimeWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageAgentRuntimeWebhook{}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageAgentRuntime{}).
		WithValidator(h).
		Complete()
}
