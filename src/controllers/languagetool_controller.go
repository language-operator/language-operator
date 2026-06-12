package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/language-operator/language-operator/pkg/mcp"
	"github.com/language-operator/language-operator/pkg/reconciler"
)

// LanguageToolReconciler reconciles a LanguageTool object
type LanguageToolReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	Log                     logr.Logger
	Recorder                record.EventRecorder
	EventManager            *events.EventManager
	NetworkIsolationEnabled bool
	// HTTPClient is used for MCP schema discovery requests. When nil, http.DefaultClient is used.
	HTTPClient *http.Client
	// MCPBridgeImage is the stdio→Streamable-HTTP bridge image injected for transport=stdio
	// tools. When empty, langopv1alpha1.DefaultMCPBridgeImage is used.
	MCPBridgeImage string
	// MCPBridgeImagePullPolicy is the pull policy for the injected bridge image.
	MCPBridgeImagePullPolicy corev1.PullPolicy
	// MCPDiscoveryTimeout bounds an MCP schema-discovery handshake. When zero, 30s is used.
	MCPDiscoveryTimeout time.Duration
}

//+kubebuilder:rbac:groups=langop.io,resources=languagetools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagetools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagetools/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageToolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use the reconciler helper for common setup
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageTool]{
		Client:       r.Client,
		TracerName:   "language-operator/tool-controller",
		ResourceType: "tool",
	}

	tool := &langopv1alpha1.LanguageTool{}
	result, err := helper.StartReconcile(ctx, req, tool)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result == nil {
		// Resource was deleted
		return ctrl.Result{}, nil
	}

	// Capture the error for proper span completion
	var reconcileErr error
	defer func() {
		result.CompleteReconcile(reconcileErr)
	}()

	ctx = result.Ctx
	span := result.Span
	log := log.FromContext(ctx)

	// Deferred status write — flushes whatever phase/conditions have been set on any return path.
	defer func() {
		if !tool.DeletionTimestamp.IsZero() {
			return
		}
		tool.Status.ObservedGeneration = tool.Generation
		if updateErr := r.Status().Update(ctx, tool); updateErr != nil && !apierrors.IsNotFound(updateErr) {
			log.Error(updateErr, "Failed to update LanguageTool status")
			if reconcileErr == nil {
				reconcileErr = updateErr
			}
		}
	}()

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
				reconcileErr = err
				return ctrl.Result{}, err
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(tool, FinalizerName)
			if err := r.Update(ctx, tool); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to remove finalizer")
				reconcileErr = err
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
			reconcileErr = err
			return ctrl.Result{}, err
		}
		r.EventManager.RecordToolCreated(tool, tool.Spec.Type)
	}

	// Skip Deployment and Service for sidecar mode tools
	// Sidecar tools are injected into agent pods directly
	if tool.Spec.DeploymentMode != "sidecar" {
		// Reconcile Deployment
		if err := r.reconcileDeployment(ctx, tool); err != nil {
			log.Error(err, "Failed to reconcile Deployment")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile Deployment")
			r.EventManager.RecordDeploymentFailed(tool, err)
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), tool.Generation)
			SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusFailed, tool.Generation)
			reconcileErr = err
			return ctrl.Result{}, err
		}

		// Reconcile Service
		if err := r.reconcileService(ctx, tool); err != nil {
			log.Error(err, "Failed to reconcile Service")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile Service")
			r.EventManager.RecordServiceFailed(tool, err)
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonServiceError, err.Error(), tool.Generation)
			SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusFailed, tool.Generation)
			reconcileErr = err
			return ctrl.Result{}, err
		}

		// Reconcile HPA (create when autoscaling configured, delete when removed)
		if err := r.reconcileHPA(ctx, tool); err != nil {
			log.Error(err, "Failed to reconcile HorizontalPodAutoscaler")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile HorizontalPodAutoscaler")
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), tool.Generation)
			SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusFailed, tool.Generation)
			reconcileErr = err
			return ctrl.Result{}, err
		}
	}

	// Reconcile NetworkPolicy for network isolation (if enabled)
	if r.NetworkIsolationEnabled {
		if err := r.reconcileNetworkPolicy(ctx, tool); err != nil {
			log.Error(err, "Failed to reconcile NetworkPolicy")
			span.RecordError(err)

			// Determine if this is a timeout error vs other error
			isTimeout := strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "timeout")

			if isTimeout {
				// For timeout errors, set a specific condition but continue reconciliation
				SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionFalse, langopv1alpha1.ReasonNetworkPolicyTimeout,
					fmt.Sprintf("NetworkPolicy creation timed out. This may indicate slow CNI response. The operator will continue to retry. Error: %v", err), tool.Generation)

				log.Info("NetworkPolicy timeout detected - continuing reconciliation with degraded network isolation",
					"error", err.Error())

				// Record warning event
				r.EventManager.RecordNetworkPolicyTimeout(tool)

				// Don't fail the entire reconciliation for timeout - continue with degraded state
			} else {
				// For non-timeout errors, fail the reconciliation
				span.SetStatus(codes.Error, "Failed to reconcile NetworkPolicy")
				r.EventManager.RecordNetworkPolicyFailed(tool, err)
				SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonNetworkPolicyError, err.Error(), tool.Generation)
				SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionFalse, langopv1alpha1.ReasonNetworkPolicyError, err.Error(), tool.Generation)
				SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusFailed, tool.Generation)
				reconcileErr = err
				return ctrl.Result{}, err
			}
		} else {
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionTrue, langopv1alpha1.ReasonNetworkPolicyReady,
				"NetworkPolicy created successfully", tool.Generation)
		}
	} else {
		// Network isolation disabled - skip NetworkPolicy creation
		SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionTrue, langopv1alpha1.ReasonNetworkPolicyDisabled,
			"NetworkPolicy creation disabled via networkIsolation.enabled=false", tool.Generation)
		log.V(1).Info("Network isolation disabled - skipping NetworkPolicy creation")
	}

	// Update status based on actual pod readiness
	if err := r.updateToolStatus(ctx, tool); err != nil {
		log.Error(err, "Failed to update LanguageTool status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update LanguageTool status")
		reconcileErr = err
		return ctrl.Result{}, err
	}

	span.SetStatus(codes.Ok, "Reconciliation successful")

	// Requeue sidecar tools that haven't discovered schemas yet — retry once an agent pod is ready
	if tool.Spec.DeploymentMode == "sidecar" && len(tool.Status.ToolSchemas) == 0 {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *LanguageToolReconciler) reconcileDeployment(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Determine target namespace and labels
	targetNamespace := tool.Namespace
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	if err := ValidateClusterReference(ctx, r.Client, tool.Namespace); err != nil {
		return err
	}

	labels[langoplabels.LabelKeyLangopCluster] = tool.Namespace

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, tool, deployment, func() error {
		replicas := int32(1)
		if tool.Spec.Deployment.Replicas != nil {
			replicas = *tool.Spec.Deployment.Replicas
		}
		// When HPA is active, preserve the current replica count rather than overwriting it.
		// The HPA controller owns spec.replicas; if we reset it every reconcile the
		// Deployment will never scale.
		deploymentReplicas := &replicas
		if tool.Spec.Deployment.Autoscaling != nil && deployment.Spec.Replicas != nil {
			deploymentReplicas = deployment.Spec.Replicas
		}

		// Merge user pod labels; operator labels take precedence to protect selector stability.
		podLabels := make(map[string]string, len(labels)+len(tool.Spec.Deployment.PodLabels))
		for k, v := range tool.Spec.Deployment.PodLabels {
			podLabels[k] = v
		}
		for k, v := range labels {
			podLabels[k] = v
		}

		// Use user-supplied pod security context if set, otherwise apply operator defaults.
		podSecCtx := buildPodSecurityContext()
		if tool.Spec.Deployment.SecurityContext != nil {
			podSecCtx = tool.Spec.Deployment.SecurityContext
		}

		shareProc := len(tool.Spec.Deployment.InitContainers) > 0

		// The container (and any volumes it needs) varies by transport — see buildToolContainer.
		toolContainer, extraVolumes := r.buildToolContainer(tool)

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: deploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: tool.Spec.Deployment.PodAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:        tool.Spec.Deployment.ServiceAccountName,
					ShareProcessNamespace:     &shareProc,
					InitContainers:            tool.Spec.Deployment.InitContainers,
					Containers:                []corev1.Container{toolContainer},
					SecurityContext:           podSecCtx,
					Volumes:                   append(append([]corev1.Volume{}, tool.Spec.Deployment.Volumes...), extraVolumes...),
					ImagePullSecrets:          tool.Spec.Deployment.ImagePullSecrets,
					NodeSelector:              tool.Spec.Deployment.NodeSelector,
					Tolerations:               tool.Spec.Deployment.Tolerations,
					TopologySpreadConstraints: tool.Spec.Deployment.TopologySpreadConstraints,
					Affinity:                  tool.Spec.Deployment.Affinity,
				},
			},
		}

		return nil
	})

	return err
}

