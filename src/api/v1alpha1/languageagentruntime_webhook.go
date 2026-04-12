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

	"github.com/language-operator/language-operator/pkg/validation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languageagentruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagentruntimes,verbs=create;update,versions=v1alpha1,name=vlanguageagentruntime.kb.io,admissionReviewVersions=v1

// LanguageAgentRuntimeWebhook handles validation for LanguageAgentRuntime.
// LanguageAgentRuntime is cluster-scoped, so no cluster-membership check is needed.
//
// +kubebuilder:object:generate=false
type LanguageAgentRuntimeWebhook struct {
	RegistryManager validation.RegistryManager
}

var _ admission.Validator[*LanguageAgentRuntime] = &LanguageAgentRuntimeWebhook{}

// ValidateCreate implements admission.Validator
func (h *LanguageAgentRuntimeWebhook) ValidateCreate(_ context.Context, rt *LanguageAgentRuntime) (admission.Warnings, error) {
	return nil, h.validateSpec(rt)
}

// ValidateUpdate implements admission.Validator
func (h *LanguageAgentRuntimeWebhook) ValidateUpdate(_ context.Context, _, rt *LanguageAgentRuntime) (admission.Warnings, error) {
	if rt.DeletionTimestamp != nil {
		return nil, nil
	}
	return nil, h.validateSpec(rt)
}

// ValidateDelete implements admission.Validator
func (h *LanguageAgentRuntimeWebhook) ValidateDelete(_ context.Context, _ *LanguageAgentRuntime) (admission.Warnings, error) {
	return nil, nil
}

// validateSpec validates the LanguageAgentRuntime spec fields.
func (h *LanguageAgentRuntimeWebhook) validateSpec(rt *LanguageAgentRuntime) error {
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

	if rt.Spec.Image != "" && h.RegistryManager != nil {
		if regs := h.RegistryManager.GetRegistries(); len(regs) > 0 {
			if err := validation.ValidateImageRegistry(rt.Spec.Image, regs); err != nil {
				return fmt.Errorf("spec.image: %w", err)
			}
		}
	}

	if err := validateResourceList("spec.deployment.resources.requests", rt.Spec.Deployment.Resources.Requests); err != nil {
		return err
	}
	if err := validateResourceList("spec.deployment.resources.limits", rt.Spec.Deployment.Resources.Limits); err != nil {
		return err
	}

	return nil
}

// validateResourceList checks that each quantity in the list is positive.
func validateResourceList(field string, list corev1.ResourceList) error {
	zero := resource.MustParse("0")
	for name, qty := range list {
		if qty.Cmp(zero) <= 0 {
			return fmt.Errorf("%s[%s]: must be a positive quantity, got %q", field, name, qty.String())
		}
	}
	return nil
}

// SetupLanguageAgentRuntimeWebhookWithManager registers the LanguageAgentRuntime validating webhook.
func SetupLanguageAgentRuntimeWebhookWithManager(mgr ctrl.Manager, registryManager validation.RegistryManager) error {
	h := &LanguageAgentRuntimeWebhook{RegistryManager: registryManager}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageAgentRuntime{}).
		WithValidator(h).
		Complete()
}
