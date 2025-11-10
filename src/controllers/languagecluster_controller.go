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

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguageClusterReconciler reconciles a LanguageCluster object
type LanguageClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// tracer is the OpenTelemetry tracer for the LanguageCluster controller
var clusterTracer = otel.Tracer("language-operator/cluster-controller")

//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Start OpenTelemetry span for reconciliation
	ctx, span := clusterTracer.Start(ctx, "cluster.reconcile")
	defer span.End()

	// Add basic span attributes from request
	span.SetAttributes(
		attribute.String("cluster.name", req.Name),
		attribute.String("cluster.namespace", req.Namespace),
	)

	log := log.FromContext(ctx)

	// Fetch LanguageCluster
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if client.IgnoreNotFound(err) == nil {
			span.SetStatus(codes.Ok, "Resource not found (deleted)")
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get LanguageCluster")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add cluster-specific attributes to span
	span.SetAttributes(
		attribute.Int64("cluster.generation", cluster.Generation),
	)

	// Reconcile dashboard if enabled
	if cluster.Spec.Dashboard != nil && cluster.Spec.Dashboard.Enabled {
		if err := r.reconcileDashboard(ctx, cluster); err != nil {
			log.Error(err, "Failed to reconcile dashboard")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile dashboard")
			SetCondition(&cluster.Status.Conditions, "DashboardReady", metav1.ConditionFalse,
				"DashboardError", err.Error(), cluster.Generation)
			cluster.Status.Phase = "Degraded"
			if statusErr := r.Status().Update(ctx, cluster); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{}, err
		}
		SetCondition(&cluster.Status.Conditions, "DashboardReady", metav1.ConditionTrue,
			"DashboardRunning", "Dashboard is deployed and running", cluster.Generation)
	}

	// LanguageCluster is now just a logical grouping - no namespace management
	// Child resources reference the cluster and live in the same namespace
	cluster.Status.Phase = "Ready"
	SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionTrue,
		"ReconcileSuccess", "LanguageCluster is ready", cluster.Generation)

	if err := r.Status().Update(ctx, cluster); err != nil {
		log.Error(err, "Failed to update status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		return ctrl.Result{}, err
	}

	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClusterReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

// reconcileDashboard creates/updates dashboard resources
func (r *LanguageClusterReconciler) reconcileDashboard(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	// Start child span for dashboard reconciliation
	ctx, span := clusterTracer.Start(ctx, "cluster.dashboard.reconcile")
	defer span.End()

	span.SetAttributes(
		attribute.String("cluster.name", cluster.Name),
		attribute.String("cluster.namespace", cluster.Namespace),
	)

	// Create ServiceAccount
	if err := r.reconcileDashboardServiceAccount(ctx, cluster); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile ServiceAccount")
		return fmt.Errorf("failed to reconcile ServiceAccount: %w", err)
	}

	// Create Role
	if err := r.reconcileDashboardRole(ctx, cluster); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile Role")
		return fmt.Errorf("failed to reconcile Role: %w", err)
	}

	// Create RoleBinding
	if err := r.reconcileDashboardRoleBinding(ctx, cluster); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile RoleBinding")
		return fmt.Errorf("failed to reconcile RoleBinding: %w", err)
	}

	// Create Deployment
	if err := r.reconcileDashboardDeployment(ctx, cluster); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile Deployment")
		return fmt.Errorf("failed to reconcile Deployment: %w", err)
	}

	// Create Service
	if err := r.reconcileDashboardService(ctx, cluster); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile Service")
		return fmt.Errorf("failed to reconcile Service: %w", err)
	}

	span.SetStatus(codes.Ok, "Dashboard reconciled successfully")
	return nil
}

// reconcileDashboardServiceAccount creates the ServiceAccount for the dashboard
func (r *LanguageClusterReconciler) reconcileDashboardServiceAccount(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	name := cluster.Name + "-dashboard"
	labels := map[string]string{
		"app.kubernetes.io/name":       "language-operator",
		"app.kubernetes.io/component":  "dashboard",
		"app.kubernetes.io/instance":   cluster.Name,
		"app.kubernetes.io/managed-by": "language-operator",
		"langop.io/cluster":            cluster.Name,
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		if err := controllerutil.SetControllerReference(cluster, sa, r.Scheme); err != nil {
			return err
		}
		sa.Labels = labels
		return nil
	})

	return err
}

