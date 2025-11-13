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
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguageModelReconciler reconciles a LanguageModel object
type LanguageModelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// tracer is the OpenTelemetry tracer for the LanguageModel controller
var modelTracer = otel.Tracer("language-operator/model-controller")

//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles a LanguageModel resource
func (r *LanguageModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Start OpenTelemetry span for reconciliation
	ctx, span := modelTracer.Start(ctx, "model.reconcile")
	defer span.End()

	// Add basic span attributes from request
	span.SetAttributes(
		attribute.String("model.name", req.Name),
		attribute.String("model.namespace", req.Namespace),
	)

	log := log.FromContext(ctx)

	// Fetch the LanguageModel instance
	model := &langopv1alpha1.LanguageModel{}
	if err := r.Get(ctx, req.NamespacedName, model); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("LanguageModel resource not found. Ignoring since object must be deleted")
			span.SetStatus(codes.Ok, "Resource not found (deleted)")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageModel")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get LanguageModel")
		return ctrl.Result{}, err
	}

	// Add model-specific attributes to span
	span.SetAttributes(
		attribute.String("model.provider", model.Spec.Provider),
		attribute.Int64("model.generation", model.Generation),
	)

	// Handle deletion
	if !model.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, model)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(model, FinalizerName) {
		controllerutil.AddFinalizer(model, FinalizerName)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the ConfigMap
	if err := r.reconcileConfigMap(ctx, model); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile ConfigMap")
		SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionFalse, "ReconcileError", err.Error(), model.Generation)
		model.Status.Phase = "Failed"
		if statusErr := r.Status().Update(ctx, model); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Reconcile the Deployment
	if err := r.reconcileDeployment(ctx, model); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile Deployment")
		SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), model.Generation)
		model.Status.Phase = "Failed"
		if statusErr := r.Status().Update(ctx, model); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Reconcile the Service
	if err := r.reconcileService(ctx, model); err != nil {
		log.Error(err, "Failed to reconcile Service")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile Service")
		SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionFalse, "ServiceError", err.Error(), model.Generation)
		model.Status.Phase = "Failed"
		if statusErr := r.Status().Update(ctx, model); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Reconcile NetworkPolicy for network isolation
	if err := r.reconcileNetworkPolicy(ctx, model); err != nil {
		log.Error(err, "Failed to reconcile NetworkPolicy")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile NetworkPolicy")
		SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionFalse, "NetworkPolicyError", err.Error(), model.Generation)
		model.Status.Phase = "Failed"
		if statusErr := r.Status().Update(ctx, model); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Update status
	model.Status.ObservedGeneration = model.Generation
	model.Status.Phase = "Ready"
	// Status fields updated
	SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "Model proxy is ready", model.Generation)

	if err := r.Status().Update(ctx, model); err != nil {
		log.Error(err, "Failed to update status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LanguageModel")
	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

// reconcileConfigMap creates or updates the ConfigMap for the model
func (r *LanguageModelReconciler) reconcileConfigMap(ctx context.Context, model *langopv1alpha1.LanguageModel) error {
	// Create ConfigMap data from model spec
	data := make(map[string]string)

	// Serialize the spec as JSON
	specJSON, err := json.Marshal(model.Spec)
	if err != nil {
		return err
	}
	data["model.json"] = string(specJSON)

	// Add individual fields for easy access
	data["provider"] = model.Spec.Provider
	data["modelName"] = model.Spec.ModelName
	if model.Spec.Endpoint != "" {
		data["endpoint"] = model.Spec.Endpoint
	}
	if model.Spec.Timeout != "" {
		data["timeout"] = model.Spec.Timeout
	}

	// Add API key secret reference info (not the actual secret)
	if model.Spec.APIKeySecretRef != nil {
		secretRefJSON, err := json.Marshal(model.Spec.APIKeySecretRef)
		if err != nil {
			return err
		}
		data["apiKeySecretRef.json"] = string(secretRefJSON)
	}

	// Add rate limits if specified
	if model.Spec.RateLimits != nil {
		rateLimitsJSON, err := json.Marshal(model.Spec.RateLimits)
		if err != nil {
			return err
		}
		data["rateLimits.json"] = string(rateLimitsJSON)
	}

	// Add fallbacks if specified
	if len(model.Spec.Fallbacks) > 0 {
		fallbacksJSON, err := json.Marshal(model.Spec.Fallbacks)
		if err != nil {
			return err
		}
		data["fallbacks.json"] = string(fallbacksJSON)
	}

	// Create or update the ConfigMap
	configMapName := GenerateConfigMapName(model.Name, "model")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, model, configMapName, model.Namespace, data)
}

