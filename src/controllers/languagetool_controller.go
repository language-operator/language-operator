package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/validation"
)

// LanguageToolReconciler reconciles a LanguageTool object
type LanguageToolReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Log               logr.Logger
	AllowedRegistries []string
}

// tracer is the OpenTelemetry tracer for the LanguageTool controller
var toolTracer = otel.Tracer("language-operator/tool-controller")

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents an MCP JSON-RPC error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPToolsListResult represents the result of tools/list MCP method
type MCPToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPTool represents an MCP tool definition
type MCPTool struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	InputSchema *MCPToolInputSchema `json:"inputSchema,omitempty"`
}

// MCPToolInputSchema represents the input schema for an MCP tool
type MCPToolInputSchema struct {
	Type       string                       `json:"type,omitempty"`
	Properties map[string]MCPSchemaProperty `json:"properties,omitempty"`
	Required   []string                     `json:"required,omitempty"`
}

// MCPSchemaProperty represents a property in an MCP schema
type MCPSchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

//+kubebuilder:rbac:groups=langop.io,resources=languagetools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagetools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagetools/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageToolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Start OpenTelemetry span for reconciliation
	ctx, span := toolTracer.Start(ctx, "tool.reconcile")
	defer span.End()

	// Add basic span attributes from request
	span.SetAttributes(
		attribute.String("tool.name", req.Name),
		attribute.String("tool.namespace", req.Namespace),
	)

	log := log.FromContext(ctx)

	// Fetch the LanguageTool instance
	tool := &langopv1alpha1.LanguageTool{}
	if err := r.Get(ctx, req.NamespacedName, tool); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			span.SetStatus(codes.Ok, "Resource not found (deleted)")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageTool")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get LanguageTool")
		return ctrl.Result{}, err
	}

	// Add tool-specific attributes to span
	span.SetAttributes(
		attribute.String("tool.type", tool.Spec.Type),
		attribute.String("tool.deployment_mode", tool.Spec.DeploymentMode),
		attribute.Int64("tool.generation", tool.Generation),
	)

	// Handle deletion
	if !tool.DeletionTimestamp.IsZero() {
		span.AddEvent("Deleting tool")
		if controllerutil.ContainsFinalizer(tool, FinalizerName) {
			// Perform cleanup
			if err := r.cleanupResources(ctx, tool); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to cleanup resources")
				return ctrl.Result{}, err
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(tool, FinalizerName)
			if err := r.Update(ctx, tool); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		span.SetStatus(codes.Ok, "Tool deleted successfully")
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(tool, FinalizerName) {
		controllerutil.AddFinalizer(tool, FinalizerName)
		if err := r.Update(ctx, tool); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Validate image registry against whitelist
	if err := r.validateImageRegistry(tool); err != nil {
		log.Error(err, "Image registry validation failed", "image", tool.Spec.Image)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Image registry validation failed")
		SetCondition(&tool.Status.Conditions, "RegistryValidated", metav1.ConditionFalse, "RegistryNotAllowed", err.Error(), tool.Generation)
		if updateErr := r.Status().Update(ctx, tool); updateErr != nil {
			log.Error(updateErr, "Failed to update status after registry validation failure")
		}
		return ctrl.Result{}, err
	}
	SetCondition(&tool.Status.Conditions, "RegistryValidated", metav1.ConditionTrue, "Validated", "Image registry is in whitelist", tool.Generation)

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, tool); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile ConfigMap")
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "ConfigMapError", err.Error(), tool.Generation)
		r.Status().Update(ctx, tool)
		return ctrl.Result{}, err
	}

	// Skip Deployment and Service for sidecar mode tools
	// Sidecar tools are injected into agent pods directly
	if tool.Spec.DeploymentMode != "sidecar" {
		// Reconcile Deployment
		if err := r.reconcileDeployment(ctx, tool); err != nil {
			log.Error(err, "Failed to reconcile Deployment")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile Deployment")
			SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), tool.Generation)
			r.Status().Update(ctx, tool)
			return ctrl.Result{}, err
		}

		// Reconcile Service
		if err := r.reconcileService(ctx, tool); err != nil {
			log.Error(err, "Failed to reconcile Service")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile Service")
			SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "ServiceError", err.Error(), tool.Generation)
			r.Status().Update(ctx, tool)
			return ctrl.Result{}, err
		}
	}

	// Reconcile NetworkPolicy for network isolation
	if err := r.reconcileNetworkPolicy(ctx, tool); err != nil {
		log.Error(err, "Failed to reconcile NetworkPolicy")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile NetworkPolicy")
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "NetworkPolicyError", err.Error(), tool.Generation)
		r.Status().Update(ctx, tool)
		return ctrl.Result{}, err
	}

	// Update status based on actual pod readiness
	if err := r.updateToolStatus(ctx, tool); err != nil {
		log.Error(err, "Failed to update LanguageTool status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update LanguageTool status")
		return ctrl.Result{}, err
	}

	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *LanguageToolReconciler) reconcileConfigMap(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	data := make(map[string]string)

	// Add tool spec as JSON
	specJSON, err := json.Marshal(tool.Spec)
	if err != nil {
		return err
	}
	data["tool.json"] = string(specJSON)

	// Add other useful data
	data["name"] = tool.Name
	data["namespace"] = tool.Namespace
	data["type"] = string(tool.Spec.Type)

	configMapName := GenerateConfigMapName(tool.Name, "tool")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, tool, configMapName, tool.Namespace, data)
}