// reconcileDashboardRole creates the Role for the dashboard
func (r *LanguageClusterReconciler) reconcileDashboardRole(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	name := cluster.Name + "-dashboard"
	labels := map[string]string{
		"app.kubernetes.io/name":       "language-operator",
		"app.kubernetes.io/component":  "dashboard",
		"app.kubernetes.io/instance":   cluster.Name,
		"app.kubernetes.io/managed-by": "language-operator",
		"langop.io/cluster":            cluster.Name,
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if err := controllerutil.SetControllerReference(cluster, role, r.Scheme); err != nil {
			return err
		}
		role.Labels = labels
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"langop.io"},
				Resources: []string{"languageagents", "languagetools", "languagemodels"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/log"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}
		return nil
	})

	return err
}

// reconcileDashboardRoleBinding creates the RoleBinding for the dashboard
func (r *LanguageClusterReconciler) reconcileDashboardRoleBinding(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	name := cluster.Name + "-dashboard"
	labels := map[string]string{
		"app.kubernetes.io/name":       "language-operator",
		"app.kubernetes.io/component":  "dashboard",
		"app.kubernetes.io/instance":   cluster.Name,
		"app.kubernetes.io/managed-by": "language-operator",
		"langop.io/cluster":            cluster.Name,
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
		if err := controllerutil.SetControllerReference(cluster, rb, r.Scheme); err != nil {
			return err
		}
		rb.Labels = labels
		rb.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: cluster.Namespace,
			},
		}
		return nil
	})

	return err
}

// reconcileDashboardDeployment creates the Deployment for the dashboard
func (r *LanguageClusterReconciler) reconcileDashboardDeployment(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	name := cluster.Name + "-dashboard"
	labels := map[string]string{
		"app.kubernetes.io/name":       "language-operator",
		"app.kubernetes.io/component":  "dashboard",
		"app.kubernetes.io/instance":   cluster.Name,
		"app.kubernetes.io/managed-by": "language-operator",
		"langop.io/cluster":            cluster.Name,
	}

	// Determine image
	image := "git.theryans.io/language-operator/dashboard:latest"
	if cluster.Spec.Dashboard.Image != "" {
		image = cluster.Spec.Dashboard.Image
	}

	// Determine port
	port := int32(8080)
	if cluster.Spec.Dashboard.Port != 0 {
		port = cluster.Spec.Dashboard.Port
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(cluster, deployment, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
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
					ServiceAccountName: name,
					Containers: []corev1.Container{
						{
							Name:  "dashboard",
							Image: image,
							Env: []corev1.EnvVar{
								{
									Name:  "CLUSTER_NAME",
									Value: cluster.Name,
								},
								{
									Name:  "NAMESPACE",
									Value: cluster.Namespace,
								},
								{
									Name:  "PORT",
									Value: fmt.Sprintf("%d", port),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								RunAsNonRoot:             ptr.To(true),
								RunAsUser:                ptr.To[int64](1000),
								ReadOnlyRootFilesystem:   ptr.To(true),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						RunAsUser:    ptr.To[int64](1000),
						FSGroup:      ptr.To[int64](101),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			},
		}

		return nil
	})

	return err
}

// reconcileDashboardService creates the Service for the dashboard
func (r *LanguageClusterReconciler) reconcileDashboardService(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	name := cluster.Name + "-dashboard"
	labels := map[string]string{
		"app.kubernetes.io/name":       "language-operator",
		"app.kubernetes.io/component":  "dashboard",
		"app.kubernetes.io/instance":   cluster.Name,
		"app.kubernetes.io/managed-by": "language-operator",
		"langop.io/cluster":            cluster.Name,
	}

	// Determine port
	port := int32(8080)
	if cluster.Spec.Dashboard.Port != 0 {
		port = cluster.Spec.Dashboard.Port
	}

	// Determine service type
	serviceType := corev1.ServiceTypeClusterIP
	if cluster.Spec.Dashboard.ServiceType != "" {
		serviceType = corev1.ServiceType(cluster.Spec.Dashboard.ServiceType)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if err := controllerutil.SetControllerReference(cluster, service, r.Scheme); err != nil {
			return err
		}

		service.Spec = corev1.ServiceSpec{
			Type:     serviceType,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(int(port)),
				},
			},
		}

		return nil
	})

	return err
}