// reconcileDeployment creates or updates the LiteLLM proxy Deployment
func (r *LanguageModelReconciler) reconcileDeployment(ctx context.Context, model *langopv1alpha1.LanguageModel) error {
	// Start child span for deployment creation
	ctx, span := modelTracer.Start(ctx, "model.deployment.create")
	defer span.End()

	span.SetAttributes(
		attribute.String("model.name", model.Name),
		attribute.String("model.namespace", model.Namespace),
	)

	labels := GetCommonLabels(model.Name, "LanguageModel")
	configMapName := GenerateConfigMapName(model.Name, "model")

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      model.Name,
			Namespace: model.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(model, deployment, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1) // Default to 1 replica

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
							Name:  "proxy",
							Image: "git.theryans.io/language-operator/model:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 4000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/langop",
									ReadOnly:  true,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(4000),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      30,
								PeriodSeconds:       300, // 5 minutes
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(4000),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      30,
								PeriodSeconds:       300, // 5 minutes
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
				},
			},
		}

		// Mount API key secret if specified
		if model.Spec.APIKeySecretRef != nil {
			secretName := model.Spec.APIKeySecretRef.Name
			secretKey := model.Spec.APIKeySecretRef.Key
			if secretKey == "" {
				secretKey = "api-key"
			}

			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: "secrets",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName,
						Items: []corev1.KeyToPath{
							{
								Key:  secretKey,
								Path: fmt.Sprintf("%s/%s", secretName, secretKey),
							},
						},
					},
				},
			})

			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
				corev1.VolumeMount{
					Name:      "secrets",
					MountPath: "/etc/secrets",
					ReadOnly:  true,
				},
			)
		}

		// TODO: Add resource requirements if Resources field is added to LanguageModelSpec

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create/update deployment")
		return err
	}

	span.SetStatus(codes.Ok, "Deployment reconciled successfully")
	return nil
}

// reconcileService creates or updates the Service for the LiteLLM proxy
func (r *LanguageModelReconciler) reconcileService(ctx context.Context, model *langopv1alpha1.LanguageModel) error {
	labels := GetCommonLabels(model.Name, "LanguageModel")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      model.Name,
			Namespace: model.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if err := controllerutil.SetControllerReference(model, service, r.Scheme); err != nil {
			return err
		}

		port := int32(8000) // Default port for model proxy

		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       port,
					TargetPort: intstr.FromInt(4000),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		}

		return nil
	})

	return err
}

func (r *LanguageModelReconciler) reconcileNetworkPolicy(ctx context.Context, model *langopv1alpha1.LanguageModel) error {
	labels := GetCommonLabels(model.Name, "LanguageModel")

	// Build NetworkPolicy using helper from utils.go
	networkPolicy := BuildEgressNetworkPolicy(
		model.Name,
		model.Namespace,
		labels,
		model.Spec.Provider,
		model.Spec.Endpoint,
		model.Spec.Egress,
	)

	// Add Ingress rules to allow agents to connect to the model service
	// Only allow pods labeled as LanguageAgents to connect
	networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	networkPolicy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			// Allow from LanguageAgent pods only
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"langop.io/kind": "LanguageAgent",
						},
					},
				},
			},
		},
	}

	// Create or update the NetworkPolicy with owner reference
	return CreateOrUpdateNetworkPolicy(ctx, r.Client, r.Scheme, model, networkPolicy)
}

// handleDeletion handles the deletion of the LanguageModel
func (r *LanguageModelReconciler) handleDeletion(ctx context.Context, model *langopv1alpha1.LanguageModel) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(model, FinalizerName) {
		// Delete the ConfigMap
		configMapName := GenerateConfigMapName(model.Name, "model")
		if err := DeleteConfigMap(ctx, r.Client, configMapName, model.Namespace); err != nil {
			log.Error(err, "Failed to delete ConfigMap")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(model, FinalizerName)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *LanguageModelReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageModel{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
