/*
Copyright 2025 Based Team.

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
	"net"
	"strings"

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
	// Validate group names are unique
	groupNames := make(map[string]bool)
	for _, group := range c.Spec.Groups {
		if groupNames[group.Name] {
			return fmt.Errorf("duplicate group name: %s", group.Name)
		}
		groupNames[group.Name] = true

		// Reserved name
		if group.Name == "default" {
			return fmt.Errorf("group name 'default' is reserved")
		}
	}

	// Validate group references in rules
	for _, group := range c.Spec.Groups {
		for _, rule := range group.Ingress {
			if err := validateNetworkRule(rule, groupNames); err != nil {
				return fmt.Errorf("invalid ingress rule in group %s: %w", group.Name, err)
			}
		}
		for _, rule := range group.Egress {
			if err := validateNetworkRule(rule, groupNames); err != nil {
				return fmt.Errorf("invalid egress rule in group %s: %w", group.Name, err)
			}
		}
	}

	return nil
}

func validateNetworkRule(rule NetworkRule, validGroups map[string]bool) error {
	peer := rule.From
	if peer == nil {
		peer = rule.To
	}
	if peer == nil {
		return fmt.Errorf("rule must have either 'from' or 'to'")
	}

	// Validate group reference
	if peer.Group != "" && !validGroups[peer.Group] && peer.Group != "default" {
		return fmt.Errorf("group '%s' does not exist", peer.Group)
	}

	// Validate CIDR
	if peer.CIDR != "" {
		if _, _, err := net.ParseCIDR(peer.CIDR); err != nil {
			return fmt.Errorf("invalid CIDR: %w", err)
		}
	}

	// Validate DNS patterns
	for _, dns := range peer.DNS {
		if !isValidDNSPattern(dns) {
			return fmt.Errorf("invalid DNS pattern: %s", dns)
		}
	}

	// Validate ports
	for _, port := range rule.Ports {
		if port.Port < 1 || port.Port > 65535 {
			return fmt.Errorf("invalid port: %d", port.Port)
		}
	}

	return nil
}

func isValidDNSPattern(pattern string) bool {
	// Allow wildcards with *
	// Validate basic DNS format
	if pattern == "" {
		return false
	}

	// Replace * with valid char for validation
	testPattern := strings.ReplaceAll(pattern, "*", "a")

	// Basic DNS validation
	parts := strings.Split(testPattern, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		// Check for valid DNS characters
		for _, ch := range part {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
				(ch >= '0' && ch <= '9') || ch == '-') {
				return false
			}
		}
	}

	return true
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (c *LanguageCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}
