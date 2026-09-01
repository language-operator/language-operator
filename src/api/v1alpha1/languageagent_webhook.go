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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	// Default workspace storage when the agent doesn't specify it. Provisioning is an
	// agent/cluster concern, so it is always defaulted here regardless of runtime or repository.
	if a.Spec.Workspace == nil {
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

	// Agents run as Argo Workflows, which have no replica or scale semantics.
	// Reject rather than silently ignore — a user who sets replicas: 3 would
	// otherwise get one pod and no explanation.
	if a.Spec.Deployment.Replicas != nil {
		return fmt.Errorf("spec.deployment.replicas: not supported; agents run as Argo Workflows, which have no replicas")
	}
	if a.Spec.Deployment.Autoscaling != nil {
		return fmt.Errorf("spec.deployment.autoscaling: not supported; agents run as Argo Workflows, which have no scale subresource to target")
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

	if a.Spec.Repository != nil {
		if err := validateRepository(a.Spec.Repository, a.Spec.Workspace); err != nil {
			return fmt.Errorf("spec.repository: %w", err)
		}
	}

	if err := validateExecution(&a.Spec.Execution, a.Spec.Ports); err != nil {
		return err
	}

	return nil
}

// ExecutionModeService is the always-on execution mode: a long-lived Argo Workflow.
const ExecutionModeService = "service"

// ExecutionModeTask is the invoked execution mode: one-shot Argo Workflow runs.
const ExecutionModeTask = "task"

// EffectiveExecutionMode returns the agent's execution mode, defaulting to "service".
// The CRD defaults spec.execution.mode, but objects built in code (tests, fixtures)
// bypass defaulting, so callers must not assume the field is populated.
func (e *ExecutionSpec) EffectiveExecutionMode() string {
	if e.Mode == "" {
		return ExecutionModeService
	}
	return e.Mode
}

// validateExecution rejects execution settings that are meaningless for the chosen mode.
// Silently ignoring them would leave the user believing a schedule or deadline is in
// force when nothing acts on it.
func validateExecution(e *ExecutionSpec, ports []AgentPort) error {
	if e.EffectiveExecutionMode() == ExecutionModeService {
		switch {
		case e.Schedule != "":
			return fmt.Errorf("spec.execution.schedule: only valid when mode is %q (a service agent runs continuously)", ExecutionModeTask)
		case e.ActiveDeadlineSeconds != nil:
			return fmt.Errorf("spec.execution.activeDeadlineSeconds: only valid when mode is %q (a service agent is not expected to finish)", ExecutionModeTask)
		case e.TTLSecondsAfterFinished != nil:
			return fmt.Errorf("spec.execution.ttlSecondsAfterFinished: only valid when mode is %q", ExecutionModeTask)
		case e.RetryLimit != nil:
			return fmt.Errorf("spec.execution.retryLimit: only valid when mode is %q (a service agent always retries so it stays up)", ExecutionModeTask)
		}
	}

	// timezone and concurrencyPolicy only take effect through a CronWorkflow.
	if e.Schedule == "" {
		if e.Timezone != "" {
			return fmt.Errorf("spec.execution.timezone: only valid alongside spec.execution.schedule")
		}
	} else {
		if err := validateCronSchedule(e.Schedule); err != nil {
			return fmt.Errorf("spec.execution.schedule: %w", err)
		}
		if e.Timezone != "" {
			if _, err := time.LoadLocation(e.Timezone); err != nil {
				return fmt.Errorf("spec.execution.timezone: %q is not a valid IANA timezone", e.Timezone)
			}
		}
	}

	// A task agent is not addressable: its pods come and go, so a Service and its
	// ports would point at nothing between runs.
	if e.EffectiveExecutionMode() == ExecutionModeTask && len(ports) > 0 {
		return fmt.Errorf("spec.ports: not supported when spec.execution.mode is %q (task runs are not addressable)", ExecutionModeTask)
	}

	return nil
}

// cronFieldRange is the permitted numeric range for each of the five cron fields.
var cronFieldRange = [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}

// cronMacros are the shorthand schedules Argo's cron parser accepts.
var cronMacros = map[string]bool{
	"@yearly": true, "@annually": true, "@monthly": true, "@weekly": true,
	"@daily": true, "@midnight": true, "@hourly": true,
}

// cronMonths and cronDays are the textual aliases accepted in the month and
// day-of-week fields respectively.
var cronMonths = map[string]bool{
	"jan": true, "feb": true, "mar": true, "apr": true, "may": true, "jun": true,
	"jul": true, "aug": true, "sep": true, "oct": true, "nov": true, "dec": true,
}
var cronDays = map[string]bool{
	"sun": true, "mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true,
}

// validateCronSchedule checks a standard 5-field cron expression or an @-macro.
// Catching a typo at admission is far cheaper than an agent that silently never fires.
func validateCronSchedule(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("cannot be empty")
	}
	if strings.HasPrefix(s, "@") {
		if cronMacros[strings.ToLower(s)] {
			return nil
		}
		if rest, ok := strings.CutPrefix(strings.ToLower(s), "@every "); ok {
			if _, err := time.ParseDuration(strings.TrimSpace(rest)); err != nil {
				return fmt.Errorf("%q has an invalid @every duration", s)
			}
			return nil
		}
		return fmt.Errorf("%q is not a recognized cron macro", s)
	}

	fields := strings.Fields(s)
	if len(fields) != 5 {
		return fmt.Errorf("%q must have 5 fields (minute hour day-of-month month day-of-week), got %d", s, len(fields))
	}
	for i, f := range fields {
		if err := validateCronField(f, cronFieldRange[i][0], cronFieldRange[i][1], i); err != nil {
			return fmt.Errorf("%q: field %d: %w", s, i+1, err)
		}
	}
	return nil
}

