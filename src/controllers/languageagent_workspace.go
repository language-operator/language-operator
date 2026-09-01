package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

func (r *LanguageAgentReconciler) reconcilePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Skip if workspace is not enabled
	if agent.Spec.Workspace == nil || (agent.Spec.Workspace.Enabled != nil && !*agent.Spec.Workspace.Enabled) {
		return nil
	}

	// Determine target namespace - always use agent's namespace
	targetNamespace := agent.Namespace
	if err := ValidateClusterReference(ctx, r.Client, agent.Namespace); err != nil {
		return err
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
			Name:      GeneratePVCName(agent.Name),
			Namespace: targetNamespace,
			Labels:    GetCommonLabels(agent.Name, "LanguageAgent"),
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, pvc, func() error {
		// Only set spec on creation (PVCs are immutable after creation)
		if pvc.CreationTimestamp.IsZero() {
			// Parse storage size safely to avoid controller panic
			quantity, err := resource.ParseQuantity(size)
			if err != nil {
				return fmt.Errorf("invalid workspace size %q: %w", size, err)
			}

			// Validate minimum size - PVCs cannot have zero storage
			if quantity.IsZero() {
				return fmt.Errorf("workspace size cannot be zero, got: %s", size)
			}

			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: quantity,
					},
				},
			}

			if agent.Spec.Workspace.StorageClassName != nil {
				pvc.Spec.StorageClassName = agent.Spec.Workspace.StorageClassName
			} else if r.DefaultStorageClassName != "" {
				pvc.Spec.StorageClassName = &r.DefaultStorageClassName
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if agent.Spec.Workspace.Retain != nil && *agent.Spec.Workspace.Retain {
		agent.Status.WorkspacePVCName = GeneratePVCName(agent.Name)
	}
	return nil
}

// orphanWorkspacePVC removes the ownerReference from the workspace PVC so that
// Kubernetes GC does not delete it when the LanguageAgent is deleted.
func (r *LanguageAgentReconciler) orphanWorkspacePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: GeneratePVCName(agent.Name), Namespace: agent.Namespace}, pvc)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get workspace PVC: %w", err)
	}
	patch := client.MergeFrom(pvc.DeepCopy())
	pvc.OwnerReferences = nil
	if err := r.Patch(ctx, pvc, patch); err != nil {
		return fmt.Errorf("failed to patch workspace PVC owner refs: %w", err)
	}
	log.FromContext(ctx).Info("Retained workspace PVC", "pvc", pvc.Name)
	return nil
}
