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
	ReasonResourceUpdated = "ResourceUpdated"
	ReasonResourceDeleted = "ResourceDeleted"

	// Configuration and validation failures
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
	ReasonNetworkPolicyTimeout       = "NetworkPolicyTimeout"
	ReasonNetworkPolicyUnsupported   = "NetworkPolicyUnsupported"
	ReasonNetworkIsolationConfigured = "NetworkIsolationConfigured"

	// Synthesis and AI-related events
	ReasonSynthesisStarted      = "SynthesisStarted"
	ReasonSynthesisSucceeded    = "SynthesisSucceeded"
	ReasonSynthesisFailed       = "SynthesisFailed"
	ReasonPersonaUpdated        = "PersonaUpdated"
	ReasonExecutionModeDetected = "ExecutionModeDetected"

	// Self-healing and optimization
	ReasonSelfHealingTriggered          = "SelfHealingTriggered"
	ReasonSelfHealingMaxAttempts        = "SelfHealingMaxAttempts"
	ReasonSelfHealingSynthesisStarted   = "SelfHealingSynthesisStarted"
	ReasonSelfHealingSynthesisSucceeded = "SelfHealingSynthesisSucceeded"
	ReasonSelfHealingSynthesisFailed    = "SelfHealingSynthesisFailed"
	ReasonSelfHealingValidationFailed   = "SelfHealingValidationFailed"
	ReasonOptimizationTriggered         = "OptimizationTriggered"

	// Rate limiting and quota
	ReasonRateLimitExceeded = "RateLimitExceeded"
	ReasonQuotaExceeded     = "QuotaExceeded"

	// Runtime and operational events
	ReasonRuntimeError          = "RuntimeError"
	ReasonProxyDeploymentFailed = "ProxyDeploymentFailed"
)

// EventManager provides centralized, type-safe event recording for all controllers
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
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceCreated,
		"%s created: %s", resourceType, details)
}

func (e *EventManager) RecordResourceReady(obj runtime.Object, resourceType, details string) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"%s is ready: %s", resourceType, details)
}

// Configuration and validation failures

func (e *EventManager) RecordRegistryValidationFailed(obj runtime.Object, image string) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRegistryValidationFailed,
		"Image registry not in whitelist: %s", image)
}

func (e *EventManager) RecordConfigurationFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonConfigurationFailed,
		"Failed to store configuration: %v", err)
}

func (e *EventManager) RecordValidationFailed(obj runtime.Object, details string) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonValidationFailed,
		"Validation failed: %s", details)
}

// Infrastructure component failures

func (e *EventManager) RecordConfigMapFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonConfigMapFailed,
		"Failed to reconcile configuration: %v", err)
}

func (e *EventManager) RecordDeploymentFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonDeploymentFailed,
		"Failed to create or update deployment: %v", err)
}

func (e *EventManager) RecordServiceFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonServiceFailed,
		"Failed to create or update service: %v", err)
}

func (e *EventManager) RecordNetworkPolicyFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonNetworkPolicyFailed,
		"Failed to configure network isolation: %v", err)
}

func (e *EventManager) RecordRBACFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRBACFailed,
		"Failed to configure permissions: %v", err)
}

func (e *EventManager) RecordProxyDeploymentFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonProxyDeploymentFailed,
		"Failed to create or update proxy deployment: %v", err)
}

// Network-related events

func (e *EventManager) RecordNetworkPolicyTimeout(obj runtime.Object) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonNetworkPolicyTimeout,
		"NetworkPolicy creation timed out. Network isolation may be degraded. Consider increasing --network-policy-timeout flag for slow CNI.")
}

func (e *EventManager) RecordNetworkPolicyUnsupported(obj runtime.Object, cni string) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonNetworkPolicyUnsupported,
		"CNI '%s' does not enforce NetworkPolicy", cni)
}

// Synthesis and AI-related events

func (e *EventManager) RecordSynthesisStarted(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonSynthesisStarted,
		"Starting code synthesis from natural language instructions")
}

func (e *EventManager) RecordSynthesisSucceeded(obj runtime.Object, duration float64) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonSynthesisSucceeded,
		"Code synthesized successfully in %.2fs", duration)
}