// buildToolContainer constructs the tool's main container and any extra pod volumes it needs.
// For transport=stdio the operator injects a persistent stdio→Streamable-HTTP bridge (image,
// command, args) plus writable cache + /tmp volumes so npx/uvx can run under the hardened,
// read-only-root securityContext. For streamable-http/sse the tool image is run directly.
func (r *LanguageToolReconciler) buildToolContainer(tool *langopv1alpha1.LanguageTool) (corev1.Container, []corev1.Volume) {
	container := corev1.Container{
		Name: "tool",
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: tool.Spec.Port, Protocol: corev1.ProtocolTCP},
		},
		Env:             r.buildToolEnv(tool),
		EnvFrom:         tool.Spec.Deployment.EnvFrom,
		Resources:       tool.Spec.Deployment.Resources,
		VolumeMounts:    tool.Spec.Deployment.VolumeMounts,
		LivenessProbe:   tool.Spec.Deployment.LivenessProbe,
		ReadinessProbe:  tool.Spec.Deployment.ReadinessProbe,
		StartupProbe:    tool.Spec.Deployment.StartupProbe,
		SecurityContext: buildContainerSecurityContext(),
	}

	if !isStdioTool(tool) {
		container.Image = tool.Spec.Image
		container.ImagePullPolicy = tool.Spec.Deployment.ImagePullPolicy
		container.Command = tool.Spec.Deployment.Command
		container.Args = tool.Spec.Deployment.Args
		return container, nil
	}

	// transport=stdio: wrap the stdio command in the operator-controlled persistent bridge.
	container.Image = resolveMCPBridgeImage(r.MCPBridgeImage)
	container.ImagePullPolicy = r.MCPBridgeImagePullPolicy
	container.Command = bridgeContainerCommand()
	container.Args = buildBridgeArgs(stdioCommandOf(tool), tool.Spec.Port)
	container.Env = append(container.Env, bridgeCacheEnv()...)
	container.VolumeMounts = append(container.VolumeMounts, bridgeVolumeMounts(mcpBridgeVolumePrefix)...)
	if container.ReadinessProbe == nil {
		container.ReadinessProbe = defaultBridgeReadinessProbe(tool.Spec.Port)
	}
	return container, bridgeVolumes(mcpBridgeVolumePrefix)
}