func (r *LanguageToolReconciler) reconcileDeployment(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Determine target namespace and labels
	targetNamespace := tool.Namespace
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	// If cluster ref is set, verify cluster exists and is ready
	if err := ValidateClusterReference(ctx, r.Client, tool.Spec.ClusterRef, tool.Namespace); err != nil {
		return err
	}

	// Add cluster label if cluster ref is set
	if tool.Spec.ClusterRef != "" {
		labels["langop.io/cluster"] = tool.Spec.ClusterRef
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(tool, deployment, r.Scheme); err != nil {
			return err
		}

		// Set deployment spec
		replicas := int32(1)
		if tool.Spec.Replicas != nil {
			replicas = *tool.Spec.Replicas
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tool",
							Image: tool.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: tool.Spec.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: tool.Spec.Env,
						},
					},
				},
			},
		}

		// Add resource requirements if specified
		deployment.Spec.Template.Spec.Containers[0].Resources = tool.Spec.Resources

		// Add affinity if specified
		if tool.Spec.Affinity != nil {
			deployment.Spec.Template.Spec.Affinity = tool.Spec.Affinity
		}

		return nil
	})

	return err
}

func (r *LanguageToolReconciler) reconcileService(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Determine target namespace and labels (same logic as deployment)
	targetNamespace := tool.Namespace
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	// If cluster ref is set, verify cluster exists in same namespace
	if tool.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: tool.Spec.ClusterRef, Namespace: tool.Namespace}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", tool.Spec.ClusterRef)
		}

		// Add cluster label
		labels["langop.io/cluster"] = tool.Spec.ClusterRef
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(tool, service, r.Scheme); err != nil {
			return err
		}

		// Set service spec
		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       tool.Spec.Port,
					TargetPort: intstr.FromInt(int(tool.Spec.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		}

		return nil
	})

	return err
}

func (r *LanguageToolReconciler) reconcileNetworkPolicy(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	// Get OTEL endpoint from operator environment
	// This ensures tools can send traces to the collector
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Build NetworkPolicy using helper from utils.go
	networkPolicy := BuildEgressNetworkPolicy(
		tool.Name,
		tool.Namespace,
		labels,
		"", // provider - not applicable for tools
		"", // endpoint - not applicable for tools
		otelEndpoint,
		tool.Spec.Egress,
	)

	// Create or update the NetworkPolicy with owner reference
	return CreateOrUpdateNetworkPolicy(ctx, r.Client, r.Scheme, tool, networkPolicy)
}

// discoverMCPToolSchemas queries an MCP server to discover available tools and their schemas
func (r *LanguageToolReconciler) discoverMCPToolSchemas(ctx context.Context, endpoint string) ([]langopv1alpha1.ToolSchema, error) {
	log := log.FromContext(ctx)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create MCP tools/list request
	request := MCPRequest{
		JSONRpc: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	// Make HTTP POST request to MCP server
	resp, err := client.Post(fmt.Sprintf("http://%s/mcp", endpoint), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP server at %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d", resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP response: %w", err)
	}

	// Parse MCP response
	var mcpResp MCPResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP server error: %s (code %d)", mcpResp.Error.Message, mcpResp.Error.Code)
	}

	// Parse tools list result
	var toolsResult MCPToolsListResult
	if err := json.Unmarshal(mcpResp.Result, &toolsResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools list: %w", err)
	}

	// Convert MCP tools to LanguageOperator ToolSchema format
	var schemas []langopv1alpha1.ToolSchema
	for _, mcpTool := range toolsResult.Tools {
		schema := langopv1alpha1.ToolSchema{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
		}

		// Convert input schema if present
		if mcpTool.InputSchema != nil {
			schema.InputSchema = &langopv1alpha1.ToolSchemaDefinition{
				Type:       mcpTool.InputSchema.Type,
				Required:   mcpTool.InputSchema.Required,
				Properties: make(map[string]langopv1alpha1.ToolProperty),
			}

			// Convert properties
			for propName, mcpProp := range mcpTool.InputSchema.Properties {
				prop := langopv1alpha1.ToolProperty{
					Type:        mcpProp.Type,
					Description: mcpProp.Description,
				}

				// Convert first example if available
				if len(mcpProp.Examples) > 0 {
					exampleBytes, _ := json.Marshal(mcpProp.Examples[0])
					prop.Example = string(exampleBytes)
				}

				schema.InputSchema.Properties[propName] = prop
			}
		}

		schemas = append(schemas, schema)
	}

	log.Info("Successfully discovered MCP tool schemas", "endpoint", endpoint, "toolCount", len(schemas))
	return schemas, nil
}

