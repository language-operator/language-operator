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

package v1alpha1

// Condition type constants shared across all Language Operator resources.
// Use these instead of raw string literals to get compile-time safety.
const (
	ConditionReady                 = "Ready"
	ConditionRegistryValidated     = "RegistryValidated"
	ConditionNetworkPolicyReady    = "NetworkPolicyReady"
	ConditionNetworkPolicyEnforced = "NetworkPolicyEnforced"
	ConditionWebhooksReady         = "WebhooksReady"
	ConditionGatewayReady          = "GatewayReady"
	ConditionDNSConfigured         = "DNSConfigured"
	ConditionCapacityReady         = "CapacityReady"
	ConditionSchemasDiscovered     = "SchemasDiscovered"
	ConditionWebhookRouteCreated   = "WebhookRouteCreated"
	ConditionWebhookRouteReady     = "WebhookRouteReady"
	ConditionRuntimeResolved       = "RuntimeResolved"
	ConditionWorkspaceSeeded       = "WorkspaceSeeded"
)

// Condition reason constants used in SetCondition calls across all controllers.
// Use these instead of raw string literals to get compile-time safety.
const (
	ReasonReconcileSuccess      = "ReconcileSuccess"
	ReasonNetworkPolicyReady    = "NetworkPolicyReady"
	ReasonNetworkPolicyError    = "NetworkPolicyError"
	ReasonNetworkPolicyDisabled = "NetworkPolicyDisabled"
	ReasonNetworkPolicyTimeout  = "NetworkPolicyTimeout"
	ReasonValidated             = "Validated"
	ReasonServiceError          = "ServiceError"
	ReasonRegistryNotAllowed    = "RegistryNotAllowed"
	ReasonPending               = "Pending"
	ReasonUpdating              = "Updating"
	ReasonPodsNotReady          = "PodsNotReady"
	ReasonDeploymentError       = "DeploymentError"
	ReasonDeploymentNotFound    = "DeploymentNotFound"
	ReasonConfigMapError        = "ConfigMapError"
	ReasonPVCError              = "PVCError"
	ReasonServiceAccountError   = "ServiceAccountError"
	ReasonSchemasDiscovered     = "SchemasDiscovered"
	ReasonSchemaDiscoveryFailed = "SchemaDiscoveryFailed"
	ReasonNoRunningAgentPod     = "NoRunningAgentPod"
	ReasonGatewayReady          = "GatewayReady"
	ReasonWebhookRouteReady     = "WebhookRouteReady"
	ReasonWebhookRouteNotReady  = "WebhookRouteNotReady"
	ReasonIngressCreated        = "IngressCreated"
	ReasonIngressCreationFailed = "IngressCreationFailed"
	ReasonCNINotSupported       = "CNINotSupported"
	ReasonEnforced              = "Enforced"
	ReasonDisabled              = "Disabled"
	ReasonConfigured            = "Configured"
	ReasonRuntimeNotFound       = "RuntimeNotFound"
	ReasonRuntimeApplied        = "RuntimeApplied"
	ReasonWorkspaceSeedReady    = "WorkspaceSeedReady"
	ReasonWorkspaceSeedError    = "WorkspaceSeedError"
	ReasonSecretNotFound        = "SecretNotFound"
)
