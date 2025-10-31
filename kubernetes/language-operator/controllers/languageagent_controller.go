package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguageAgentReconciler reconciles a LanguageAgent object
type LanguageAgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LanguageAgent instance
	agent := &langopv1alpha1.LanguageAgent{}
	if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageAgent")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !agent.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(agent, FinalizerName) {
			if err := r.cleanupResources(ctx, agent); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(agent, FinalizerName)
			if err := r.Update(ctx, agent); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(agent, FinalizerName) {
		controllerutil.AddFinalizer(agent, FinalizerName)
		if err := r.Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "ConfigMapError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile PVC for workspace if enabled
	if err := r.reconcilePVC(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile PVC")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "PVCError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile workload based on execution mode
	switch agent.Spec.ExecutionMode {
	case "continuous", "reactive", "":
		if err := r.reconcileDeployment(ctx, agent); err != nil {
			log.Error(err, "Failed to reconcile Deployment")
			SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
	case "scheduled":
		if err := r.reconcileCronJob(ctx, agent); err != nil {
			log.Error(err, "Failed to reconcile CronJob")
			SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "CronJobError", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
	}

	// Update status
	agent.Status.Phase = "Ready"
	SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageAgent is ready", agent.Generation)

	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update LanguageAgent status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LanguageAgentReconciler) reconcileConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	data := make(map[string]string)

	// Add agent spec as JSON
	specJSON, err := json.Marshal(agent.Spec)
	if err != nil {
		return err
	}
	data["agent.json"] = string(specJSON)

	// Add other useful data
	data["name"] = agent.Name
	data["namespace"] = agent.Namespace
	data["mode"] = agent.Spec.ExecutionMode
	if agent.Spec.Goal != "" {
		data["goal"] = agent.Spec.Goal
	}

	configMapName := GenerateConfigMapName(agent.Name, "agent")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, configMapName, agent.Namespace, data)
}

func (r *LanguageAgentReconciler) reconcilePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Skip if workspace is not enabled
	if agent.Spec.Workspace == nil || !agent.Spec.Workspace.Enabled {
		return nil
	}

	// Determine target namespace
	targetNamespace := agent.Namespace
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef}, cluster); err != nil {
			return err
		}
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", agent.Spec.ClusterRef)
		}
		targetNamespace = cluster.Status.Namespace
	}

	// Set defaults from WorkspaceSpec
	size := agent.Spec.Workspace.Size
	if size == "" {
		size = "10Gi"
	}

	accessMode := corev1.PersistentVolumeAccessMode(agent.Spec.Workspace.AccessMode)
	if accessMode == "" {
		accessMode = corev1.ReadWriteOnce
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name + "-workspace",
			Namespace: targetNamespace,
			Labels:    GetCommonLabels(agent.Name, "LanguageAgent"),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		if err := controllerutil.SetControllerReference(agent, pvc, r.Scheme); err != nil {
			return err
		}

		// Only set spec on creation (PVCs are immutable after creation)
		if pvc.CreationTimestamp.IsZero() {
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(size),
					},
				},
			}

			if agent.Spec.Workspace.StorageClassName != nil {
				pvc.Spec.StorageClassName = agent.Spec.Workspace.StorageClassName
			}
		}

		return nil
	})

	return err
}

func (r *LanguageAgentReconciler) reconcileDeployment(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Resolve model URLs
	modelURLs, err := r.resolveModels(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve models: %w", err)
	}

	// Resolve tool URLs
	toolURLs, err := r.resolveTools(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve tools: %w", err)
	}

	// Determine target namespace and labels
	targetNamespace := agent.Namespace
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// If cluster ref is set, fetch cluster and use its namespace
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", agent.Spec.ClusterRef)
		}

		// Use cluster's namespace
		targetNamespace = cluster.Status.Namespace

		// Add cluster label
		labels["langop.io/cluster"] = agent.Spec.ClusterRef
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(agent, deployment, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		if agent.Spec.Replicas != nil {
			replicas = *agent.Spec.Replicas
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: agent.Spec.Image,
							Env:   r.buildAgentEnv(agent, modelURLs, toolURLs),
						},
					},
				},
			},
		}

		// Add resource requirements if specified
		deployment.Spec.Template.Spec.Containers[0].Resources = agent.Spec.Resources

		// Add workspace volume if enabled
		if agent.Spec.Workspace != nil && agent.Spec.Workspace.Enabled {
			mountPath := agent.Spec.Workspace.MountPath
			if mountPath == "" {
				mountPath = "/workspace"
			}

			deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: agent.Name + "-workspace",
						},
					},
				},
			}

			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: mountPath,
				},
			}
		}

		return nil
	})

	return err
}

