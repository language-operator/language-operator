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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/reconciler"
)

const selfConfigTTL = time.Hour

// LanguageAgentSelfConfigReconciler reconciles LanguageAgentSelfConfig objects.
type LanguageAgentSelfConfigReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Log          logr.Logger
	Recorder     record.EventRecorder
	EventManager *events.EventManager
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagentselfconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagentselfconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagentselfconfigs/finalizers,verbs=update

// Reconcile reconciles a LanguageAgentSelfConfig resource.
func (r *LanguageAgentSelfConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageAgentSelfConfig]{
		Client:       r.Client,
		TracerName:   "language-operator/selfconfig-controller",
		ResourceType: "selfconfig",
	}

	sc := &langopv1alpha1.LanguageAgentSelfConfig{}
	result, err := helper.StartReconcile(ctx, req, sc)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result == nil {
		return ctrl.Result{}, nil
	}

	var reconcileErr error
	defer func() {
		result.CompleteReconcile(reconcileErr)
	}()

	ctx = result.Ctx
	span := result.Span
	log := log.FromContext(ctx)

	// Handle deletion
	if !sc.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, RemoveFinalizer(ctx, r.Client, sc)
	}

	// Add finalizer on first pass
	if !controllerutil.ContainsFinalizer(sc, FinalizerName) {
		controllerutil.AddFinalizer(sc, FinalizerName)
		if err := r.Update(ctx, sc); err != nil {
			log.Error(err, "Failed to add finalizer")
			reconcileErr = err
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// If already in a terminal phase, manage TTL deletion.
	if isTerminal(sc.Status.Phase) {
		return r.handleTTL(ctx, sc)
	}

	// Look up the parent LanguageAgent.
	agent := &langopv1alpha1.LanguageAgent{}
	if err := r.Get(ctx, client.ObjectKey{Name: sc.Spec.InstanceRef, Namespace: sc.Namespace}, agent); err != nil {
		if apierrors.IsNotFound(err) {
			return r.setTerminal(ctx, sc, langopv1alpha1.SelfConfigPhaseFailed,
				fmt.Sprintf("LanguageAgent %q not found", sc.Spec.InstanceRef))
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Set owner reference so the SelfConfig is GC'd when the agent is deleted.
	if err := controllerutil.SetOwnerReference(agent, sc, r.Scheme); err != nil {
		reconcileErr = err
		return ctrl.Result{}, fmt.Errorf("failed to set owner reference: %w", err)
	}
	if err := r.Update(ctx, sc); err != nil {
		reconcileErr = err
		return ctrl.Result{}, fmt.Errorf("failed to update owner reference: %w", err)
	}

	// Gate: self-configure must be enabled on the parent.
	if agent.Spec.SelfConfigure == nil || !agent.Spec.SelfConfigure.Enabled {
		return r.setTerminal(ctx, sc, langopv1alpha1.SelfConfigPhaseDenied,
			"self-configure is not enabled on the target LanguageAgent")
	}

	// Gate: validate each requested action against the allowlist.
	if denied, msg := r.validateActions(sc, agent.Spec.SelfConfigure.AllowedActions); denied {
		return r.setTerminal(ctx, sc, langopv1alpha1.SelfConfigPhaseDenied, msg)
	}

	// Apply the patch to the parent LanguageAgent spec with optimistic concurrency.
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &langopv1alpha1.LanguageAgent{}
		if err := r.Get(ctx, client.ObjectKey{Name: sc.Spec.InstanceRef, Namespace: sc.Namespace}, latest); err != nil {
			return err
		}
		applyPatch(latest, sc)
		return r.Update(ctx, latest)
	}); err != nil {
		log.Error(err, "Failed to patch parent LanguageAgent")
		span.RecordError(err)
		span.SetStatus(codes.Error, "patch failed")
		reconcileErr = err
		return ctrl.Result{}, err
	}

	log.Info("Self-config applied", "agent", sc.Spec.InstanceRef)
	span.SetStatus(codes.Ok, "Applied")
	return r.setTerminal(ctx, sc, langopv1alpha1.SelfConfigPhaseApplied, "changes applied to LanguageAgent spec")
}

// handleTTL deletes the SelfConfig once the TTL has elapsed after CompletionTime.
func (r *LanguageAgentSelfConfigReconciler) handleTTL(ctx context.Context, sc *langopv1alpha1.LanguageAgentSelfConfig) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if sc.Status.CompletionTime == nil {
		return ctrl.Result{}, nil
	}
	expiry := sc.Status.CompletionTime.Add(selfConfigTTL)
	remaining := time.Until(expiry)
	if remaining <= 0 {
		log.Info("Deleting expired SelfConfig", "phase", sc.Status.Phase)
		if err := r.Delete(ctx, sc); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: remaining}, nil
}

// setTerminal writes a terminal status and returns the TTL requeue.
func (r *LanguageAgentSelfConfigReconciler) setTerminal(ctx context.Context, sc *langopv1alpha1.LanguageAgentSelfConfig, phase langopv1alpha1.SelfConfigPhase, msg string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	now := metav1.Now()
	sc.Status.Phase = phase
	sc.Status.Message = msg
	sc.Status.CompletionTime = &now
	if err := r.Status().Update(ctx, sc); err != nil {
		log.Error(err, "Failed to update SelfConfig status", "phase", phase)
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: selfConfigTTL}, nil
}

