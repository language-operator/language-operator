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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	reader client.Reader
}

var _ admission.Defaulter[*LanguageAgent] = &LanguageAgentWebhook{}
var _ admission.Validator[*LanguageAgent] = &LanguageAgentWebhook{}

// Default implements admission.Defaulter
func (h *LanguageAgentWebhook) Default(ctx context.Context, a *LanguageAgent) error {
	// Default workspace only when no runtime is set.
	// When a runtime is referenced, its workspace preset takes effect at reconcile time.
	if a.Spec.Workspace == nil && a.Spec.Runtime == "" {
		enabled := true
		a.Spec.Workspace = &WorkspaceSpec{
			Enabled:    &enabled,
			Size:       "10Gi",
			AccessMode: "ReadWriteOnce",
			MountPath:  "/workspace",
		}
	}

	// Default resources
	if a.Spec.Deployment.Resources.Requests == nil && a.Spec.Deployment.Resources.Limits == nil {
		a.Spec.Deployment.Resources = corev1.ResourceRequirements{
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

// ValidateCreate implements admission.Validator
func (h *LanguageAgentWebhook) ValidateCreate(ctx context.Context, a *LanguageAgent) (admission.Warnings, error) {
	if err := h.validateClusterMembership(ctx, a.Namespace); err != nil {
		return nil, err
	}
	warns, err := h.validateRuntime(ctx, a)
	if err != nil {
		return warns, err
	}
	warns = append(warns, securityWarnings(a)...)
	return warns, a.validateSpec()
}

// ValidateUpdate implements admission.Validator
func (h *LanguageAgentWebhook) ValidateUpdate(ctx context.Context, old, a *LanguageAgent) (admission.Warnings, error) {
	// Skip spec validation during deletion — the operator is only removing the finalizer.
	if a.DeletionTimestamp != nil {
		return nil, nil
	}
	if err := h.validateClusterMembership(ctx, a.Namespace); err != nil {
		return nil, err
	}
	if old.Spec.Workspace != nil && a.Spec.Workspace != nil {
		oldQ, err := resource.ParseQuantity(old.Spec.Workspace.Size)
		if err == nil {
			newQ, err := resource.ParseQuantity(a.Spec.Workspace.Size)
			if err == nil && newQ.Cmp(oldQ) < 0 {
				return nil, fmt.Errorf("spec.workspace.size: cannot decrease storage size (was %s, got %s)", old.Spec.Workspace.Size, a.Spec.Workspace.Size)
			}
		}
		// storageClassName and accessMode are immutable PVC fields after creation.
		oldEnabled := old.Spec.Workspace.Enabled == nil || *old.Spec.Workspace.Enabled
		if oldEnabled {
			if ptrStr(old.Spec.Workspace.StorageClassName) != ptrStr(a.Spec.Workspace.StorageClassName) {
				return nil, fmt.Errorf("spec.workspace.storageClassName: field is immutable after PVC creation (was %q, got %q)",
					ptrStr(old.Spec.Workspace.StorageClassName), ptrStr(a.Spec.Workspace.StorageClassName))
			}
			if old.Spec.Workspace.AccessMode != a.Spec.Workspace.AccessMode {
				return nil, fmt.Errorf("spec.workspace.accessMode: field is immutable after PVC creation (was %q, got %q)",
					old.Spec.Workspace.AccessMode, a.Spec.Workspace.AccessMode)
			}
		}
	}
	warns, err := h.validateRuntime(ctx, a)
	if err != nil {
		return warns, err
	}
	warns = append(warns, securityWarnings(a)...)
	return warns, a.validateSpec()
}

// ptrStr dereferences a string pointer, returning "" for nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// securityWarnings returns advisory (non-blocking) warnings for dangerous agent configurations.
func securityWarnings(a *LanguageAgent) admission.Warnings {
	var warns admission.Warnings

	if strings.Contains(a.Spec.Image, ":latest") && !strings.Contains(a.Spec.Image, "@sha256:") {
		warns = append(warns, "spec.image: using ':latest' tag without a digest pin is not reproducible; consider pinning to a specific digest")
	}

	limits := a.Spec.Deployment.Resources.Limits
	if limits != nil {
		if cpu, ok := limits[corev1.ResourceCPU]; ok && cpu.IsZero() {
			warns = append(warns, "spec.deployment.resources.limits.cpu: explicitly set to zero; the container will be throttled")
		}
		if mem, ok := limits[corev1.ResourceMemory]; ok && mem.IsZero() {
			warns = append(warns, "spec.deployment.resources.limits.memory: explicitly set to zero; the container may be OOM-killed unexpectedly")
		}
	}

	sc := a.Spec.Deployment.SecurityContext
	if sc != nil && sc.RunAsNonRoot != nil && !*sc.RunAsNonRoot {
		warns = append(warns, "spec.deployment.securityContext.runAsNonRoot: explicitly set to false; the container will run as root")
	}

	return warns
}

// validateRuntime verifies the referenced LanguageAgentRuntime exists.
// Returns a warning (not an error) on transient API failures so admission is not blocked.
func (h *LanguageAgentWebhook) validateRuntime(ctx context.Context, a *LanguageAgent) (admission.Warnings, error) {
	if a.Spec.Runtime == "" {
		return nil, nil
	}
	rt := &LanguageAgentRuntime{}
	if err := h.Client.Get(ctx, types.NamespacedName{Name: a.Spec.Runtime}, rt); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("spec.runtime: LanguageAgentRuntime %q not found", a.Spec.Runtime)
		}
		// Transient API error — admit with a warning rather than blocking the user.
		return admission.Warnings{fmt.Sprintf("could not verify LanguageAgentRuntime %q: %v", a.Spec.Runtime, err)}, nil
	}
	return nil, nil
}

// ValidateDelete implements admission.Validator
func (h *LanguageAgentWebhook) ValidateDelete(_ context.Context, _ *LanguageAgent) (admission.Warnings, error) {
	return nil, nil
}

func (h *LanguageAgentWebhook) validateClusterMembership(ctx context.Context, namespace string) error {
	r := client.Reader(h.reader)
	if r == nil {
		r = h.Client
	}
	return validateClusterMembership(ctx, r, namespace)
}

// validateSpec performs pure spec validation (no API calls)
func (a *LanguageAgent) validateSpec() error {
	// Image is required unless a runtime is set (the runtime provides the default image).
	if a.Spec.Image == "" && a.Spec.Runtime == "" {
		return fmt.Errorf("spec.image: required when spec.runtime is not set")
	}

	if len(a.Spec.Models) > 0 {
		if err := a.validateModelReferences(); err != nil {
			return fmt.Errorf("spec.models: %w", err)
		}
	}

	if a.Spec.Workspace != nil && (a.Spec.Workspace.Enabled == nil || *a.Spec.Workspace.Enabled) {
		if err := validateWorkspaceSize(a.Spec.Workspace.Size); err != nil {
			return fmt.Errorf("spec.workspace.size: %w", err)
		}
	}

	if len(a.Spec.Ports) > 0 {
		if err := validateAgentPorts(a.Spec.Ports); err != nil {
			return fmt.Errorf("spec.ports: %w", err)
		}
	}

	return nil
}

func validateAgentPorts(ports []AgentPort) error {
	names := make(map[string]bool, len(ports))
	nums := make(map[int32]bool, len(ports))
	exposeCount := 0
	for i, p := range ports {
		if names[p.Name] {
			return fmt.Errorf("ports[%d].name %q is not unique", i, p.Name)
		}
		names[p.Name] = true
		if nums[p.Port] {
			return fmt.Errorf("ports[%d].port %d is not unique", i, p.Port)
		}
		nums[p.Port] = true
		if p.Expose != nil && *p.Expose {
			exposeCount++
		}
	}
	if exposeCount > 1 {
		return fmt.Errorf("at most one port may have expose: true (found %d)", exposeCount)
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

func (a *LanguageAgent) validateModelReferences() error {
	if len(a.Spec.Models) == 0 {
		return nil
	}
	primaryCount := 0
	for i, modelRef := range a.Spec.Models {
		if modelRef.Name == "" {
			return fmt.Errorf("models[%d].name cannot be empty", i)
		}
		if modelRef.Role == "primary" || modelRef.Role == "" {
			primaryCount++
		}
		if modelRef.Priority != nil && *modelRef.Priority < 0 {
			return fmt.Errorf("models[%d].priority cannot be negative", i)
		}
	}
	if primaryCount == 0 {
		return fmt.Errorf("at least one model must have role 'primary'")
	}
	return nil
}

// SetupWebhookWithManager registers the LanguageAgent mutating and validating webhooks.
func SetupLanguageAgentWebhookWithManager(mgr ctrl.Manager) error {
	h := &LanguageAgentWebhook{Client: mgr.GetClient(), reader: mgr.GetAPIReader()}
	return ctrl.NewWebhookManagedBy(mgr, &LanguageAgent{}).
		WithDefaulter(h).
		WithValidator(h).
		Complete()
}
