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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// Event reason constants - standardized across all controllers
const (
	// Resource creation/lifecycle events
	ReasonResourceCreated = "ResourceCreated"
	ReasonResourceReady   = "ResourceReady"

	// Configuration and validation failures
	ReasonRegistryValidated        = "RegistryValidated"
	ReasonRegistryValidationFailed = "RegistryValidationFailed"
	ReasonConfigurationFailed      = "ConfigurationFailed"
	ReasonValidationFailed         = "ValidationFailed"

	// Infrastructure component failures
	ReasonConfigMapFailed     = "ConfigMapFailed"
	ReasonDeploymentFailed    = "DeploymentFailed"
	ReasonServiceFailed       = "ServiceFailed"
	ReasonNetworkPolicyFailed = "NetworkPolicyFailed"
	ReasonRBACFailed          = "RBACFailed"

	// Network-related events
	ReasonNetworkPolicyTimeout     = "NetworkPolicyTimeout"
	ReasonNetworkPolicyUnsupported = "NetworkPolicyUnsupported"

	// Runtime and operational events
	ReasonRuntimeError            = "RuntimeError"
	ReasonGatewayDeploymentFailed = "GatewayDeploymentFailed"
)

// Resource status phase constants - standardized across all controllers
const (
	PhaseStatusPending  = "Pending"
	PhaseStatusReady    = "Ready"
	PhaseStatusFailed   = "Failed"
	PhaseStatusRunning  = "Running"
	PhaseStatusUpdating = "Updating"
	PhaseStatusDegraded = "Degraded"
	// PhaseStatusSucceeded is a completed run — a task-mode LanguageAgent whose
	// most recent Argo Workflow finished successfully.
	PhaseStatusSucceeded = "Succeeded"
	// PhaseStatusSuspended is a LanguageAgent held back by spec.execution.suspend.
	PhaseStatusSuspended = "Suspended"
)

// EventManager provides centralized, type-safe event recording for all controllers.
// All methods are nil-safe: calling any method on a nil *EventManager is a no-op.
// This lets tests omit EventManager initialization without guard boilerplate.
type EventManager struct {
	recorder record.EventRecorder
}

// NewEventManager creates a new EventManager instance
func NewEventManager(recorder record.EventRecorder) *EventManager {
	return &EventManager{
		recorder: recorder,
	}
}

// Resource lifecycle events

func (e *EventManager) RecordResourceCreated(obj runtime.Object, resourceType, details string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceCreated,
		"%s created: %s", resourceType, details)
}

func (e *EventManager) RecordResourceReady(obj runtime.Object, resourceType, details string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"%s is ready: %s", resourceType, details)
}

// Configuration and validation failures

func (e *EventManager) RecordRegistryValidationFailed(obj runtime.Object, image string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRegistryValidationFailed,
		"Image registry not in whitelist: %s", image)
}

func (e *EventManager) RecordConfigurationFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonConfigurationFailed,
		"Failed to store configuration: %v", err)
}

func (e *EventManager) RecordValidationFailed(obj runtime.Object, details string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonValidationFailed,
		"Validation failed: %s", details)
}

// Infrastructure component failures

func (e *EventManager) RecordConfigMapFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonConfigMapFailed,
		"Failed to reconcile configuration: %v", err)
}

func (e *EventManager) RecordDeploymentFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonDeploymentFailed,
		"Failed to create or update deployment: %v", err)
}

func (e *EventManager) RecordServiceFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonServiceFailed,
		"Failed to create or update service: %v", err)
}

func (e *EventManager) RecordNetworkPolicyFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonNetworkPolicyFailed,
		"Failed to configure network isolation: %v", err)
}

func (e *EventManager) RecordRBACFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRBACFailed,
		"Failed to configure permissions: %v", err)
}

func (e *EventManager) RecordGatewayDeploymentFailed(obj runtime.Object, err error) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonGatewayDeploymentFailed,
		"Failed to create or update gateway deployment: %v", err)
}

// Network-related events

func (e *EventManager) RecordNetworkPolicyTimeout(obj runtime.Object) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonNetworkPolicyTimeout,
		"NetworkPolicy creation timed out. Network isolation may be degraded. Consider increasing --network-policy-timeout flag for slow CNI.")
}

func (e *EventManager) RecordNetworkPolicyUnsupported(obj runtime.Object, cni string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonNetworkPolicyUnsupported,
		"CNI '%s' does not enforce NetworkPolicy", cni)
}

// Runtime and operational events

func (e *EventManager) RecordRuntimeError(obj runtime.Object, podName, errorMessage string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRuntimeError,
		"Pod %s failed: %s", podName, errorMessage)
}

// Specific resource type convenience methods

func (e *EventManager) RecordToolCreated(obj runtime.Object, toolType string) {
	if e == nil {
		return
	}
	e.RecordResourceCreated(obj, "LanguageTool", fmt.Sprintf("tool with type %s", toolType))
}

func (e *EventManager) RecordModelCreated(obj runtime.Object) {
	if e == nil {
		return
	}
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageModel created")
}

func (e *EventManager) RecordModelReady(obj runtime.Object, provider string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"Model gateway is ready for provider %s", provider)
}

func (e *EventManager) RecordPersonaCreated(obj runtime.Object, displayName string) {
	if e == nil {
		return
	}
	e.RecordResourceCreated(obj, "LanguagePersona", displayName)
}

func (e *EventManager) RecordPersonaReady(obj runtime.Object, displayName string) {
	if e == nil {
		return
	}
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"Persona '%s' is ready for agent assignment", displayName)
}

func (e *EventManager) RecordClusterCreated(obj runtime.Object) {
	if e == nil {
		return
	}
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageCluster created")
}

func (e *EventManager) RecordClusterReady(obj runtime.Object) {
	if e == nil {
		return
	}
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceReady, "LanguageCluster is ready for agents")
}

func (e *EventManager) RecordVersionReady(obj runtime.Object) {
	if e == nil {
		return
	}
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"LanguageAgentVersion is ready for deployment")
}

func (e *EventManager) RecordVersionError(obj runtime.Object, reason string, err error) {
	if e == nil {
		return
	}
	e.recorder.Event(obj, corev1.EventTypeWarning, reason, err.Error())
}

// Registry validation convenience method
func (e *EventManager) RecordRegistryValidated(obj runtime.Object) {
	if e == nil {
		return
	}
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonRegistryValidated,
		"Container image registry validated against whitelist")
}