func (r *LanguageToolReconciler) buildToolEnv(tool *langopv1alpha1.LanguageTool) []corev1.EnvVar {
	// Start with user-specified environment variables
	env := append([]corev1.EnvVar{}, tool.Spec.Deployment.Env...)

	// Inject OpenTelemetry configuration from operator environment
	// Tools use the collector endpoint for sending telemetry data
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		// Tools may use various languages/runtimes, so we provide gRPC endpoint (default)
		// Most OTLP implementations support gRPC
		toolEndpoint := endpoint

		// Ensure http:// protocol is present for HTTP-based collectors
		if !strings.HasPrefix(toolEndpoint, "http://") && !strings.HasPrefix(toolEndpoint, "https://") {
			toolEndpoint = "http://" + toolEndpoint
		}

		// Configure OpenTelemetry auto-instrumentation via standard env vars
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: toolEndpoint,
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_TRACES_EXPORTER",
			Value: "otlp",
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
			Value: "grpc",
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_LOGS_EXPORTER",
			Value: "otlp",
		})

		// Inject additional OTEL variables from operator environment if present
		if resourceAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); resourceAttrs != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: resourceAttrs,
			})
		}

		if sampler := os.Getenv("OTEL_TRACES_SAMPLER"); sampler != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_TRACES_SAMPLER",
				Value: sampler,
			})
		}

		if samplerArg := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); samplerArg != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_TRACES_SAMPLER_ARG",
				Value: samplerArg,
			})
		}
	}

	// Set unique service name for tool
	env = append(env, corev1.EnvVar{
		Name:  "OTEL_SERVICE_NAME",
		Value: fmt.Sprintf("language-operator-tool-%s", tool.Name),
	})

	return env
}

