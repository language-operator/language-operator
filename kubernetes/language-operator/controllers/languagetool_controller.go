package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguageToolReconciler reconciles a LanguageTool object
type LanguageToolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languagetools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagetools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagetools/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageToolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LanguageTool instance
	tool := &langopv1alpha1.LanguageTool{}
	if err := r.Get(ctx, req.NamespacedName, tool); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageTool")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !tool.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(tool, FinalizerName) {
			// Perform cleanup
			if err := r.cleanupResources(ctx, tool); err != nil {
				return ctrl.Result{}, err
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(tool, FinalizerName)
			if err := r.Update(ctx, tool); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(tool, FinalizerName) {
		controllerutil.AddFinalizer(tool, FinalizerName)
		if err := r.Update(ctx, tool); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, tool); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "ConfigMapError", err.Error(), tool.Generation)
		r.Status().Update(ctx, tool)
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, tool); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), tool.Generation)
		r.Status().Update(ctx, tool)
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, tool); err != nil {
		log.Error(err, "Failed to reconcile Service")
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "ServiceError", err.Error(), tool.Generation)
		r.Status().Update(ctx, tool)
		return ctrl.Result{}, err
	}

	// Reconcile NetworkPolicy for network isolation
	if err := r.reconcileNetworkPolicy(ctx, tool); err != nil {
		log.Error(err, "Failed to reconcile NetworkPolicy")
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "NetworkPolicyError", err.Error(), tool.Generation)
		r.Status().Update(ctx, tool)
		return ctrl.Result{}, err
	}

	// Update status
	tool.Status.Phase = "Ready"
	SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageTool is ready", tool.Generation)

	if err := r.Status().Update(ctx, tool); err != nil {
		log.Error(err, "Failed to update LanguageTool status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LanguageToolReconciler) reconcileConfigMap(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	data := make(map[string]string)

	// Add tool spec as JSON
	specJSON, err := json.Marshal(tool.Spec)
	if err != nil {
		return err
	}
	data["tool.json"] = string(specJSON)

	// Add other useful data
	data["name"] = tool.Name
	data["namespace"] = tool.Namespace
	data["type"] = string(tool.Spec.Type)

	configMapName := GenerateConfigMapName(tool.Name, "tool")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, tool, configMapName, tool.Namespace, data)
}

func (r *LanguageToolReconciler) reconcileDeployment(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Determine target namespace and labels
	targetNamespace := tool.Namespace
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	// If cluster ref is set, fetch cluster and use its namespace
	if tool.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: tool.Spec.ClusterRef}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			// Return error to trigger requeue
			return fmt.Errorf("cluster %s is not ready yet", tool.Spec.ClusterRef)
		}

		// Use cluster's namespace
		targetNamespace = cluster.Status.Namespace

		// Add cluster label
		labels["langop.io/cluster"] = tool.Spec.ClusterRef
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(tool, deployment, r.Scheme); err != nil {
			return err
		}

		// Set deployment spec
		replicas := int32(1)
		if tool.Spec.Replicas != nil {
			replicas = *tool.Spec.Replicas
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
							Name:  "tool",
							Image: tool.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: tool.Spec.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: tool.Spec.Env,
						},
					},
				},
			},
		}

		// Add resource requirements if specified
		deployment.Spec.Template.Spec.Containers[0].Resources = tool.Spec.Resources

		// Add affinity if specified
		if tool.Spec.Affinity != nil {
			deployment.Spec.Template.Spec.Affinity = tool.Spec.Affinity
		}

		return nil
	})

	return err
}

func (r *LanguageToolReconciler) reconcileService(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Determine target namespace and labels (same logic as deployment)
	targetNamespace := tool.Namespace
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	// If cluster ref is set, fetch cluster and use its namespace
	if tool.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: tool.Spec.ClusterRef}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", tool.Spec.ClusterRef)
		}

		// Use cluster's namespace
		targetNamespace = cluster.Status.Namespace

		// Add cluster label
		labels["langop.io/cluster"] = tool.Spec.ClusterRef
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(tool, service, r.Scheme); err != nil {
			return err
		}

		// Set service spec
		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       tool.Spec.Port,
					TargetPort: intstr.FromInt(int(tool.Spec.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		}

		return nil
	})

	return err
}

func (r *LanguageToolReconciler) reconcileNetworkPolicy(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	// Build NetworkPolicy using helper from utils.go
	networkPolicy := BuildEgressNetworkPolicy(
		tool.Name,
		tool.Namespace,
		labels,
		tool.Spec.Egress,
	)

	// Set owner reference so NetworkPolicy is cleaned up with tool
	if err := controllerutil.SetControllerReference(tool, networkPolicy, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update the NetworkPolicy
	existingPolicy := &networkingv1.NetworkPolicy{}
	err := r.Get(ctx, types.NamespacedName{Name: networkPolicy.Name, Namespace: networkPolicy.Namespace}, existingPolicy)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create new NetworkPolicy
			if err := r.Create(ctx, networkPolicy); err != nil {
				return fmt.Errorf("failed to create NetworkPolicy: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	// Update existing NetworkPolicy
	existingPolicy.Spec = networkPolicy.Spec
	existingPolicy.Labels = networkPolicy.Labels
	if err := r.Update(ctx, existingPolicy); err != nil {
		return fmt.Errorf("failed to update NetworkPolicy: %w", err)
	}

	return nil
}

func (r *LanguageToolReconciler) cleanupResources(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Resources will be cleaned up automatically via owner references
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageToolReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageTool{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
