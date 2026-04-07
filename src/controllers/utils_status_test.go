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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetCondition_AddNew(t *testing.T) {
	var conditions []metav1.Condition
	changed := SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	assert.True(t, changed)
	require.Len(t, conditions, 1)
	assert.Equal(t, "Ready", conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
	assert.Equal(t, "ReconcileSuccess", conditions[0].Reason)
}

func TestSetCondition_UpdateExistingStatusChange(t *testing.T) {
	var conditions []metav1.Condition
	SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	originalTime := conditions[0].LastTransitionTime

	// Status changes from True to False → LastTransitionTime should be updated
	changed := SetCondition(&conditions, "Ready", metav1.ConditionFalse, "Error", "something failed", 2)
	assert.True(t, changed)
	require.Len(t, conditions, 1, "should update in-place, not append")
	assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
	assert.Equal(t, "Error", conditions[0].Reason)
	// LastTransitionTime must differ when status changes
	assert.NotEqual(t, originalTime, conditions[0].LastTransitionTime)
}

func TestSetCondition_UpdateNoStatusChange(t *testing.T) {
	var conditions []metav1.Condition
	SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	originalTime := conditions[0].LastTransitionTime

	// Same status, different reason → updated but LastTransitionTime preserved
	changed := SetCondition(&conditions, "Ready", metav1.ConditionTrue, "NewReason", "updated message", 2)
	assert.True(t, changed)
	require.Len(t, conditions, 1)
	assert.Equal(t, "NewReason", conditions[0].Reason)
	assert.Equal(t, originalTime, conditions[0].LastTransitionTime, "LastTransitionTime must not change when status is unchanged")
}

func TestSetCondition_NoOpWhenUnchanged(t *testing.T) {
	var conditions []metav1.Condition
	SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)

	// Identical call must return false (nothing changed)
	changed := SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	assert.False(t, changed)
	require.Len(t, conditions, 1)
}