// reconcileHPA creates a HorizontalPodAutoscaler for tools with autoscaling configured,
// and deletes any existing HPA when autoscaling is removed from the spec.
func (r *LanguageToolReconciler) reconcileHPA(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: tool.Namespace,
		},
	}

	if tool.Spec.Deployment.Autoscaling == nil {
		if err := r.Delete(ctx, hpa); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete HorizontalPodAutoscaler: %w", err)
		}
		return nil
	}

	as := tool.Spec.Deployment.Autoscaling
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	metrics := as.Metrics
	if len(metrics) == 0 {
		// Default: 80% average CPU utilization
		avgUtil := int32(80)
		metrics = []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &avgUtil,
					},
				},
			},
		}
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, tool, hpa, func() error {
		hpa.Labels = labels
		hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       tool.Name,
			},
			MinReplicas: as.MinReplicas,
			MaxReplicas: as.MaxReplicas,
			Metrics:     metrics,
		}
		return nil
	})
	return err
}

func (r *LanguageToolReconciler) reconcileService(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Determine target namespace and labels (same logic as deployment)
	targetNamespace := tool.Namespace
	labels := GetCommonLabels(tool.Name, "LanguageTool")

	if err := ValidateClusterReference(ctx, r.Client, tool.Namespace); err != nil {
		return err
	}

	labels[langoplabels.LabelKeyLangopCluster] = tool.Namespace

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tool.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, tool, service, func() error {
		// Set service spec
		svcType := tool.Spec.Deployment.ServiceType
		if svcType == "" {
			svcType = corev1.ServiceTypeClusterIP
		}
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
			Type: svcType,
		}
		if len(tool.Spec.Deployment.ServiceAnnotations) > 0 {
			if service.Annotations == nil {
				service.Annotations = make(map[string]string)
			}
			for k, v := range tool.Spec.Deployment.ServiceAnnotations {
				service.Annotations[k] = v
			}
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
		ctx,
		r.Client,
		tool.Name,
		tool.Namespace,
		labels,
		otelEndpoint,
		tool.Spec.NetworkPolicies,
	)

	// Wire user-defined ingress rules from spec.networkPolicies.ingress
	if tool.Spec.NetworkPolicies != nil {
		ingressRules := buildNetworkPolicyIngressRules(tool.Spec.NetworkPolicies.Ingress, tool.Namespace)
		if len(ingressRules) > 0 {
			networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
			networkPolicy.Spec.Ingress = ingressRules
		}
	}

	// Create or update the NetworkPolicy with owner reference
	return CreateOrUpdateNetworkPolicy(ctx, r.Client, r.Scheme, tool, networkPolicy)
}

// discoveryTimeout returns the bound for an MCP schema-discovery handshake.
func (r *LanguageToolReconciler) discoveryTimeout() time.Duration {
	if r.MCPDiscoveryTimeout > 0 {
		return r.MCPDiscoveryTimeout
	}
	return 30 * time.Second
}

// discoverMCPToolSchemas queries an MCP server (over Streamable HTTP, with a proper
// initialize → tools/list handshake and a legacy fallback) to discover its tools and schemas.
// endpoint is "host:port" (no scheme or path).
func (r *LanguageToolReconciler) discoverMCPToolSchemas(ctx context.Context, endpoint string) ([]langopv1alpha1.ToolSchema, error) {
	log := log.FromContext(ctx)

	// Bound the handshake; the context deadline is the primary bound (interruptible on
	// operator shutdown or reconcile deadline).
	ctx, cancel := context.WithTimeout(ctx, r.discoveryTimeout())
	defer cancel()

	client := &mcp.Client{HTTP: r.HTTPClient}
	tools, err := client.ListTools(ctx, "http://"+endpoint)
	if err != nil {
		return nil, err
	}

	schemas := toolSchemasFromMCP(tools)
	log.Info("Successfully discovered MCP tool schemas", "endpoint", endpoint, "toolCount", len(schemas))
	return schemas, nil
}