// validateActions checks every requested action against the allowlist.
// Returns (true, message) if any action is disallowed.
func (r *LanguageAgentSelfConfigReconciler) validateActions(sc *langopv1alpha1.LanguageAgentSelfConfig, allowed []langopv1alpha1.SelfConfigAction) (bool, string) {
	allowedSet := make(map[langopv1alpha1.SelfConfigAction]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}

	check := func(action langopv1alpha1.SelfConfigAction, present bool) (bool, string) {
		if present && !allowedSet[action] {
			return true, fmt.Sprintf("action %q is not in spec.selfConfigure.allowedActions", action)
		}
		return false, ""
	}

	spec := sc.Spec
	if denied, msg := check(langopv1alpha1.SelfConfigActionTools, len(spec.AddTools) > 0 || len(spec.RemoveTools) > 0); denied {
		return true, msg
	}
	if denied, msg := check(langopv1alpha1.SelfConfigActionModels, len(spec.AddModels) > 0 || len(spec.RemoveModels) > 0); denied {
		return true, msg
	}
	if denied, msg := check(langopv1alpha1.SelfConfigActionEnvVars, len(spec.AddEnvVars) > 0); denied {
		return true, msg
	}
	if denied, msg := check(langopv1alpha1.SelfConfigActionInstructions, spec.UpdateInstructions != ""); denied {
		return true, msg
	}
	if denied, msg := check(langopv1alpha1.SelfConfigActionRoleRules, len(spec.AddRoleRules) > 0); denied {
		return true, msg
	}
	return false, ""
}

// applyPatch mutates the LanguageAgent spec according to the SelfConfig request.
func applyPatch(agent *langopv1alpha1.LanguageAgent, sc *langopv1alpha1.LanguageAgentSelfConfig) {
	spec := sc.Spec

	// Add tools (skip duplicates).
	existing := make(map[string]bool, len(agent.Spec.Tools))
	for _, t := range agent.Spec.Tools {
		existing[t.Name] = true
	}
	for _, name := range spec.AddTools {
		if !existing[name] {
			agent.Spec.Tools = append(agent.Spec.Tools, langopv1alpha1.ToolReference{Name: name})
			existing[name] = true
		}
	}

	// Remove tools.
	if len(spec.RemoveTools) > 0 {
		remove := make(map[string]bool, len(spec.RemoveTools))
		for _, name := range spec.RemoveTools {
			remove[name] = true
		}
		filtered := agent.Spec.Tools[:0]
		for _, t := range agent.Spec.Tools {
			if !remove[t.Name] {
				filtered = append(filtered, t)
			}
		}
		agent.Spec.Tools = filtered
	}

	// Add models (skip duplicates).
	existingModels := make(map[string]bool, len(agent.Spec.Models))
	for _, m := range agent.Spec.Models {
		existingModels[m.Name] = true
	}
	for _, name := range spec.AddModels {
		if !existingModels[name] {
			agent.Spec.Models = append(agent.Spec.Models, langopv1alpha1.ModelReference{Name: name})
			existingModels[name] = true
		}
	}

	// Remove models.
	if len(spec.RemoveModels) > 0 {
		remove := make(map[string]bool, len(spec.RemoveModels))
		for _, name := range spec.RemoveModels {
			remove[name] = true
		}
		filtered := agent.Spec.Models[:0]
		for _, m := range agent.Spec.Models {
			if !remove[m.Name] {
				filtered = append(filtered, m)
			}
		}
		agent.Spec.Models = filtered
	}

	// Inject env vars (overwrite existing by name).
	if len(spec.AddEnvVars) > 0 {
		envIndex := make(map[string]int, len(agent.Spec.Deployment.Env))
		for i, e := range agent.Spec.Deployment.Env {
			envIndex[e.Name] = i
		}
		for _, ev := range spec.AddEnvVars {
			envVar := corev1.EnvVar{Name: ev.Name, Value: ev.Value}
			if idx, found := envIndex[ev.Name]; found {
				agent.Spec.Deployment.Env[idx] = envVar
			} else {
				agent.Spec.Deployment.Env = append(agent.Spec.Deployment.Env, envVar)
				envIndex[ev.Name] = len(agent.Spec.Deployment.Env) - 1
			}
		}
	}

	// Replace instructions.
	if spec.UpdateInstructions != "" {
		agent.Spec.Instructions = spec.UpdateInstructions
	}

	// Append role rules (de-duplicate by APIGroups+Resources+Verbs fingerprint).
	if len(spec.AddRoleRules) > 0 {
		existing := make(map[string]bool, len(agent.Spec.Deployment.RoleRules))
		for _, r := range agent.Spec.Deployment.RoleRules {
			existing[ruleKey(r)] = true
		}
		for _, r := range spec.AddRoleRules {
			if !existing[ruleKey(r)] {
				agent.Spec.Deployment.RoleRules = append(agent.Spec.Deployment.RoleRules, r)
				existing[ruleKey(r)] = true
			}
		}
	}
}

// ruleKey produces a stable string key for a PolicyRule for de-duplication.
func ruleKey(r rbacv1.PolicyRule) string {
	return fmt.Sprintf("%v|%v|%v", r.APIGroups, r.Resources, r.Verbs)
}

// isTerminal returns true for phases that require no further action (only TTL cleanup).
func isTerminal(phase langopv1alpha1.SelfConfigPhase) bool {
	return phase == langopv1alpha1.SelfConfigPhaseApplied ||
		phase == langopv1alpha1.SelfConfigPhaseFailed ||
		phase == langopv1alpha1.SelfConfigPhaseDenied
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentSelfConfigReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgentSelfConfig{}).
		Complete(r)
}
