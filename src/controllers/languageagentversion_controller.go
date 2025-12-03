package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageAgentVersionReconciler reconciles a LanguageAgentVersion object
type LanguageAgentVersionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagentversions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagentversions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagentversions/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile manages the lifecycle of LanguageAgentVersion resources
func (r *LanguageAgentVersionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LanguageAgentVersion instance
	version := &langopv1alpha1.LanguageAgentVersion{}
	if err := r.Get(ctx, req.NamespacedName, version); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return without error (may have been deleted)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageAgentVersion")
		return ctrl.Result{}, err
	}

	// Set initial status if not set
	if version.Status.Phase == "" {
		version.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, version); err != nil {
			log.Error(err, "Failed to initialize status")
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate the version resource
	if err := r.validateVersion(ctx, version); err != nil {
		return r.updateStatusWithError(ctx, version, "ValidationFailed", err)
	}

	// Ensure the referenced agent exists
	agent, err := r.validateAgentReference(ctx, version)
	if err != nil {
		return r.updateStatusWithError(ctx, version, "AgentNotFound", err)
	}

	// Create or update the underlying ConfigMap
	if err := r.reconcileConfigMap(ctx, version, agent); err != nil {
		return r.updateStatusWithError(ctx, version, "ConfigMapFailed", err)
	}

	// Update status to Ready
	if version.Status.Phase != "Ready" {
		version.Status.Phase = "Ready"
		now := metav1.Now()
		version.Status.AppliedAt = &now
		version.Status.Message = "LanguageAgentVersion is ready and ConfigMap has been created"

		// Add ready condition
		r.setCondition(version, "Ready", metav1.ConditionTrue, "ConfigMapCreated",
			"Successfully created underlying ConfigMap")

		if err := r.Status().Update(ctx, version); err != nil {
			log.Error(err, "Failed to update status to Ready")
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}

		r.Recorder.Event(version, corev1.EventTypeNormal, "Ready",
			fmt.Sprintf("LanguageAgentVersion v%d for agent %s is ready",
				version.Spec.Version, version.Spec.AgentRef.Name))
	}

	log.Info("LanguageAgentVersion reconciled successfully",
		"version", version.Spec.Version,
		"agent", version.Spec.AgentRef.Name,
		"phase", version.Status.Phase)

	return ctrl.Result{}, nil
}

// validateVersion performs validation on the LanguageAgentVersion resource
func (r *LanguageAgentVersionReconciler) validateVersion(ctx context.Context, version *langopv1alpha1.LanguageAgentVersion) error {
	// Validate version number
	if version.Spec.Version < 1 {
		return fmt.Errorf("version must be >= 1, got %d", version.Spec.Version)
	}

	// Validate code is not empty
	if version.Spec.Code == "" {
		return fmt.Errorf("code field cannot be empty")
	}

	// Validate agent reference
	if version.Spec.AgentRef.Name == "" {
		return fmt.Errorf("agentRef.name cannot be empty")
	}

	return nil
}

// validateAgentReference ensures the referenced LanguageAgent exists
func (r *LanguageAgentVersionReconciler) validateAgentReference(ctx context.Context, version *langopv1alpha1.LanguageAgentVersion) (*langopv1alpha1.LanguageAgent, error) {
	// Determine namespace for agent lookup
	agentNamespace := version.Spec.AgentRef.Namespace
	if agentNamespace == "" {
		agentNamespace = version.Namespace
	}

	// Fetch the referenced agent
	agent := &langopv1alpha1.LanguageAgent{}
	agentKey := types.NamespacedName{
		Name:      version.Spec.AgentRef.Name,
		Namespace: agentNamespace,
	}

	if err := r.Get(ctx, agentKey, agent); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("referenced LanguageAgent %s not found in namespace %s",
				version.Spec.AgentRef.Name, agentNamespace)
		}
		return nil, fmt.Errorf("failed to get referenced LanguageAgent: %w", err)
	}

	return agent, nil
}