// toolSchemasFromMCP converts discovered MCP tools into the operator's ToolSchema status shape.
func toolSchemasFromMCP(tools []mcp.Tool) []langopv1alpha1.ToolSchema {
	schemas := make([]langopv1alpha1.ToolSchema, 0, len(tools))
	for _, t := range tools {
		schema := langopv1alpha1.ToolSchema{Name: t.Name, Description: t.Description}
		if t.InputSchema != nil {
			schema.InputSchema = &langopv1alpha1.ToolSchemaDefinition{
				Type:       t.InputSchema.Type,
				Required:   t.InputSchema.Required,
				Properties: make(map[string]langopv1alpha1.ToolProperty),
			}
			for propName, p := range t.InputSchema.Properties {
				prop := langopv1alpha1.ToolProperty{Type: p.Type, Description: p.Description}
				if len(p.Examples) > 0 {
					exampleBytes, _ := json.Marshal(p.Examples[0])
					prop.Example = string(exampleBytes)
				}
				schema.InputSchema.Properties[propName] = prop
			}
		}
		schemas = append(schemas, schema)
	}
	return schemas
}

// discoverSidecarSchemas finds a running agent pod that carries this sidecar tool container
// and queries it for MCP tool schemas. Returns nil schemas (not an error) when no suitable pod
// is ready yet — the caller should requeue and retry.
func (r *LanguageToolReconciler) discoverSidecarSchemas(ctx context.Context, tool *langopv1alpha1.LanguageTool) ([]langopv1alpha1.ToolSchema, error) {
	log := log.FromContext(ctx)

	// Sidecar containers are named "tool-<toolname>" inside agent pods
	containerName := fmt.Sprintf("tool-%s", tool.Name)
	port := tool.Spec.Port
	if port == 0 {
		port = 8080
	}

	// List all agent pods in this namespace
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(tool.Namespace),
		client.MatchingLabels{"langop.io/kind": "LanguageAgent"},
	); err != nil {
		return nil, fmt.Errorf("failed to list agent pods: %w", err)
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase != corev1.PodRunning || pod.Status.PodIP == "" {
			continue
		}

		// Check init containers first (native sidecars run as init containers with restartPolicy=Always)
		if isSidecarContainerReady(pod.Status.InitContainerStatuses, containerName) {
			endpoint := fmt.Sprintf("%s:%d", pod.Status.PodIP, port)
			schemas, err := r.discoverMCPToolSchemas(ctx, endpoint)
			if err != nil {
				log.Error(err, "Failed to discover sidecar schemas via agent pod", "pod", pod.Name, "endpoint", endpoint)
				continue
			}
			log.Info("Discovered sidecar schemas via agent pod", "pod", pod.Name, "toolCount", len(schemas))
			return schemas, nil
		}

		// Also check regular containers for compatibility
		if isSidecarContainerReady(pod.Status.ContainerStatuses, containerName) {
			endpoint := fmt.Sprintf("%s:%d", pod.Status.PodIP, port)
			schemas, err := r.discoverMCPToolSchemas(ctx, endpoint)
			if err != nil {
				log.Error(err, "Failed to discover sidecar schemas via agent pod", "pod", pod.Name, "endpoint", endpoint)
				continue
			}
			log.Info("Discovered sidecar schemas via agent pod", "pod", pod.Name, "toolCount", len(schemas))
			return schemas, nil
		}
	}

	return nil, nil
}

// isSidecarContainerReady returns true if a container with the given name is ready in the status slice.
func isSidecarContainerReady(statuses []corev1.ContainerStatus, name string) bool {
	for _, cs := range statuses {
		if cs.Name == name {
			return cs.Ready
		}
	}
	return false
}

