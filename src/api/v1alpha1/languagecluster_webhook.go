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

	"github.com/language-operator/language-operator/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languagecluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageclusters,verbs=create;update,versions=v1alpha1,name=vlanguagecluster.kb.io,admissionReviewVersions=v1

// LanguageClusterWebhook handles validation for LanguageCluster.
// LanguageCluster is cluster-scoped, so no cluster-membership check is needed.
//
// +kubebuilder:object:generate=false
type LanguageClusterWebhook struct {
	client.Client
}

var _ admission.Validator[*LanguageCluster] = &LanguageClusterWebhook{}

// ValidateCreate implements admission.Validator
func (h *LanguageClusterWebhook) ValidateCreate(ctx context.Context, lc *LanguageCluster) (admission.Warnings, error) {
	if err := lc.validateSpec(); err != nil {
		return nil, err
	}
	return nil, h.validateNamespaceNotClaimed(ctx, lc)
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
	if err := validateNetworkPortPolicies(lc.Spec.NetworkPolicies); err != nil {
		return err
	}
	return nil
}

// validateNetworkPortPolicies validates that all port numbers in networkPolicies are in [1, 65535].
func validateNetworkPortPolicies(policies *AgentNetworkPolicies) error {
	if policies == nil {
		return nil
	}
	for i, rule := range policies.Ingress {
		for j, p := range rule.Ports {
			if p.Port < 1 || p.Port > 65535 {
				return fmt.Errorf("spec.networkPolicies.ingress[%d].ports[%d].port: %d is not in the valid port range [1, 65535]", i, j, p.Port)
			}
		}
	}
	for i, rule := range policies.Egress {
		for j, p := range rule.Ports {
			if p.Port < 1 || p.Port > 65535 {
				return fmt.Errorf("spec.networkPolicies.egress[%d].ports[%d].port: %d is not in the valid port range [1, 65535]", i, j, p.Port)
			}
		}
	}
	return nil
}

// validateNamespaceNotClaimed rejects a new LanguageCluster if a namespace with the same name
// already exists but is not managed by language-operator. This prevents the operator from
// silently taking over pre-existing namespaces.
func (h *LanguageClusterWebhook) validateNamespaceNotClaimed(ctx context.Context, lc *LanguageCluster) error {
	if h.Client == nil {
		return nil
	}
	ns := &corev1.Namespace{}
	err := h.Client.Get(ctx, types.NamespacedName{Name: lc.Name}, ns)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check namespace %q: %w", lc.Name, err)
	}
	if v, ok := ns.Labels[labels.LabelKeyLangopCluster]; ok && v == lc.Name {
		return nil
	}
	return fmt.Errorf("namespace %q already exists and is not managed by language-operator; "+
		"choose a different name or delete the existing namespace first", lc.Name)
}

// SetupLanguageClusterWebhookWithManager registers the LanguageCluster validating webhook.
func SetupLanguageClusterWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageClusterWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageCluster{}).
		WithValidator(h).
		Complete()
}