func (r *LanguageAgentReconciler) reconcileCronJob(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Resolve model URLs
	modelURLs, err := r.resolveModels(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve models: %w", err)
	}

	// Resolve tool URLs
	toolURLs, err := r.resolveTools(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve tools: %w", err)
	}

	// Determine target namespace and labels
	targetNamespace := agent.Namespace
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// If cluster ref is set, fetch cluster and use its namespace
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", agent.Spec.ClusterRef)
		}

		// Use cluster's namespace
		targetNamespace = cluster.Status.Namespace

		// Add cluster label
		labels["langop.io/cluster"] = agent.Spec.ClusterRef
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cronJob, func() error {
		if err := controllerutil.SetControllerReference(agent, cronJob, r.Scheme); err != nil {
			return err
		}

		schedule := "0 * * * *" // Default: hourly
		if agent.Spec.Schedule != "" {
			schedule = agent.Spec.Schedule
		}

		cronJob.Spec = batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:  "agent",
									Image: agent.Spec.Image,
									Env:   r.buildAgentEnv(agent, modelURLs, toolURLs),
								},
							},
						},
					},
				},
			},
		}

		// Add resource requirements if specified
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources = agent.Spec.Resources

		// Add workspace volume if enabled
		if agent.Spec.Workspace != nil && agent.Spec.Workspace.Enabled {
			mountPath := agent.Spec.Workspace.MountPath
			if mountPath == "" {
				mountPath = "/workspace"
			}

			cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: agent.Name + "-workspace",
						},
					},
				},
			}

			cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: mountPath,
				},
			}
		}

		return nil
	})

	return err
}

func (r *LanguageAgentReconciler) resolveModels(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, error) {
	var modelURLs []string

	for _, modelRef := range agent.Spec.ModelRefs {
		// Determine namespace
		namespace := modelRef.Namespace
		if namespace == "" {
			namespace = agent.Namespace
		}

		// Fetch the LanguageModel
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: namespace}, model); err != nil {
			return nil, fmt.Errorf("failed to get model %s/%s: %w", namespace, modelRef.Name, err)
		}

		// Build LiteLLM proxy URL
		// Format: http://<service-name>.<namespace>.svc.cluster.local:<port>
		// TODO: Once LanguageModel controller creates Service, get actual port from service
		port := 8000 // Default LiteLLM port

		serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", model.Name, namespace, port)
		modelURLs = append(modelURLs, serviceURL)
	}

	return modelURLs, nil
}

func (r *LanguageAgentReconciler) resolveTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, error) {
	var toolURLs []string

	for _, toolRef := range agent.Spec.ToolRefs {
		// Determine namespace
		namespace := toolRef.Namespace
		if namespace == "" {
			namespace = agent.Namespace
		}

		// Fetch the LanguageTool
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", namespace, toolRef.Name, err)
		}

		// Skip sidecar tools (they'll be added as containers, not via URL)
		if tool.Spec.DeploymentMode == "sidecar" {
			continue
		}

		// Build MCP server URL (service mode)
		// Format: http://<service-name>.<namespace>.svc.cluster.local:<port>
		port := tool.Spec.Port
		if port == 0 {
			port = 8080 // Default MCP port
		}

		serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", tool.Name, namespace, port)
		toolURLs = append(toolURLs, serviceURL)
	}

	return toolURLs, nil
}

func (r *LanguageAgentReconciler) buildAgentEnv(agent *langopv1alpha1.LanguageAgent, modelURLs []string, toolURLs []string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "AGENT_NAME",
			Value: agent.Name,
		},
		{
			Name:  "AGENT_NAMESPACE",
			Value: agent.Namespace,
		},
		{
			Name:  "AGENT_MODE",
			Value: agent.Spec.ExecutionMode,
		},
	}

	if agent.Spec.Goal != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_GOAL",
			Value: agent.Spec.Goal,
		})
	}

	// Add LiteLLM model proxy URLs (comma-separated)
	if len(modelURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MODEL_ENDPOINTS",
			Value: strings.Join(modelURLs, ","),
		})
	}

	// Add MCP tool server URLs (comma-separated)
	if len(toolURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MCP_SERVERS",
			Value: strings.Join(toolURLs, ","),
		})
	}

	// Add environment variables from spec
	env = append(env, agent.Spec.Env...)

	return env
}

func (r *LanguageAgentReconciler) cleanupResources(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Resources will be cleaned up automatically via owner references
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&batchv1.CronJob{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
