package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// reconcileGatewayKey issues the agent its gateway key: derived from the cluster's
// HMAC secret and the agent's UID, stored in the agent's own Secret, and read by
// the containers as MODEL_API_KEY (see agentGatewayKeyEnv). The gateway rejects
// calls without a valid key, and attributes each call to the agent's UID.
//
// It returns a hash of the key, to fold into the agent's config hash so a rotated
// HMAC secret replaces running Workflows, and pending=true when the cluster's
// secret does not exist yet (a new cluster still reconciling), in which case the
// caller requeues rather than fails.
func (r *LanguageAgentReconciler) reconcileGatewayKey(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (keyHash string, pending bool, err error) {
	if agent.UID == "" {
		return "", false, nil
	}
	authSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: GatewayAuthSecretName, Namespace: agent.Namespace}, authSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return "", true, nil
		}
		return "", false, fmt.Errorf("failed to read gateway auth secret: %w", err)
	}
	hmacSecret := string(authSecret.Data[GatewayAuthSecretKey])
	if hmacSecret == "" {
		return "", true, nil
	}

	key := gatewayAgentKey(hmacSecret, string(agent.UID))
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      gatewayAgentKeySecretName(agent.Name),
		Namespace: agent.Namespace,
	}}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, secret, func() error {
		secret.Data = map[string][]byte{ModelAPIKeyEnv: []byte(key)}
		return nil
	}); err != nil {
		return "", false, fmt.Errorf("failed to reconcile gateway key secret: %w", err)
	}
	return hashString(key)[:16], false, nil
}

// agentGatewayKeyEnv reads MODEL_API_KEY from the agent's key Secret. Optional, so
// a pod rendered before the key exists still starts; issuing the key changes the
// config hash, which replaces the pod with one that has it.
func agentGatewayKeyEnv(agent *langopv1alpha1.LanguageAgent) corev1.EnvVar {
	return corev1.EnvVar{
		Name: ModelAPIKeyEnv,
		ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: gatewayAgentKeySecretName(agent.Name)},
			Key:                  ModelAPIKeyEnv,
			Optional:             ptr.To(true),
		}},
	}
}
