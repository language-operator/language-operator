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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	webhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers"
	"github.com/language-operator/language-operator/pkg/cni"
	registryconfig "github.com/language-operator/language-operator/pkg/config"
	"github.com/language-operator/language-operator/pkg/learning"
	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/language-operator/language-operator/pkg/telemetry"
	"github.com/language-operator/language-operator/pkg/telemetry/adapters"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(langopv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// initializeTelemetryAdapter creates and initializes a telemetry adapter based on environment configuration
func initializeTelemetryAdapter() telemetry.TelemetryAdapter {
	adapterType := os.Getenv("TELEMETRY_ADAPTER_TYPE")

	// Default to NoOpAdapter if no type specified
	if adapterType == "" {
		setupLog.Info("No telemetry adapter type specified, using NoOpAdapter")
		return telemetry.NewNoOpAdapter()
	}

	switch strings.ToLower(adapterType) {
	case "signoz":
		return initializeSigNozAdapter()
	case "noop", "disabled":
		setupLog.Info("Telemetry adapter explicitly disabled")
		return telemetry.NewNoOpAdapter()
	default:
		setupLog.Info("Unknown telemetry adapter type, falling back to NoOpAdapter",
			"type", adapterType)
		return telemetry.NewNoOpAdapter()
	}
}

// initializeSigNozAdapter creates a SigNoz telemetry adapter from environment variables
func initializeSigNozAdapter() telemetry.TelemetryAdapter {
	endpoint := os.Getenv("TELEMETRY_ADAPTER_ENDPOINT")
	apiKey := os.Getenv("TELEMETRY_ADAPTER_API_KEY")

	// Validate required configuration
	if endpoint == "" {
		setupLog.Error(nil, "SigNoz adapter requires TELEMETRY_ADAPTER_ENDPOINT environment variable")
		return telemetry.NewNoOpAdapter()
	}

	if apiKey == "" {
		setupLog.Error(nil, "SigNoz adapter requires TELEMETRY_ADAPTER_API_KEY environment variable")
		return telemetry.NewNoOpAdapter()
	}

	// Parse timeout with default
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv("TELEMETRY_ADAPTER_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		} else {
			setupLog.Error(err, "Invalid TELEMETRY_ADAPTER_TIMEOUT, using default 30s", "value", timeoutStr)
		}
	}

	// Create SigNoz adapter
	adapter, err := adapters.NewSignozAdapter(endpoint, apiKey, timeout)
	if err != nil {
		setupLog.Error(err, "Failed to create SigNoz telemetry adapter, falling back to NoOpAdapter")
		return telemetry.NewNoOpAdapter()
	}

	// Log configuration details (excluding sensitive information)
	setupLog.Info("SigNoz telemetry adapter initialized successfully",
		"endpoint", endpoint,
		"timeout", timeout,
		"retryAttempts", getEnvOrDefault("TELEMETRY_ADAPTER_RETRY_ATTEMPTS", "3"),
		"retryBackoff", getEnvOrDefault("TELEMETRY_ADAPTER_RETRY_BACKOFF", "1s"),
		"maxTraces", getEnvOrDefault("TELEMETRY_ADAPTER_MAX_TRACES", "100"),
		"lookbackPeriod", getEnvOrDefault("TELEMETRY_ADAPTER_LOOKBACK_PERIOD", "24h"),
		"queryTimeout", getEnvOrDefault("TELEMETRY_ADAPTER_QUERY_TIMEOUT", "10s"),
		"healthCheckEnabled", getEnvOrDefault("TELEMETRY_ADAPTER_HEALTH_CHECK_ENABLED", "true"),
		"healthCheckInterval", getEnvOrDefault("TELEMETRY_ADAPTER_HEALTH_CHECK_INTERVAL", "5m"))

	return adapter
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var leaseDuration time.Duration
	var renewDeadline time.Duration
	var retryPeriod time.Duration
	var syncPeriod time.Duration
	var watchNamespaces string
	var concurrency int
	var requireNetworkPolicy bool
	var networkPolicyTimeout time.Duration
	var networkPolicyRetries int

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8443", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&requireNetworkPolicy, "require-network-policy", false,
		"Fail operator startup if CNI does not support NetworkPolicy enforcement. "+
			"Default is false to allow operation on local/development clusters.")
	flag.DurationVar(&networkPolicyTimeout, "network-policy-timeout", 30*time.Second,
		"Timeout for NetworkPolicy operations. Increase for slow CNI plugins.")
	flag.IntVar(&networkPolicyRetries, "network-policy-retries", 3,
		"Number of retry attempts for NetworkPolicy operations.")
	flag.DurationVar(&leaseDuration, "leader-elect-lease-duration", 15*time.Second,
		"The duration that non-leader candidates will wait after observing a leadership renewal.")
	flag.DurationVar(&renewDeadline, "leader-elect-renew-deadline", 10*time.Second,
		"The interval between attempts by the acting leader to renew a leadership slot.")
	flag.DurationVar(&retryPeriod, "leader-elect-retry-period", 2*time.Second,
		"The duration the clients should wait between attempting acquisition and renewal of a leadership.")
	flag.DurationVar(&syncPeriod, "sync-period", 10*time.Minute,
		"The resync period for controllers.")
	flag.StringVar(&watchNamespaces, "watch-namespaces", "",
		"Comma-separated list of namespaces to watch. Empty means all namespaces.")
	flag.IntVar(&concurrency, "concurrency", 5,
		"The number of concurrent reconciles per controller.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Initialize OpenTelemetry tracing with startup timeout
	startupTimeout := 60 * time.Second
	if timeoutStr := os.Getenv("STARTUP_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			startupTimeout = parsedTimeout
		} else {
			setupLog.Error(err, "Invalid STARTUP_TIMEOUT, using default 60s", "value", timeoutStr)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	setupLog.Info("Startup operations timeout configured", "timeout", startupTimeout)

	tracerProvider, err := telemetry.InitTracer(ctx)
	if err != nil {
		setupLog.Error(err, "failed to initialize OpenTelemetry, tracing disabled")
	} else if tracerProvider != nil {
		setupLog.Info("OpenTelemetry tracing enabled")
		// Defer shutdown with timeout
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := telemetry.Shutdown(shutdownCtx, tracerProvider); err != nil {
				setupLog.Error(err, "failed to shutdown OpenTelemetry TracerProvider")
			}
		}()
	} else {
		setupLog.Info("OpenTelemetry tracing disabled (OTEL_EXPORTER_OTLP_ENDPOINT not set)")
	}

	// Detect CNI capabilities and load registry whitelist before starting manager
	config := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "unable to create kubernetes clientset for CNI detection")
		os.Exit(1)
	}

	// Create timeout context for CNI detection
	cniCtx, cniCancel := context.WithTimeout(ctx, networkPolicyTimeout)
	defer cniCancel()

	cniCaps, cniErr := cni.DetectNetworkPolicySupport(cniCtx, clientset)
	if cniErr != nil && errors.Is(cniErr, context.DeadlineExceeded) {
		setupLog.Error(cniErr, "CNI detection timed out", "timeout", networkPolicyTimeout)
		cniErr = fmt.Errorf("CNI detection timed out after %v - CNI may still be initializing", networkPolicyTimeout)
	}

	// Initialize registry configuration manager for dynamic configuration updates
	registryManager := registryconfig.NewRegistryConfigManager(clientset)
	if err := registryManager.StartWatcher(ctx); err != nil {
		setupLog.Error(err, "failed to start registry configuration watcher")
		os.Exit(1)
	}
	defer registryManager.Stop()

	setupLog.Info("Registry configuration manager started", "registries", registryManager.GetRegistries())

	// Validate schema compatibility between operator and gem
	setupLog.Info("Checking schema compatibility with language_operator gem")
	synthesis.ValidateSchemaCompatibility(ctx, setupLog)

	if cniErr != nil {
		setupLog.Info("CNI detection failed", "error", cniErr.Error())
		if requireNetworkPolicy {
			setupLog.Error(cniErr, "CNI detection is required but failed")
			os.Exit(1)
		}
	}

	if cniCaps != nil {
		if cniCaps.SupportsNetworkPolicy {
			setupLog.Info("CNI detected with NetworkPolicy support",
				"cni", cniCaps.Name,
				"version", cniCaps.Version,
				"networkPolicy", "supported")
			setupLog.Info("Network isolation will be enforced for LanguageAgent pods")
		} else {
			setupLog.Info("WARNING: CNI does not support NetworkPolicy enforcement",
				"cni", cniCaps.Name,
				"version", cniCaps.Version,
				"networkPolicy", "not supported")
			setupLog.Info("Impact: Network isolation for LanguageAgent pods will NOT be enforced")
			setupLog.Info("Agents will be able to make unrestricted network connections")
			setupLog.Info("For production use, consider installing a NetworkPolicy-capable CNI:")
			setupLog.Info("  - Cilium (recommended): kubectl apply -f https://raw.githubusercontent.com/cilium/cilium/v1.18/install/kubernetes/quick-install.yaml")
			setupLog.Info("  - Calico: https://docs.tigera.io/calico/latest/getting-started/kubernetes/quickstart")
			setupLog.Info("  - Weave Net: kubectl apply -f https://github.com/weaveworks/weave/releases/download/v2.8.1/weave-daemonset-k8s.yaml")
			setupLog.Info("  - Antrea: https://antrea.io/docs/main/docs/getting-started/")

			if requireNetworkPolicy {
				setupLog.Error(nil, "NetworkPolicy support is required but CNI does not support it",
					"cni", cniCaps.Name)
				os.Exit(1)
			}
		}
	}

	// Parse watch namespaces
	var namespaces map[string]struct{}
	if watchNamespaces != "" {
		namespaces = make(map[string]struct{})
		for _, ns := range parseNamespaces(watchNamespaces) {
			namespaces[ns] = struct{}{}
		}
		setupLog.Info("Watching specific namespaces", "namespaces", namespaces)
	} else {
		setupLog.Info("Watching all namespaces")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "langop.io",
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		RetryPeriod:            &retryPeriod,
		//Cache: cache.Options{
		//	DefaultNamespaces: namespaces,
		//	SyncPeriod:        &syncPeriod,
		//},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup LanguageTool controller
	if err = (&controllers.LanguageToolReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Log:             ctrl.Log.WithName("controllers").WithName("LanguageTool"),
		RegistryManager: registryManager,
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageTool")
		os.Exit(1)
	}

	// Setup LanguageModel controller
	if err = (&controllers.LanguageModelReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LanguageModel"),
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageModel")
		os.Exit(1)
	}

	// Setup LanguageAgent controller with optional synthesizer
	agentReconciler := &controllers.LanguageAgentReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		Log:                  ctrl.Log.WithName("controllers").WithName("LanguageAgent"),
		Recorder:             mgr.GetEventRecorderFor("languageagent-controller"),
		RegistryManager:      registryManager,
		NetworkPolicyTimeout: networkPolicyTimeout,
		NetworkPolicyRetries: networkPolicyRetries,
	}

	// Initialize Gateway API cache
	agentReconciler.InitializeGatewayCache()

	// Initialize rate limiter and quota manager for synthesis cost controls
	maxSynthesisPerHour := 500 // Default: 500 synthesis per namespace per hour
	rateLimiter := synthesis.NewRateLimiter(maxSynthesisPerHour, ctrl.Log.WithName("rate-limiter"))
	agentReconciler.RateLimiter = rateLimiter
	setupLog.Info("Synthesis rate limiter initialized", "maxPerHour", maxSynthesisPerHour)

	maxCostPerDay := 10.0    // Default: $10 per namespace per day
	maxAttemptsPerDay := 100 // Default: 100 attempts per namespace per day
	quotaManager := synthesis.NewQuotaManager(maxCostPerDay, maxAttemptsPerDay, "USD", ctrl.Log.WithName("quota-manager"))
	agentReconciler.QuotaManager = quotaManager
	setupLog.Info("Synthesis quota manager initialized", "maxCostPerDay", maxCostPerDay, "maxAttemptsPerDay", maxAttemptsPerDay)

	// Synthesis is now configured per-agent via ModelRefs - no global setup needed
	setupLog.Info("Synthesis engine uses per-agent ModelRefs configuration")

	if err = agentReconciler.SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageAgent")
		os.Exit(1)
	}

	// Setup LanguagePersona controller
	if err = (&controllers.LanguagePersonaReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LanguagePersona"),
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguagePersona")
		os.Exit(1)
	}

	// Setup LanguageCluster controller
	if err = (&controllers.LanguageClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LanguageCluster"),
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageCluster")
		os.Exit(1)
	}

	// Setup Learning controller with metrics collection
	learningLog := ctrl.Log.WithName("controllers").WithName("Learning")
	metricsCollector := learning.NewMetricsCollector(learningLog)
	eventProcessor := learning.NewLearningEventProcessor(metricsCollector)

	configMapManager := &synthesis.ConfigMapManager{
		Client: mgr.GetClient(),
		Log:    learningLog.WithName("configmap"),
	}

	// Initialize telemetry adapter for learning system
	telemetryAdapter := initializeTelemetryAdapter()

	if err = (&controllers.LearningReconciler{
		Client:                      mgr.GetClient(),
		Scheme:                      mgr.GetScheme(),
		Log:                         learningLog,
		Recorder:                    mgr.GetEventRecorderFor("learning-controller"),
		ConfigMapManager:            configMapManager,
		MetricsCollector:            metricsCollector,
		EventProcessor:              eventProcessor,
		TelemetryAdapter:            telemetryAdapter,
		SuccessRateAggregator:       make(map[string]*learning.LearningSuccessRateAggregator),
		LearningEnabled:             true,
		LearningThreshold:           10,              // Trigger learning after 10 traces
		LearningInterval:            5 * time.Minute, // 5 minute cooldown between attempts
		MaxVersions:                 5,               // Keep last 5 ConfigMap versions
		PatternConfidenceMin:        0.8,             // Require 80% confidence
		ErrorFailureThreshold:       3,               // Re-synthesize after 3 consecutive failures
		ErrorCooldownPeriod:         5 * time.Minute, // 5 minute cooldown for error re-synthesis
		MaxErrorResynthesisAttempts: 3,               // Max 3 error re-synthesis attempts per task
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Learning")
		os.Exit(1)
	}

	// Setup LanguageCluster webhook
	if err = (&langopv1alpha1.LanguageCluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "LanguageCluster")
		os.Exit(1)
	}

	// Setup LanguageAgent webhook for synthesis cost controls
	if err = (&langopv1alpha1.LanguageAgent{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "LanguageAgent")
		os.Exit(1)
	}
	setupLog.Info("LanguageAgent validation webhook registered")
	//+kubebuilder:scaffold:builder

	// Add health and readiness checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func parseNamespaces(namespaces string) []string {
	var result []string
	for _, ns := range splitAndTrim(namespaces, ",") {
		if ns != "" {
			result = append(result, ns)
		}
	}
	return result
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		if trimmed := trimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// loadAllowedRegistries loads the allowed container registries from the operator-config ConfigMap
// with strict validation to prevent configuration drift from unknown fields
func loadAllowedRegistries(ctx context.Context, clientset kubernetes.Interface) ([]string, error) {
	// Get operator namespace from environment (set by k8s downward API or default to kube-system)
	operatorNamespace := os.Getenv("OPERATOR_NAMESPACE")
	if operatorNamespace == "" {
		operatorNamespace = "kube-system" // Default namespace for the operator
	}

	// Get the ConfigMap
	configMap, err := clientset.CoreV1().ConfigMaps(operatorNamespace).Get(ctx, "operator-config", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator-config ConfigMap: %w", err)
	}

	// Validate ConfigMap structure - reject unknown fields to prevent configuration drift
	if err := validateOperatorConfigMapSchema(configMap.Data); err != nil {
		return nil, fmt.Errorf("invalid operator-config ConfigMap structure: %w", err)
	}

	// Parse the allowed-registries data
	registriesData, ok := configMap.Data["allowed-registries"]
	if !ok {
		return nil, fmt.Errorf("allowed-registries key not found in ConfigMap")
	}

	// Split by newlines and filter empty lines
	var registries []string
	for _, line := range splitAndTrim(registriesData, "\n") {
		line = trimSpace(line)
		// Skip empty lines and comments
		if line != "" && !hasPrefix(line, "#") {
			registries = append(registries, line)
		}
	}

	if len(registries) == 0 {
		return nil, fmt.Errorf("no registries found in ConfigMap")
	}

	return registries, nil
}

// validateOperatorConfigMapSchema validates that the operator-config ConfigMap contains only supported fields
// This prevents configuration drift and security issues from unknown fields like invalid version fields
func validateOperatorConfigMapSchema(data map[string]string) error {
	// Define supported fields for operator-config ConfigMap
	supportedFields := map[string]bool{
		"allowed-registries": true,
	}

	// Check for unknown fields
	var unknownFields []string
	for field := range data {
		if !supportedFields[field] {
			unknownFields = append(unknownFields, field)
		}
	}

	// Reject ConfigMap with unknown fields
	if len(unknownFields) > 0 {
		return fmt.Errorf("unsupported fields found: %v. operator-config ConfigMap only supports: allowed-registries", unknownFields)
	}

	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
