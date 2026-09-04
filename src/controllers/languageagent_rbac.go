package controllers

import (
	"context"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
)

// reconcileAgentServiceAccount ensures the ServiceAccount for agent pods exists with proper permissions
func (r *LanguageAgentReconciler) reconcileAgentServiceAccount(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Skip if custom ServiceAccount is specified - assume it exists and has proper permissions
	if agent.Spec.Deployment.ServiceAccountName != "" {
		return nil
	}

	// ServiceAccount always lives in the agent's own namespace
	targetNamespace := agent.Namespace
	saName := GenerateServiceAccountName(agent.Name)

	// Create ServiceAccount
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: targetNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}
		serviceAccount.Labels[langoplabels.LabelKeyK8sName] = saName
		serviceAccount.Labels[langoplabels.LabelKeyK8sComponent] = "serviceaccount"
		serviceAccount.Labels[langoplabels.LabelKeyK8sManagedBy] = "language-operator"

		// Merge user-supplied annotations (e.g. IRSA, GCP WI, AKS WI)
		if len(agent.Spec.Deployment.ServiceAccountAnnotations) > 0 {
			if serviceAccount.Annotations == nil {
				serviceAccount.Annotations = make(map[string]string)
			}
			maps.Copy(serviceAccount.Annotations, agent.Spec.Deployment.ServiceAccountAnnotations)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update ServiceAccount: %w", err)
	}

	// Create namespace-scoped Role with minimal permissions for agent pods
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: targetNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if role.Labels == nil {
			role.Labels = make(map[string]string)
		}
		role.Labels[langoplabels.LabelKeyK8sName] = saName
		role.Labels[langoplabels.LabelKeyK8sComponent] = "role"
		role.Labels[langoplabels.LabelKeyK8sManagedBy] = "language-operator"
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			// Agents run as Argo Workflow pods. The Argo executor reports each
			// node's outcome through a WorkflowTaskResult written with the pod's
			// own ServiceAccount — without this the run fails at completion.
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"workflowtaskresults"},
				Verbs:     []string{"create", "patch"},
			},
		}
		// When self-configure is enabled, grant the agent's SA permission to
		// create LanguageAgentSelfConfig requests targeting itself.
		if agent.Spec.SelfConfigure != nil && agent.Spec.SelfConfigure.Enabled != nil && *agent.Spec.SelfConfigure.Enabled {
			role.Rules = append(role.Rules, rbacv1.PolicyRule{
				APIGroups: []string{"langop.io"},
				Resources: []string{"languageagentselfconfigs"},
				Verbs:     []string{"create"},
			})
		}
		role.Rules = append(role.Rules, agent.Spec.Deployment.RoleRules...)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update Role: %w", err)
	}

	// Create namespace-scoped RoleBinding binding the ServiceAccount to the Role
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: targetNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}
		roleBinding.Labels[langoplabels.LabelKeyK8sName] = saName
		roleBinding.Labels[langoplabels.LabelKeyK8sComponent] = "rolebinding"
		roleBinding.Labels[langoplabels.LabelKeyK8sManagedBy] = "language-operator"
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     saName,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: targetNamespace,
			},
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update RoleBinding: %w", err)
	}

	log.Info("Reconciled agent ServiceAccount and permissions",
		"serviceAccount", saName,
		"namespace", targetNamespace,
		"role", role.Name,
		"roleBinding", roleBinding.Name)

	return nil
}

// getServiceAccountName returns the ServiceAccount name to use for agent pods
func (r *LanguageAgentReconciler) getServiceAccountName(agent *langopv1alpha1.LanguageAgent) string {
	// If explicitly specified in the agent spec, use that
	if agent.Spec.Deployment.ServiceAccountName != "" {
		return agent.Spec.Deployment.ServiceAccountName
	}

	// Default to an operator-managed per-agent ServiceAccount
	return GenerateServiceAccountName(agent.Name)
}

// cleanupPerAgentRBAC deletes the per-agent ServiceAccount, Role, and RoleBinding.
// These are not covered by owner-reference GC because ServiceAccounts cannot be
// owner-referenced from Pods across namespaces, so they must be deleted explicitly.
// Skipped when a custom ServiceAccountName is set (user manages their own SA).
func (r *LanguageAgentReconciler) cleanupPerAgentRBAC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	if agent.Spec.Deployment.ServiceAccountName != "" {
		return nil
	}
	log := log.FromContext(ctx)
	ns := agent.Namespace
	saName := GenerateServiceAccountName(agent.Name)

	toDelete := []client.Object{
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns}},
	}
	for _, obj := range toDelete {
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %T %s: %w", obj, saName, err)
		}
		log.Info("Deleted per-agent RBAC resource", "kind", fmt.Sprintf("%T", obj), "name", saName, "namespace", ns)
	}
	return nil
}
