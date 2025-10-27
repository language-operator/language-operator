package controllers

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguageClientReconciler reconciles a LanguageClient object
type LanguageClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languageclients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageclients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageclients/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LanguageClient instance
	lc := &langopv1alpha1.LanguageClient{}
	if err := r.Get(ctx, req.NamespacedName, lc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageClient")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !lc.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(lc, FinalizerName) {
			if err := r.cleanupResources(ctx, lc); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(lc, FinalizerName)
			if err := r.Update(ctx, lc); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(lc, FinalizerName) {
		controllerutil.AddFinalizer(lc, FinalizerName)
		if err := r.Update(ctx, lc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, lc); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		SetCondition(&lc.Status.Conditions, "Ready", metav1.ConditionFalse, "ConfigMapError", err.Error(), lc.Generation)
		r.Status().Update(ctx, lc)
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, lc); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		SetCondition(&lc.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), lc.Generation)
		r.Status().Update(ctx, lc)
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, lc); err != nil {
		log.Error(err, "Failed to reconcile Service")
		SetCondition(&lc.Status.Conditions, "Ready", metav1.ConditionFalse, "ServiceError", err.Error(), lc.Generation)
		r.Status().Update(ctx, lc)
		return ctrl.Result{}, err
	}

	// TODO: Reconcile Ingress (if ingress is configured)
	// The Ingress reconciliation is complex and requires mapping the custom IngressSpec
	// to Kubernetes Ingress resources. This will be implemented in a future update.

	// Update status
	lc.Status.Phase = "Ready"
	SetCondition(&lc.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageClient is ready", lc.Generation)

	if err := r.Status().Update(ctx, lc); err != nil {
		log.Error(err, "Failed to update LanguageClient status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LanguageClientReconciler) reconcileConfigMap(ctx context.Context, lc *langopv1alpha1.LanguageClient) error {
	data := make(map[string]string)

	// Add client spec as JSON
	specJSON, err := json.Marshal(lc.Spec)
	if err != nil {
		return err
	}
	data["client.json"] = string(specJSON)

	// Add other useful data
	data["name"] = lc.Name
	data["namespace"] = lc.Namespace
	data["type"] = string(lc.Spec.Type)

	// Add model references as JSON
	if len(lc.Spec.ModelRefs) > 0 {
		modelRefsJSON, err := json.Marshal(lc.Spec.ModelRefs)
		if err != nil {
			return err
		}
		data["model-refs.json"] = string(modelRefsJSON)
	}

	// Add tool references as JSON
	if len(lc.Spec.ToolRefs) > 0 {
		toolRefsJSON, err := json.Marshal(lc.Spec.ToolRefs)
		if err != nil {
			return err
		}
		data["tool-refs.json"] = string(toolRefsJSON)
	}

	configMapName := GenerateConfigMapName(lc.Name, "client")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, lc, configMapName, lc.Namespace, data)
}

func (r *LanguageClientReconciler) reconcileDeployment(ctx context.Context, lc *langopv1alpha1.LanguageClient) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lc.Name,
			Namespace: lc.Namespace,
			Labels:    GetCommonLabels(lc.Name, "LanguageClient"),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(lc, deployment, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		if lc.Spec.Replicas != nil {
			replicas = *lc.Spec.Replicas
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GetCommonLabels(lc.Name, "LanguageClient"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: GetCommonLabels(lc.Name, "LanguageClient"),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "client",
							Image: lc.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: lc.Spec.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: r.buildClientEnv(lc),
						},
					},
				},
			},
		}

		// Add resource requirements if specified
		deployment.Spec.Template.Spec.Containers[0].Resources = lc.Spec.Resources

		// Add affinity if specified
		if lc.Spec.Affinity != nil {
			deployment.Spec.Template.Spec.Affinity = lc.Spec.Affinity
		}

		return nil
	})

	return err
}

func (r *LanguageClientReconciler) reconcileService(ctx context.Context, lc *langopv1alpha1.LanguageClient) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lc.Name,
			Namespace: lc.Namespace,
			Labels:    GetCommonLabels(lc.Name, "LanguageClient"),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if err := controllerutil.SetControllerReference(lc, service, r.Scheme); err != nil {
			return err
		}

		serviceType := corev1.ServiceTypeClusterIP
		if lc.Spec.ServiceType != "" {
			serviceType = lc.Spec.ServiceType
		}

		service.Spec = corev1.ServiceSpec{
			Selector: GetCommonLabels(lc.Name, "LanguageClient"),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       lc.Spec.Port,
					TargetPort: intstr.FromInt(int(lc.Spec.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: serviceType,
		}

		// Add service annotations if specified
		if lc.Spec.ServiceAnnotations != nil {
			service.Annotations = lc.Spec.ServiceAnnotations
		}

		return nil
	})

	return err
}

func (r *LanguageClientReconciler) buildClientEnv(lc *langopv1alpha1.LanguageClient) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "CLIENT_NAME",
			Value: lc.Name,
		},
		{
			Name:  "CLIENT_NAMESPACE",
			Value: lc.Namespace,
		},
		{
			Name:  "CLIENT_TYPE",
			Value: string(lc.Spec.Type),
		},
	}

	// Add environment variables from spec
	env = append(env, lc.Spec.Env...)

	return env
}

func (r *LanguageClientReconciler) cleanupResources(ctx context.Context, lc *langopv1alpha1.LanguageClient) error {
	// Resources will be cleaned up automatically via owner references
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClientReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageClient{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