func (e *EventManager) RecordSynthesisFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonSynthesisFailed,
		"Code synthesis failed: %v", err)
}

func (e *EventManager) RecordPersonaUpdated(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonPersonaUpdated,
		"Persona re-distilled without code re-synthesis")
}

func (e *EventManager) RecordExecutionModeDetected(obj runtime.Object, mode string) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonExecutionModeDetected,
		"Auto-detected executionMode: %s", mode)
}

// Self-healing and optimization events

func (e *EventManager) RecordSelfHealingTriggered(obj runtime.Object, failures, attempt, maxAttempts int) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonSelfHealingTriggered,
		"Self-healing synthesis triggered after %d consecutive failures (attempt %d/%d)",
		failures, attempt, maxAttempts)
}

func (e *EventManager) RecordSelfHealingMaxAttempts(obj runtime.Object, maxAttempts int) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonSelfHealingMaxAttempts,
		"Self-healing max attempts (%d) reached, agent marked as failed", maxAttempts)
}

func (e *EventManager) RecordSelfHealingSynthesisStarted(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonSelfHealingSynthesisStarted,
		"Self-healing synthesis started to recover from runtime errors")
}

func (e *EventManager) RecordSelfHealingSynthesisSucceeded(obj runtime.Object, duration float64, attempts int) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonSelfHealingSynthesisSucceeded,
		"Self-healing synthesis succeeded in %.2fs (attempt %d)", duration, attempts)
}

func (e *EventManager) RecordSelfHealingSynthesisFailed(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonSelfHealingSynthesisFailed,
		"Self-healing synthesis failed: %v", err)
}

func (e *EventManager) RecordSelfHealingValidationFailed(obj runtime.Object, details string) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonSelfHealingValidationFailed,
		"Self-healing validation failed: %s", details)
}

func (e *EventManager) RecordOptimizationTriggered(obj runtime.Object, message string) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonOptimizationTriggered, message)
}

// Rate limiting and quota events

func (e *EventManager) RecordRateLimitExceeded(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRateLimitExceeded,
		"Synthesis rate limit exceeded: %v", err)
}

func (e *EventManager) RecordQuotaExceeded(obj runtime.Object, err error) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonQuotaExceeded,
		"Synthesis attempt quota exceeded: %v", err)
}

// Runtime and operational events

func (e *EventManager) RecordRuntimeError(obj runtime.Object, podName, errorMessage string) {
	e.recorder.Eventf(obj, corev1.EventTypeWarning, ReasonRuntimeError,
		"Pod %s failed: %s", podName, errorMessage)
}

// Specific resource type convenience methods

func (e *EventManager) RecordToolCreated(obj runtime.Object, toolType string) {
	e.RecordResourceCreated(obj, "LanguageTool", fmt.Sprintf("tool with type %s", toolType))
}

func (e *EventManager) RecordModelCreated(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageModel created")
}

func (e *EventManager) RecordModelReady(obj runtime.Object, provider string) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"Model proxy is ready for provider %s", provider)
}

func (e *EventManager) RecordPersonaCreated(obj runtime.Object, displayName string) {
	e.RecordResourceCreated(obj, "LanguagePersona", displayName)
}

func (e *EventManager) RecordPersonaReady(obj runtime.Object, displayName string) {
	e.recorder.Eventf(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"Persona '%s' is ready for agent assignment", displayName)
}

func (e *EventManager) RecordClusterCreated(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceCreated, "LanguageCluster created")
}

func (e *EventManager) RecordClusterReady(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceReady, "LanguageCluster is ready for agents")
}

func (e *EventManager) RecordVersionReady(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, ReasonResourceReady,
		"LanguageAgentVersion is ready for deployment")
}

func (e *EventManager) RecordVersionError(obj runtime.Object, reason string, err error) {
	e.recorder.Event(obj, corev1.EventTypeWarning, reason, err.Error())
}

// Registry validation convenience method
func (e *EventManager) RecordRegistryValidated(obj runtime.Object) {
	e.recorder.Event(obj, corev1.EventTypeNormal, "RegistryValidated",
		"Container image registry validated against whitelist")
}
