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
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// mockEventRecorder captures events for assertion in tests.
type mockEventRecorder struct {
	Events []string
}

func (m *mockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	m.Events = append(m.Events, eventtype+":"+reason+":"+message)
}

func (m *mockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Event(object, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (m *mockEventRecorder) AnnotatedEventf(object runtime.Object, _ map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Event(object, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func newTestSetup() (*mockEventRecorder, *EventManager, *corev1.Pod) {
	rec := &mockEventRecorder{}
	mgr := NewEventManager(rec)
	obj := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}}
	return rec, mgr, obj
}

func assertEvent(t *testing.T, rec *mockEventRecorder, idx int, parts ...string) {
	t.Helper()
	if idx >= len(rec.Events) {
		t.Fatalf("expected at least %d event(s), got %d", idx+1, len(rec.Events))
	}
	ev := rec.Events[idx]
	for _, p := range parts {
		if !strings.Contains(ev, p) {
			t.Errorf("event[%d] = %q; expected to contain %q", idx, ev, p)
		}
	}
}

// --- Lifecycle events ---

func TestEventManager_RecordResourceCreated(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordResourceCreated(obj, "LanguageAgent", "my-agent")
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageAgent", "my-agent")
}

func TestEventManager_RecordResourceReady(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordResourceReady(obj, "LanguageAgent", "running fine")
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceReady, "LanguageAgent", "running fine")
}

// --- Warning/failure events (table-driven) ---

func TestEventManager_WarningEvents(t *testing.T) {
	testErr := errors.New("something broke")

	cases := []struct {
		name   string
		call   func(*EventManager, runtime.Object)
		reason string
	}{
		{
			name:   "RecordRegistryValidationFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordRegistryValidationFailed(o, "bad.registry/image:tag") },
			reason: ReasonRegistryValidationFailed,
		},
		{
			name:   "RecordConfigurationFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordConfigurationFailed(o, testErr) },
			reason: ReasonConfigurationFailed,
		},
		{
			name:   "RecordValidationFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordValidationFailed(o, "spec is invalid") },
			reason: ReasonValidationFailed,
		},
		{
			name:   "RecordConfigMapFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordConfigMapFailed(o, testErr) },
			reason: ReasonConfigMapFailed,
		},
		{
			name:   "RecordDeploymentFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordDeploymentFailed(o, testErr) },
			reason: ReasonDeploymentFailed,
		},
		{
			name:   "RecordServiceFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordServiceFailed(o, testErr) },
			reason: ReasonServiceFailed,
		},
		{
			name:   "RecordNetworkPolicyFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordNetworkPolicyFailed(o, testErr) },
			reason: ReasonNetworkPolicyFailed,
		},
		{
			name:   "RecordRBACFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordRBACFailed(o, testErr) },
			reason: ReasonRBACFailed,
		},
		{
			name:   "RecordProxyDeploymentFailed",
			call:   func(m *EventManager, o runtime.Object) { m.RecordProxyDeploymentFailed(o, testErr) },
			reason: ReasonProxyDeploymentFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, mgr, obj := newTestSetup()
			tc.call(mgr, obj)
			if len(rec.Events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(rec.Events))
			}
			assertEvent(t, rec, 0, corev1.EventTypeWarning, tc.reason)
		})
	}
}

// --- Network events ---

func TestEventManager_RecordNetworkPolicyTimeout(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordNetworkPolicyTimeout(obj)
	assertEvent(t, rec, 0, corev1.EventTypeWarning, ReasonNetworkPolicyTimeout, "NetworkPolicy")
}

func TestEventManager_RecordNetworkPolicyUnsupported(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordNetworkPolicyUnsupported(obj, "flannel")
	assertEvent(t, rec, 0, corev1.EventTypeWarning, ReasonNetworkPolicyUnsupported, "flannel")
}

// --- Runtime events ---

func TestEventManager_RecordRuntimeError(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordRuntimeError(obj, "my-pod-abc", "OOMKilled")
	assertEvent(t, rec, 0, corev1.EventTypeWarning, ReasonRuntimeError, "my-pod-abc", "OOMKilled")
}

// --- Resource-type convenience methods ---

func TestEventManager_RecordToolCreated(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordToolCreated(obj, "web-search")
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageTool", "web-search")
}

func TestEventManager_RecordModelCreated(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordModelCreated(obj)
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageModel created")
}

func TestEventManager_RecordModelReady(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordModelReady(obj, "openai")
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceReady, "openai")
}

func TestEventManager_RecordPersonaCreated(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordPersonaCreated(obj, "helpful-assistant")
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceCreated, "LanguagePersona", "helpful-assistant")
}

func TestEventManager_RecordPersonaReady(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordPersonaReady(obj, "helpful-assistant")
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceReady, "helpful-assistant")
}

func TestEventManager_RecordClusterCreated(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordClusterCreated(obj)
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageCluster created")
}

func TestEventManager_RecordClusterReady(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordClusterReady(obj)
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceReady, "LanguageCluster is ready")
}

func TestEventManager_RecordVersionReady(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordVersionReady(obj)
	assertEvent(t, rec, 0, corev1.EventTypeNormal, ReasonResourceReady)
}

func TestEventManager_RecordVersionError(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	testErr := errors.New("image pull failed")
	mgr.RecordVersionError(obj, "ImagePullFailed", testErr)
	assertEvent(t, rec, 0, corev1.EventTypeWarning, "ImagePullFailed", "image pull failed")
}

func TestEventManager_RecordRegistryValidated(t *testing.T) {
	rec, mgr, obj := newTestSetup()
	mgr.RecordRegistryValidated(obj)
	assertEvent(t, rec, 0, corev1.EventTypeNormal, "RegistryValidated")
}
