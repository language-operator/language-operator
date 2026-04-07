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
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	langoplabels "github.com/language-operator/language-operator/pkg/labels"
)

// CreateOrUpdateOwned wraps controllerutil.CreateOrUpdate, setting the controller reference
// on obj before invoking mutateFn. Callers only need to provide spec-level mutations.
func CreateOrUpdateOwned(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner, obj client.Object,
	mutateFn func() error,
) error {
	_, err := controllerutil.CreateOrUpdate(ctx, c, obj, func() error {
		if err := controllerutil.SetControllerReference(owner, obj, scheme); err != nil {
			return err
		}
		return mutateFn()
	})
	return err
}

// CreateOrUpdateConfigMap creates or updates a ConfigMap with owner reference.
func CreateOrUpdateConfigMap(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	name, namespace string,
	data map[string]string,
) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, configMap, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(owner, configMap, scheme); err != nil {
			return err
		}

		// Update data
		configMap.Data = data

		return nil
	})

	return err
}

// DeleteConfigMap deletes a ConfigMap if it exists.
func DeleteConfigMap(ctx context.Context, c client.Client, name, namespace string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := c.Delete(ctx, configMap)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// GenerateConfigMapName generates a ConfigMap name for a resource.
func GenerateConfigMapName(resourceName, suffix string) string {
	return fmt.Sprintf("%s-%s", resourceName, suffix)
}

// GenerateServiceAccountName returns the name of the operator-managed ServiceAccount for an agent.
func GenerateServiceAccountName(agentName string) string {
	return "language-agent-" + agentName
}

// GeneratePVCName returns the PVC name for an agent's workspace volume.
func GeneratePVCName(agentName string) string {
	return agentName + "-workspace"
}

// GenerateTLSSecretName returns the TLS secret name for an agent.
func GenerateTLSSecretName(agentName string) string {
	return agentName + "-tls"
}

// certManagerIssuerAnnotationSuffix returns the cert-manager annotation suffix for a given issuer kind.
// cert-manager uses "issuer" for Issuer and "cluster-issuer" (hyphenated) for ClusterIssuer.
func certManagerIssuerAnnotationSuffix(kind string) string {
	if strings.EqualFold(kind, "ClusterIssuer") {
		return "cluster-issuer"
	}
	return "issuer"
}

// GetCommonLabels returns common labels for resources.
func GetCommonLabels(resourceName, resourceKind string) map[string]string {
	return map[string]string{
		langoplabels.LabelKeyK8sName:      resourceName,
		langoplabels.LabelKeyK8sManagedBy: "language-operator",
		langoplabels.LabelKeyK8sPartOf:    "langop",
		langoplabels.LabelKeyLangopKind:   resourceKind,
	}
}