// reconcileConfigMap creates or updates the underlying ConfigMap for this version
func (r *LanguageAgentVersionReconciler) reconcileConfigMap(ctx context.Context, version *langopv1alpha1.LanguageAgentVersion, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Generate ConfigMap name
	configMapName := fmt.Sprintf("%s-v%d", version.Spec.AgentRef.Name, version.Spec.Version)

	// Create ConfigMap data
	data := map[string]string{
		"agent.rb": version.Spec.Code,
	}

	// Add metadata about the optimization
	if len(version.Spec.OptimizedTasks) > 0 {
		optimizedTaskNames := make([]string, 0, len(version.Spec.OptimizedTasks))
		for taskName := range version.Spec.OptimizedTasks {
			optimizedTaskNames = append(optimizedTaskNames, taskName)
		}
		data["optimized_tasks.txt"] = fmt.Sprintf("Optimized tasks: %v", optimizedTaskNames)
	}

	// Create the ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: version.Namespace,
			Labels: map[string]string{
				"langop.io/agent":      version.Spec.AgentRef.Name,
				"langop.io/component":  "agent-code",
				"langop.io/version":    fmt.Sprintf("%d", version.Spec.Version),
				"langop.io/source":     version.Spec.SourceType,
				"langop.io/managed-by": "language-operator",
			},
			Annotations: map[string]string{
				"langop.io/description":  version.Spec.Description,
				"langop.io/created-by":   version.Spec.CreatedBy,
				"langop.io/version-name": version.Name,
			},
		},
		Data: data,
	}

	// Set owner reference so ConfigMap is cleaned up when LanguageAgentVersion is deleted
	if err := controllerutil.SetControllerReference(version, configMap, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create or update the ConfigMap
	if err := r.Create(ctx, configMap); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// ConfigMap exists, update it
			existingConfigMap := &corev1.ConfigMap{}
			if err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: version.Namespace}, existingConfigMap); err != nil {
				return fmt.Errorf("failed to get existing ConfigMap: %w", err)
			}

			// Update data and metadata
			existingConfigMap.Data = data
			existingConfigMap.Labels = configMap.Labels
			existingConfigMap.Annotations = configMap.Annotations

			if err := r.Update(ctx, existingConfigMap); err != nil {
				return fmt.Errorf("failed to update ConfigMap: %w", err)
			}

			log.Info("Updated existing ConfigMap", "configMap", configMapName)
		} else {
			return fmt.Errorf("failed to create ConfigMap: %w", err)
		}
	} else {
		log.Info("Created new ConfigMap", "configMap", configMapName)
	}

	// Update status with ConfigMap name
	version.Status.ConfigMapName = configMapName

	return nil
}

// updateStatusWithError updates the status with error information
func (r *LanguageAgentVersionReconciler) updateStatusWithError(ctx context.Context, version *langopv1alpha1.LanguageAgentVersion, reason string, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	version.Status.Phase = "Failed"
	version.Status.Message = err.Error()

	// Add failed condition
	r.setCondition(version, "Ready", metav1.ConditionFalse, reason, err.Error())

	if updateErr := r.Status().Update(ctx, version); updateErr != nil {
		log.Error(updateErr, "Failed to update status with error")
		return ctrl.Result{RequeueAfter: time.Second * 30}, updateErr
	}

	r.Recorder.Event(version, corev1.EventTypeWarning, reason, err.Error())

	log.Error(err, "LanguageAgentVersion reconciliation failed",
		"reason", reason,
		"version", version.Spec.Version,
		"agent", version.Spec.AgentRef.Name)

	// Requeue with backoff
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// setCondition adds or updates a condition in the version status
func (r *LanguageAgentVersionReconciler) setCondition(version *langopv1alpha1.LanguageAgentVersion, conditionType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition and update it, or append new one
	for i, existing := range version.Status.Conditions {
		if existing.Type == conditionType {
			// Update existing condition
			if existing.Status != status {
				condition.LastTransitionTime = metav1.Now()
			} else {
				condition.LastTransitionTime = existing.LastTransitionTime
			}
			version.Status.Conditions[i] = condition
			return
		}
	}

	// Add new condition
	version.Status.Conditions = append(version.Status.Conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgentVersion{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
