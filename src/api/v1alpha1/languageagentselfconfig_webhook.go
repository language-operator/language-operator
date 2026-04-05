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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languageagentselfconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagentselfconfigs,verbs=create;update,versions=v1alpha1,name=vlanguageagentselfconfig.kb.io,admissionReviewVersions=v1

// LanguageAgentSelfConfigWebhook handles validation for LanguageAgentSelfConfig.
//
// +kubebuilder:object:generate=false
type LanguageAgentSelfConfigWebhook struct {
	client.Client
}

var _ webhook.CustomValidator = &LanguageAgentSelfConfigWebhook{}

// ValidateCreate validates a new LanguageAgentSelfConfig.
func (h *LanguageAgentSelfConfigWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	sc := obj.(*LanguageAgentSelfConfig)
	return nil, validateSelfConfigSpec(sc)
}

// ValidateUpdate rejects spec changes once the request has reached a terminal phase.
func (h *LanguageAgentSelfConfigWebhook) ValidateUpdate(_ context.Context, newObj runtime.Object, oldObj runtime.Object) (admission.Warnings, error) {
	old := oldObj.(*LanguageAgentSelfConfig)
	if isTerminalPhase(old.Status.Phase) {
		return nil, fmt.Errorf("LanguageAgentSelfConfig %q is already in terminal phase %q and cannot be modified",
			old.Name, old.Status.Phase)
	}
	sc := newObj.(*LanguageAgentSelfConfig)
	return nil, validateSelfConfigSpec(sc)
}

// ValidateDelete always allows deletion.
func (h *LanguageAgentSelfConfigWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateSelfConfigSpec(sc *LanguageAgentSelfConfig) error {
	if sc.Spec.InstanceRef == "" {
		return fmt.Errorf("spec.instanceRef must not be empty")
	}
	for i, ev := range sc.Spec.AddEnvVars {
		if ev.Name == "" {
			return fmt.Errorf("spec.addEnvVars[%d].name must not be empty", i)
		}
	}
	for i, rule := range sc.Spec.AddRoleRules {
		if len(rule.APIGroups) == 0 {
			return fmt.Errorf("spec.addRoleRules[%d].apiGroups must not be empty", i)
		}
		if len(rule.Resources) == 0 {
			return fmt.Errorf("spec.addRoleRules[%d].resources must not be empty", i)
		}
	}
	return nil
}

func isTerminalPhase(phase SelfConfigPhase) bool {
	return phase == SelfConfigPhaseApplied ||
		phase == SelfConfigPhaseFailed ||
		phase == SelfConfigPhaseDenied
}

// SetupLanguageAgentSelfConfigWebhookWithManager registers the LanguageAgentSelfConfig validating webhook.
func SetupLanguageAgentSelfConfigWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageAgentSelfConfigWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguageAgentSelfConfig{}).
		WithValidator(h).
		Complete()
}
