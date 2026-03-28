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

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-langop-io-v1alpha1-languageagent,mutating=true,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagents,verbs=create;update,versions=v1alpha1,name=mlanguageagent.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-langop-io-v1alpha1-languageagent,mutating=false,failurePolicy=fail,sideEffects=None,groups=langop.io,resources=languageagents,verbs=create;update,versions=v1alpha1,name=vlanguageagent.kb.io,admissionReviewVersions=v1

// LanguageAgentWebhook handles defaulting and validation for LanguageAgent.
// It holds a client so it can verify LanguageCluster membership at admission time.
//
// +kubebuilder:object:generate=false
type LanguageAgentWebhook struct {
	client.Client
}

var _ webhook.CustomDefaulter = &LanguageAgentWebhook{}
var _ webhook.CustomValidator = &LanguageAgentWebhook{}

// Default implements webhook.CustomDefaulter
func (h *LanguageAgentWebhook) Default(ctx context.Context, obj runtime.Object) error {
	a := obj.(*LanguageAgent)

	// Default workspace
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

	// Default resources
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

	return nil
}

// ValidateCreate implements webhook.CustomValidator
func (h *LanguageAgentWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	a := obj.(*LanguageAgent)
	if err := h.validateClusterMembership(ctx, a.Namespace); err != nil {
		return nil, err
	}
	return nil, a.validateSpec()
}

// ValidateUpdate implements webhook.CustomValidator
func (h *LanguageAgentWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	a := obj.(*LanguageAgent)
	if err := h.validateClusterMembership(ctx, a.Namespace); err != nil {
		return nil, err
	}
	return nil, a.validateSpec()
}

// ValidateDelete implements webhook.CustomValidator
func (h *LanguageAgentWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validateClusterMembership verifies a LanguageCluster exists for this namespace.
func (h *LanguageAgentWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	cluster := &LanguageCluster{}
	if err := h.Get(ctx, types.NamespacedName{Name: namespace}, cluster); err != nil {
		return fmt.Errorf("namespace %q is not managed by a LanguageCluster: no cluster %q exists", namespace, namespace)
	}
	return nil
}

// validateSpec performs pure spec validation (no API calls)
func (a *LanguageAgent) validateSpec() error {
	if len(a.Spec.ModelRefs) > 0 {
		if err := a.validateModelReferences(); err != nil {
			return fmt.Errorf("spec.modelRefs: %w", err)
		}
	}

	if err := a.validateExecutionMode(); err != nil {
		return fmt.Errorf("spec.executionMode: %w", err)
	}

	if a.Spec.SafetyConfig != nil {
		if a.Spec.SafetyConfig.MaxCostPerExecution != nil && *a.Spec.SafetyConfig.MaxCostPerExecution < 0 {
			return fmt.Errorf("spec.safetyConfig.maxCostPerExecution must be non-negative")
		}
	}

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

	if a.Spec.Workspace != nil && a.Spec.Workspace.Enabled {
		if err := validateWorkspaceSize(a.Spec.Workspace.Size); err != nil {
			return fmt.Errorf("spec.workspace.size: %w", err)
		}
	}

	if err := a.validateSchedule(); err != nil {
		return fmt.Errorf("spec.schedule: %w", err)
	}

	return nil
}

func validateWorkspaceSize(size string) error {
	if size == "" {
		return fmt.Errorf("cannot be empty, PersistentVolumeClaims require explicit storage size (e.g., \"10Gi\", \"1.5Ti\")")
	}
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("invalid format %q, expected Kubernetes quantity format (e.g., \"10Gi\", \"1.5Ti\")", size)
	}
	if quantity.IsZero() {
		return fmt.Errorf("cannot be zero, PersistentVolumeClaims require non-zero storage")
	}
	if quantity.Sign() < 0 {
		return fmt.Errorf("cannot be negative, got: %s", size)
	}
	return nil
}

func (a *LanguageAgent) validateSchedule() error {
	if a.Spec.ExecutionMode == "scheduled" && a.Spec.Schedule == "" {
		return fmt.Errorf("schedule is required when executionMode is 'scheduled'")
	}
	if a.Spec.Schedule != "" {
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
		if modelRef.Name == "" {
			return fmt.Errorf("modelRefs[%d].name cannot be empty", i)
		}
		if modelRef.Role == "primary" || modelRef.Role == "" {
			primaryCount++
		}
		if modelRef.Priority != nil && *modelRef.Priority < 0 {
			return fmt.Errorf("modelRefs[%d].priority cannot be negative", i)
		}
	}
	if primaryCount == 0 {
		return fmt.Errorf("at least one model must have role 'primary'")
	}
	return nil
}

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
		// no additional config required
	default:
		return fmt.Errorf("invalid executionMode %q, must be one of: autonomous, interactive, scheduled, event-driven", a.Spec.ExecutionMode)
	}
	return nil
}

// SetupWebhookWithManager registers the LanguageAgent mutating and validating webhooks.
func SetupLanguageAgentWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageAgentWebhook{Client: mgr.GetClient()}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&LanguageAgent{}).
		WithDefaulter(h).
		WithValidator(h).
		Complete()
}