// validateCronField validates one cron field, which may be a comma-separated list of
// entries, each optionally stepped ("*/5", "1-30/2") and each a wildcard, a single
// value, or a range.
func validateCronField(field string, min, max, index int) error {
	for _, entry := range strings.Split(field, ",") {
		if entry == "" {
			return fmt.Errorf("empty entry in %q", field)
		}
		value, step, hasStep := strings.Cut(entry, "/")
		if hasStep {
			n, err := strconv.Atoi(step)
			if err != nil || n < 1 {
				return fmt.Errorf("step %q must be a positive integer", step)
			}
		}
		if value == "*" {
			continue
		}
		lo, hi, isRange := strings.Cut(value, "-")
		if err := validateCronValue(lo, min, max, index); err != nil {
			return err
		}
		if isRange {
			if err := validateCronValue(hi, min, max, index); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateCronValue validates a single cron value: a number in range, or a textual
// month/day alias in the fields that accept one.
func validateCronValue(v string, min, max, index int) error {
	if v == "" {
		return fmt.Errorf("empty value")
	}
	lower := strings.ToLower(v)
	if index == 3 && cronMonths[lower] {
		return nil
	}
	if index == 4 && cronDays[lower] {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%q is not a number", v)
	}
	if n < min || n > max {
		return fmt.Errorf("%d is out of range %d-%d", n, min, max)
	}
	return nil
}

// scpLikeURL matches the scp-like SSH syntax git uses, e.g. "git@github.com:org/repo.git".
var scpLikeURL = regexp.MustCompile(`^[^@/]+@[^:/]+:.+$`)

// validateRepository validates a RepositorySpec: the URL must be a git-cloneable
// http(s)/ssh reference, the path (if set) must be relative, and a repository may not be
// declared when the workspace is explicitly disabled (the clone needs a PVC to land in).
func validateRepository(r *RepositorySpec, ws *WorkspaceSpec) error {
	if ws != nil && ws.Enabled != nil && !*ws.Enabled {
		return fmt.Errorf("cannot be set when spec.workspace.enabled is false (a workspace is required to clone into)")
	}

	if strings.TrimSpace(r.URL) == "" {
		return fmt.Errorf("url cannot be empty")
	}
	if err := validateRepositoryURL(r.URL); err != nil {
		return fmt.Errorf("url: %w", err)
	}

	if r.Path != "" {
		if strings.HasPrefix(r.Path, "/") {
			return fmt.Errorf("path %q must be relative (no leading '/')", r.Path)
		}
		for _, seg := range strings.Split(r.Path, "/") {
			if seg == ".." {
				return fmt.Errorf("path %q must not contain '..' segments", r.Path)
			}
		}
	}

	return nil
}

// validateRepositoryURL accepts http(s)/ssh URLs and the scp-like SSH form.
func validateRepositoryURL(raw string) error {
	if scpLikeURL.MatchString(raw) {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", raw)
	}
	switch u.Scheme {
	case "http", "https", "ssh":
		if u.Host == "" {
			return fmt.Errorf("%q is missing a host", raw)
		}
		return nil
	default:
		return fmt.Errorf("%q must use http, https, or ssh (or the git@host:path SSH form)", raw)
	}
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
