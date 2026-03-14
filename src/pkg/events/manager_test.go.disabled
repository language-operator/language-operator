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

package events

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MockEventRecorder implements record.EventRecorder for testing
type MockEventRecorder struct {
	Events []string
}

func (m *MockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	m.Events = append(m.Events, eventtype+":"+reason+":"+message)
}

func (m *MockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Event(object, eventtype, reason, messageFmt)
}

func (m *MockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Event(object, eventtype, reason, messageFmt)
}

func TestEventManager_RecordSynthesisEvents(t *testing.T) {
	mockRecorder := &MockEventRecorder{}
	eventManager := NewEventManager(mockRecorder)

	// Create a test object
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test synthesis started event
	eventManager.RecordSynthesisStarted(testPod)
	if len(mockRecorder.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockRecorder.Events))
	}

	expectedEvent := "Normal:" + ReasonSynthesisStarted + ":Starting code synthesis from natural language instructions"
	if mockRecorder.Events[0] != expectedEvent {
		t.Errorf("Expected event %q, got %q", expectedEvent, mockRecorder.Events[0])
	}

	// Test synthesis succeeded event
	eventManager.RecordSynthesisSucceeded(testPod, 2.5)
	if len(mockRecorder.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(mockRecorder.Events))
	}

	// Test synthesis failed event
	testErr := errors.New("synthesis timeout")
	eventManager.RecordSynthesisFailed(testPod, testErr)
	if len(mockRecorder.Events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(mockRecorder.Events))
	}

	failedEvent := mockRecorder.Events[2]
	if !contains(failedEvent, "Warning") || !contains(failedEvent, ReasonSynthesisFailed) {
		t.Errorf("Expected failed event to contain Warning and %s, got %q", ReasonSynthesisFailed, failedEvent)
	}
}

func TestEventManager_RecordRegistryEvents(t *testing.T) {
	mockRecorder := &MockEventRecorder{}
	eventManager := NewEventManager(mockRecorder)

	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test registry validation failed
	eventManager.RecordRegistryValidationFailed(testPod, "unauthorized.registry.com/image:tag")
	if len(mockRecorder.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockRecorder.Events))
	}

	event := mockRecorder.Events[0]
	if !contains(event, "Warning") || !contains(event, ReasonRegistryValidationFailed) {
		t.Errorf("Expected registry validation failed event, got %q", event)
	}

	// Test registry validated
	eventManager.RecordRegistryValidated(testPod)
	if len(mockRecorder.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(mockRecorder.Events))
	}
}

func TestEventManager_RecordSelfHealingEvents(t *testing.T) {
	mockRecorder := &MockEventRecorder{}
	eventManager := NewEventManager(mockRecorder)

	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test self-healing triggered
	eventManager.RecordSelfHealingTriggered(testPod, 3, 2, 5)
	if len(mockRecorder.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockRecorder.Events))
	}

	event := mockRecorder.Events[0]
	if !contains(event, "Normal") || !contains(event, ReasonSelfHealingTriggered) {
		t.Errorf("Expected self-healing triggered event, got %q", event)
	}

	// Test self-healing max attempts
	eventManager.RecordSelfHealingMaxAttempts(testPod, 5)
	if len(mockRecorder.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(mockRecorder.Events))
	}

	maxEvent := mockRecorder.Events[1]
	if !contains(maxEvent, "Warning") || !contains(maxEvent, ReasonSelfHealingMaxAttempts) {
		t.Errorf("Expected self-healing max attempts event, got %q", maxEvent)
	}
}

func TestEventManager_RecordResourceEvents(t *testing.T) {
	mockRecorder := &MockEventRecorder{}
	eventManager := NewEventManager(mockRecorder)

	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test tool created
	eventManager.RecordToolCreated(testPod, "web-search")
	if len(mockRecorder.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockRecorder.Events))
	}

	event := mockRecorder.Events[0]
	if !contains(event, "Normal") || !contains(event, ReasonResourceCreated) {
		t.Errorf("Expected tool created event, got %q", event)
	}

	// Test ConfigMap failed
	testErr := errors.New("failed to create configmap")
	eventManager.RecordConfigMapFailed(testPod, testErr)
	if len(mockRecorder.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(mockRecorder.Events))
	}

	failedEvent := mockRecorder.Events[1]
	if !contains(failedEvent, "Warning") || !contains(failedEvent, ReasonConfigMapFailed) {
		t.Errorf("Expected ConfigMap failed event, got %q", failedEvent)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
