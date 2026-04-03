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
	"strconv"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	webhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers"
	"github.com/language-operator/language-operator/pkg/cni"
	registryconfig "github.com/language-operator/language-operator/pkg/config"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/telemetry"
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

// getEnvInt reads an integer from environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
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
	var agentIngressClassName string
	var agentStorageClassName string
	var gatewayIngressClassName string
	var gatewayImage string
	var gatewayImagePullPolicy string
	var webhookPort int
	var webhookCertDir string
	var disableWebhooks bool

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
	flag.StringVar(&agentIngressClassName, "agent-ingress-class-name", "",
		"Default IngressClass name for agent Ingress resources. Can be overridden per LanguageCluster.")
	flag.StringVar(&agentStorageClassName, "agent-storage-class-name", "",
		"Default StorageClass for agent workspace PVCs. Uses cluster default when empty. Can be overridden per agent via spec.workspace.storageClassName.")
	flag.StringVar(&gatewayIngressClassName, "gateway-ingress-class-name", "",
		"Default IngressClass name for the gateway Ingress. Can be overridden per LanguageCluster.")
	var ingressControllerNamespace string
	flag.StringVar(&ingressControllerNamespace, "ingress-controller-namespace", "",
		"Namespace the ingress controller runs in. When set, a NetworkPolicy ingress rule is added to allow traffic from that namespace to agent ports.")
	flag.StringVar(&gatewayImage, "gateway-image", "",
		"Image for the shared LiteLLM gateway. Defaults to ghcr.io/language-operator/model-gateway:latest.")
	flag.StringVar(&gatewayImagePullPolicy, "gateway-image-pull-policy", "",
		"ImagePullPolicy for the shared LiteLLM gateway (Always, IfNotPresent, Never).")
	var tlsIssuerName string
	flag.StringVar(&tlsIssuerName, "tls-issuer-name", "",
		"cert-manager issuer name used to provision TLS certificates for gateway and agent Ingress resources. Empty disables cert-manager integration.")
	var tlsIssuerKind string
	flag.StringVar(&tlsIssuerKind, "tls-issuer-kind", "ClusterIssuer",
		"Kind of the cert-manager issuer (ClusterIssuer or Issuer). Defaults to ClusterIssuer.")
	flag.IntVar(&webhookPort, "webhook-port", 9443,
		"Port the webhook server listens on.")
	flag.StringVar(&webhookCertDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs",
		"Directory containing TLS certificates for the webhook server.")
	flag.BoolVar(&disableWebhooks, "disable-webhooks", false,
		"Disable webhook server and all admission webhooks. Use when cert-manager is not installed.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Read network isolation configuration from environment
	networkIsolationEnabled := os.Getenv("LANGOP_NETWORK_ISOLATION_ENABLED") != "false"

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
	var namespaces map[string]cache.Config
	if watchNamespaces != "" {
		namespaces = make(map[string]cache.Config)
		for _, ns := range parseNamespaces(watchNamespaces) {
			namespaces[ns] = cache.Config{}
		}
		setupLog.Info("Watching specific namespaces", "namespaces", namespaces)
	} else {
		setupLog.Info("Watching all namespaces")
	}

	mgrOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "langop.io",
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		RetryPeriod:            &retryPeriod,
		Cache: cache.Options{
			DefaultNamespaces: namespaces,
			SyncPeriod:        &syncPeriod,
		},
	}
	if !disableWebhooks {
		mgrOptions.WebhookServer = webhook.NewServer(webhook.Options{Port: webhookPort, CertDir: webhookCertDir})
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Auto-detect ingress controller namespace if not explicitly configured.
	if ingressControllerNamespace == "" && agentIngressClassName != "" {
		ingressControllerNamespace = registryconfig.DetectIngressControllerNamespace(ctx, mgr.GetClient(), agentIngressClassName)
		if ingressControllerNamespace != "" {
			setupLog.Info("Auto-detected ingress controller namespace", "namespace", ingressControllerNamespace, "ingressClass", agentIngressClassName)
		} else {
			setupLog.Info("Could not auto-detect ingress controller namespace; NetworkPolicy will not include ingress controller rule")
		}
	}

	// Setup LanguageTool controller
	if err = (&controllers.LanguageToolReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Log:                     ctrl.Log.WithName("controllers").WithName("LanguageTool"),
		RegistryManager:         registryManager,
		Recorder:                mgr.GetEventRecorderFor("languagetool-controller"),
		EventManager:            events.NewEventManager(mgr.GetEventRecorderFor("languagetool-controller")),
		NetworkIsolationEnabled: networkIsolationEnabled,
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageTool")
		os.Exit(1)
	}

	// Setup LanguageModel controller
	if err = (&controllers.LanguageModelReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Log:                     ctrl.Log.WithName("controllers").WithName("LanguageModel"),
		Recorder:                mgr.GetEventRecorderFor("languagemodel-controller"),
		EventManager:            events.NewEventManager(mgr.GetEventRecorderFor("languagemodel-controller")),
		NetworkIsolationEnabled: networkIsolationEnabled,
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageModel")
		os.Exit(1)
	}

	// Setup LanguageAgent controller with optional synthesizer
	agentReconciler := &controllers.LanguageAgentReconciler{
		Client:                     mgr.GetClient(),
		Scheme:                     mgr.GetScheme(),
		Log:                        ctrl.Log.WithName("controllers").WithName("LanguageAgent"),
		Recorder:                   mgr.GetEventRecorderFor("languageagent-controller"),
		EventManager:               events.NewEventManager(mgr.GetEventRecorderFor("languageagent-controller")),
		RegistryManager:            registryManager,
		NetworkPolicyTimeout:       networkPolicyTimeout,
		NetworkPolicyRetries:       networkPolicyRetries,
		NetworkIsolationEnabled:    networkIsolationEnabled,
		DefaultIngressClassName:    agentIngressClassName,
		DefaultStorageClassName:    agentStorageClassName,
		DefaultTLSIssuerName:       tlsIssuerName,
		DefaultTLSIssuerKind:       tlsIssuerKind,
		IngressControllerNamespace: ingressControllerNamespace,
		CNICapabilities:            cniCaps,
	}

	if err = agentReconciler.SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageAgent")
		os.Exit(1)
	}

	// Setup LanguagePersona controller
	if err = (&controllers.LanguagePersonaReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		Log:          ctrl.Log.WithName("controllers").WithName("LanguagePersona"),
		Recorder:     mgr.GetEventRecorderFor("languagepersona-controller"),
		EventManager: events.NewEventManager(mgr.GetEventRecorderFor("languagepersona-controller")),
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguagePersona")
		os.Exit(1)
	}

	// Setup LanguageAgentRuntime controller
	if err = (&controllers.LanguageAgentRuntimeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LanguageAgentRuntime"),
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageAgentRuntime")
		os.Exit(1)
	}

	// Setup LanguageCluster controller
	if err = (&controllers.LanguageClusterReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		Log:                     ctrl.Log.WithName("controllers").WithName("LanguageCluster"),
		Recorder:                mgr.GetEventRecorderFor("languagecluster-controller"),
		EventManager:            events.NewEventManager(mgr.GetEventRecorderFor("languagecluster-controller")),
		NetworkIsolationEnabled: networkIsolationEnabled,
		GatewayImage:            gatewayImage,
		GatewayImagePullPolicy:  corev1.PullPolicy(gatewayImagePullPolicy),
		DefaultIngressClassName: gatewayIngressClassName,
		DefaultTLSIssuerName:    tlsIssuerName,
		DefaultTLSIssuerKind:    tlsIssuerKind,
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageCluster")
		os.Exit(1)
	}

	if !disableWebhooks {
		// Setup LanguageAgent webhook
		if err = langopv1alpha1.SetupLanguageAgentWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "LanguageAgent")
			os.Exit(1)
		}
		setupLog.Info("LanguageAgent webhook registered")

		// Setup LanguageTool webhook
		if err = langopv1alpha1.SetupLanguageToolWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "LanguageTool")
			os.Exit(1)
		}
		setupLog.Info("LanguageTool webhook registered")

		// Setup LanguageModel webhook
		if err = langopv1alpha1.SetupLanguageModelWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "LanguageModel")
			os.Exit(1)
		}
		setupLog.Info("LanguageModel webhook registered")

		// Setup LanguagePersona webhook
		if err = langopv1alpha1.SetupLanguagePersonaWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "LanguagePersona")
			os.Exit(1)
		}
		setupLog.Info("LanguagePersona webhook registered")
	} else {
		setupLog.Info("Webhooks disabled via --disable-webhooks flag")
	}
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
