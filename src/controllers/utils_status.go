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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
)

// SetCondition updates or adds a condition to the conditions slice.
// Returns true if the condition was actually changed.
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string, generation int64) bool {
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
			// Check if anything actually changed
			if existing.Status == status &&
				existing.Reason == reason &&
				existing.Message == message &&
				existing.ObservedGeneration == generation {
				// Nothing changed, don't update
				return false
			}

			// Only update LastTransitionTime if status changed
			if existing.Status != status {
				(*conditions)[i] = condition
			} else {
				condition.LastTransitionTime = existing.LastTransitionTime
				(*conditions)[i] = condition
			}
			return true
		}
	}

	// Add new condition
	*conditions = append(*conditions, condition)
	return true
}

// SetPhase updates the status phase and records the observed generation.
// Use this instead of setting Phase and ObservedGeneration separately.
func SetPhase(phase *string, observedGeneration *int64, newPhase string, generation int64) {
	*phase = newPhase
	*observedGeneration = generation
}

// ValidateClusterReference validates that the LanguageCluster for this namespace exists and is ready.
// By convention, namespace name == cluster name.
func ValidateClusterReference(ctx context.Context, c client.Client, namespace string) error {
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := c.Get(ctx, client.ObjectKey{Name: namespace}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster %s: %w", namespace, err)
	}

	if cluster.Status.Phase != events.PhaseStatusReady {
		return fmt.Errorf("cluster %s is not ready yet (phase: %s)", namespace, cluster.Status.Phase)
	}

	return nil
}
