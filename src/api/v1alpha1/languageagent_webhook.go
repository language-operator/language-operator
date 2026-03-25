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

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	// Default port to 8080
	if a.Spec.Port == nil {
		port := int32(8080)
		a.Spec.Port = &port
	}

	// Default resources if not specified
	if a.Spec.Resources.Requests == nil && a.Spec.Resources.Limits == nil {
		a.Spec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
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

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator
func (a *LanguageAgent) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	warnings := admission.Warnings{}

	// Basic validation
	if err := a.validateSpec(); err != nil {
		return warnings, err
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator
func (a *LanguageAgent) ValidateDelete() (admission.Warnings, error) {
	// No validation needed on delete
	return nil, nil
}

// validateSpec performs basic spec validation
func (a *LanguageAgent) validateSpec() error {
	// Validate model references if any are provided
	if len(a.Spec.ModelRefs) > 0 {
		if err := a.validateModelReferences(); err != nil {
			return fmt.Errorf("spec.modelRefs: %w", err)
		}
	}

	// Validate execution mode dependencies
	if err := a.validateExecutionMode(); err != nil {
		return fmt.Errorf("spec.executionMode: %w", err)
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

	// Validate workspace configuration if present and enabled
	if a.Spec.Workspace != nil && a.Spec.Workspace.Enabled {
		if err := a.validateWorkspaceSize(a.Spec.Workspace.Size); err != nil {
			return fmt.Errorf("spec.workspace.size: %w", err)
		}
	}

	// Validate schedule configuration for scheduled agents
	if err := a.validateSchedule(); err != nil {
		return fmt.Errorf("spec.schedule: %w", err)
	}

	return nil
}

// validateWorkspaceSize validates the workspace size format and constraints
func (a *LanguageAgent) validateWorkspaceSize(size string) error {
	// Check for empty size - not allowed since PVCs require explicit storage
	if size == "" {
		return fmt.Errorf("cannot be empty, PersistentVolumeClaims require explicit storage size (e.g., \"10Gi\", \"1.5Ti\")")
	}

	// Parse the size to ensure it's valid
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("invalid format %q, expected Kubernetes quantity format (e.g., \"10Gi\", \"1.5Ti\")", size)
	}

	// Ensure size is not zero (PVCs require non-zero storage)
	if quantity.IsZero() {
		return fmt.Errorf("cannot be zero, PersistentVolumeClaims require non-zero storage")
	}

	// Ensure size is positive (negative quantities don't make sense for storage)
	if quantity.Sign() < 0 {
		return fmt.Errorf("cannot be negative, got: %s", size)
	}

	return nil
}

// validateSchedule validates the cron schedule format and constraints
func (a *LanguageAgent) validateSchedule() error {
	// If execution mode is scheduled, schedule is required
	if a.Spec.ExecutionMode == "scheduled" {
		if a.Spec.Schedule == "" {
			return fmt.Errorf("schedule is required when executionMode is 'scheduled'")
		}
	}

	// If schedule is provided, validate the cron syntax
	if a.Spec.Schedule != "" {
		// Use cron parser to validate the schedule
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

		if _, err := parser.Parse(a.Spec.Schedule); err != nil {
			return fmt.Errorf("invalid cron expression %q: %w", a.Spec.Schedule, err)
		}
	}

	return nil
}

func (a *LanguageAgent) validateModelReferences() error {
	if len(a.Spec.ModelRefs) == 0 {
		return nil
	}
	primaryCount := 0
	for i, modelRef := range a.Spec.ModelRefs {
		// Validate basic fields
		if modelRef.Name == "" {
			return fmt.Errorf("modelRefs[%d].name cannot be empty", i)
		}

		// Count primary models
		if modelRef.Role == "primary" || modelRef.Role == "" {
			primaryCount++
		}

		// Validate priority if set
		if modelRef.Priority != nil && *modelRef.Priority < 0 {
			return fmt.Errorf("modelRefs[%d].priority cannot be negative", i)
		}
	}

	// Ensure at least one primary model
	if primaryCount == 0 {
		return fmt.Errorf("at least one model must have role 'primary'")
	}

	return nil
}

// validateExecutionMode validates execution mode dependencies
func (a *LanguageAgent) validateExecutionMode() error {
	switch a.Spec.ExecutionMode {
	case "scheduled":
		if a.Spec.Schedule == "" {
			return fmt.Errorf("schedule is required when executionMode is 'scheduled'")
		}
	case "event-driven":
		if len(a.Spec.EventTriggers) == 0 {
			return fmt.Errorf("eventTriggers is required when executionMode is 'event-driven'")
		}
	case "autonomous", "interactive", "":
		// These modes don't require additional configuration
	default:
		return fmt.Errorf("invalid executionMode %q, must be one of: autonomous, interactive, scheduled, event-driven", a.Spec.ExecutionMode)
	}

	return nil
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (a *LanguageAgent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(a).
		Complete()
}