func (r *LanguageToolReconciler) updateToolStatus(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// For sidecar mode tools, discover schemas from a running agent pod
	if tool.Spec.DeploymentMode == "sidecar" {
		SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusRunning, tool.Generation)
		SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue, langopv1alpha1.ReasonReconcileSuccess, "LanguageTool is ready", tool.Generation)

		schemas, err := r.discoverSidecarSchemas(ctx, tool)
		if err != nil {
			// Log but don't fail — tool is still injected even without cached schemas
			log := log.FromContext(ctx)
			log.Error(err, "Failed to discover sidecar schemas", "tool", tool.Name)
		}

		if len(schemas) > 0 {
			tool.Status.ToolSchemas = schemas
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionSchemasDiscovered, metav1.ConditionTrue, langopv1alpha1.ReasonSchemasDiscovered, "Tool schemas discovered from running agent pod", tool.Generation)
		} else {
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionSchemasDiscovered, metav1.ConditionFalse, langopv1alpha1.ReasonNoRunningAgentPod, "No running agent pod with this sidecar found; schemas will populate once an agent pod is ready", tool.Generation)
		}

		return nil
	}

	// For service mode tools, check deployment status
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusPending, tool.Generation)
			SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentNotFound, "Deployment not found", tool.Generation)
			return nil
		}
		return err
	}

	// Update replica counts from deployment status
	tool.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	tool.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	tool.Status.UpdatedReplicas = deployment.Status.UpdatedReplicas
	tool.Status.UnavailableReplicas = deployment.Status.UnavailableReplicas

	// Determine phase based on deployment status.
	// When HPA is active, the desired count is managed by the HPA and stored in
	// deployment.Spec.Replicas. Use that instead of spec.deployment.replicas so
	// the Updating phase reflects the full rollout, not just the first pod.
	desiredReplicas := int32(1)
	if tool.Spec.Deployment.Autoscaling != nil && deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	} else if tool.Spec.Deployment.Replicas != nil {
		desiredReplicas = *tool.Spec.Deployment.Replicas
	}

	// Check if deployment is updating
	if deployment.Status.UpdatedReplicas < desiredReplicas {
		SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusUpdating, tool.Generation)
		SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonUpdating, "Deployment is updating", tool.Generation)
		return nil
	}

	// Check if any pods are ready
	if deployment.Status.ReadyReplicas > 0 {
		// Downgrade Running to Degraded when a non-critical subsystem has failed
		// (e.g. NetworkPolicy timed out). The tool is operational but at reduced capability.
		phase := events.PhaseStatusRunning
		for _, c := range tool.Status.Conditions {
			if c.Type == langopv1alpha1.ConditionNetworkPolicyReady && c.Status == metav1.ConditionFalse {
				phase = events.PhaseStatusDegraded
				break
			}
		}
		SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, phase, tool.Generation)
		tool.Status.Endpoint = serviceURL(tool.Name, tool.Namespace, tool.Spec.Port)
		SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue, langopv1alpha1.ReasonReconcileSuccess, "LanguageTool is ready", tool.Generation)

		// Discover MCP tool schemas for service mode tools.
		// Pass host:port (not the full URL) because discoverMCPToolSchemas prepends "http://".
		if tool.Spec.Type == "mcp" {
			hostPort := fmt.Sprintf("%s.%s.svc.cluster.local:%d", tool.Name, tool.Namespace, tool.Spec.Port)
			schemas, err := r.discoverMCPToolSchemas(ctx, hostPort)
			if err != nil {
				log := log.FromContext(ctx)
				log.Error(err, "Failed to discover MCP tool schemas", "tool", tool.Name, "endpoint", hostPort)
				SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionSchemasDiscovered, metav1.ConditionFalse, langopv1alpha1.ReasonSchemaDiscoveryFailed, err.Error(), tool.Generation)
			} else {
				tool.Status.ToolSchemas = schemas
				SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionSchemasDiscovered, metav1.ConditionTrue, langopv1alpha1.ReasonSchemasDiscovered, "Tool schemas discovered", tool.Generation)
			}
		}

		return nil
	}

	// No pods ready - check if deployment has been created recently
	if deployment.Status.AvailableReplicas == 0 && deployment.Status.UnavailableReplicas > 0 {
		// Pods exist but none are ready - likely CrashLoopBackOff or similar
		SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusFailed, tool.Generation)
		SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonPodsNotReady, "No pods are ready", tool.Generation)
		return nil
	}

	// Deployment exists but no replicas yet
	SetPhase(&tool.Status.Phase, &tool.Status.ObservedGeneration, events.PhaseStatusPending, tool.Generation)
	SetCondition(&tool.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonPending, "Waiting for pods to be scheduled", tool.Generation)
	return nil
}

func (r *LanguageToolReconciler) cleanupResources(ctx context.Context, tool *langopv1alpha1.LanguageTool) error {
	// Resources will be cleaned up automatically via owner references
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageToolReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageTool{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