func (r *LanguageToolReconciler) updateToolStatus(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// For sidecar mode tools, just set as ready (no deployment to check)
	if tool.Spec.DeploymentMode == "sidecar" {
		tool.Status.Phase = "Running"
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageTool is ready", tool.Generation)

		// Note: Sidecar tools don't have a service endpoint, so we can't discover schemas
		// Schemas will be populated from agent runtime when the sidecar is used

		return r.Status().Update(ctx, tool)
	}

	// For service mode tools, check deployment status
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			// Deployment doesn't exist yet
			tool.Status.Phase = "Pending"
			SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentNotFound", "Deployment not found", tool.Generation)
			return r.Status().Update(ctx, tool)
		}
		return err
	}

	// Update replica counts from deployment status
	tool.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	tool.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	tool.Status.UpdatedReplicas = deployment.Status.UpdatedReplicas
	tool.Status.UnavailableReplicas = deployment.Status.UnavailableReplicas

	// Determine phase based on deployment status
	desiredReplicas := int32(1)
	if tool.Spec.Replicas != nil {
		desiredReplicas = *tool.Spec.Replicas
	}

	// Check if deployment is updating
	if deployment.Status.UpdatedReplicas < desiredReplicas {
		tool.Status.Phase = "Updating"
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "Updating", "Deployment is updating", tool.Generation)
		return r.Status().Update(ctx, tool)
	}

	// Check if any pods are ready
	if deployment.Status.ReadyReplicas > 0 {
		tool.Status.Phase = "Running"
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageTool is ready", tool.Generation)

		// Discover MCP tool schemas for service mode tools
		if tool.Status.Endpoint != "" && tool.Spec.Type == "mcp" {
			schemas, err := r.discoverMCPToolSchemas(ctx, tool.Status.Endpoint)
			if err != nil {
				// Log error but don't fail - tool is still ready even if schema discovery fails
				log := log.FromContext(ctx)
				log.Error(err, "Failed to discover MCP tool schemas", "tool", tool.Name, "endpoint", tool.Status.Endpoint)
			} else {
				// Update tool schemas and available tools list
				tool.Status.ToolSchemas = schemas

				// Update the AvailableTools list for backward compatibility
				var toolNames []string
				for _, schema := range schemas {
					toolNames = append(toolNames, schema.Name)
				}
				tool.Status.AvailableTools = toolNames
			}
		}

		return r.Status().Update(ctx, tool)
	}

	// No pods ready - check if deployment has been created recently
	if deployment.Status.AvailableReplicas == 0 && deployment.Status.UnavailableReplicas > 0 {
		// Pods exist but none are ready - likely CrashLoopBackOff or similar
		tool.Status.Phase = "Failed"
		SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "PodsNotReady", "No pods are ready", tool.Generation)
		return r.Status().Update(ctx, tool)
	}

	// Deployment exists but no replicas yet
	tool.Status.Phase = "Pending"
	SetCondition(&tool.Status.Conditions, "Ready", metav1.ConditionFalse, "Pending", "Waiting for pods to be scheduled", tool.Generation)
	return r.Status().Update(ctx, tool)
}

func (r *LanguageToolReconciler) cleanupResources(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Resources will be cleaned up automatically via owner references
	return nil
}

// validateImageRegistry validates that the tool's container image registry is in the whitelist
func (r *LanguageToolReconciler) validateImageRegistry(tool *langopv1alpha1.LanguageTool) error {
	// Skip validation if no whitelist configured
	if len(r.AllowedRegistries) == 0 {
		return nil
	}

	return validation.ValidateImageRegistry(tool.Spec.Image, r.AllowedRegistries)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageToolReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageTool{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
