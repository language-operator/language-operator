/*
Copyright 2025 Based Team.

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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	FinalizerName = "langop.io/finalizer"
)

// SetCondition updates or adds a condition to the conditions slice
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string, generation int64) {
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	for i, existing := range *conditions {
		if existing.Type == conditionType {
			// Only update LastTransitionTime if status changed
			if existing.Status != status {
				(*conditions)[i] = condition
			} else {
				condition.LastTransitionTime = existing.LastTransitionTime
				(*conditions)[i] = condition
			}
			return
		}
	}

	// Add new condition
	*conditions = append(*conditions, condition)
}

// CreateOrUpdateConfigMap creates or updates a ConfigMap with owner reference
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

	op, err := controllerutil.CreateOrUpdate(ctx, c, configMap, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(owner, configMap, scheme); err != nil {
			return err
		}

		// Update data
		configMap.Data = data

		return nil
	})

	if err != nil {
		return err
	}

	if op != controllerutil.OperationResultNone {
		// Log or track the operation if needed
		_ = op
	}

	return nil
}

// DeleteConfigMap deletes a ConfigMap if it exists
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

// AddFinalizer adds a finalizer to the object
func AddFinalizer(obj client.Object) bool {
	finalizers := obj.GetFinalizers()
	for _, f := range finalizers {
		if f == FinalizerName {
			return false
		}
	}
	obj.SetFinalizers(append(finalizers, FinalizerName))
	return true
}

// RemoveFinalizer removes a finalizer from the object
func RemoveFinalizer(obj client.Object) bool {
	finalizers := obj.GetFinalizers()
	for i, f := range finalizers {
		if f == FinalizerName {
			obj.SetFinalizers(append(finalizers[:i], finalizers[i+1:]...))
			return true
		}
	}
	return false
}

// HasFinalizer checks if the object has the finalizer
func HasFinalizer(obj client.Object) bool {
	finalizers := obj.GetFinalizers()
	for _, f := range finalizers {
		if f == FinalizerName {
			return true
		}
	}
	return false
}

// GenerateConfigMapName generates a ConfigMap name for a resource
func GenerateConfigMapName(resourceName, suffix string) string {
	return fmt.Sprintf("%s-%s", resourceName, suffix)
}

// GenerateServiceName generates a Service name for a resource
func GenerateServiceName(resourceName string) string {
	return resourceName
}

// GenerateDeploymentName generates a Deployment name for a resource
func GenerateDeploymentName(resourceName string) string {
	return resourceName
}

// GenerateIngressName generates an Ingress name for a resource
func GenerateIngressName(resourceName string) string {
	return resourceName
}

// GetCommonLabels returns common labels for resources
func GetCommonLabels(resourceName, resourceKind string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       resourceName,
		"app.kubernetes.io/managed-by": "language-operator",
		"app.kubernetes.io/part-of":    "langop",
		"langop.io/kind":               resourceKind,
	}
}

// MergeLabels merges two label maps, with the second map taking precedence
func MergeLabels(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// Int32Ptr returns a pointer to an int32 value
func Int32Ptr(i int32) *int32 {
	return &i
}

// StringPtr returns a pointer to a string value
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}
