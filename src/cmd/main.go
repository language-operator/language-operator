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
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	webhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/cloudwego/eino-ext/components/model/openai"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	"github.com/based/language-operator/controllers"
	"github.com/based/language-operator/pkg/synthesis"
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

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8443", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
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
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LanguageTool"),
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
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("LanguageAgent"),
		Recorder: mgr.GetEventRecorderFor("languageagent-controller"),
	}

	// Initialize synthesizer if LLM configuration is provided
	synthesisModel := os.Getenv("SYNTHESIS_MODEL")
	synthesisAPIKey := os.Getenv("SYNTHESIS_API_KEY")
	synthesisEndpoint := os.Getenv("SYNTHESIS_ENDPOINT") // For custom OpenAI-compatible endpoints

	if synthesisModel != "" {
		setupLog.Info("Initializing synthesis engine", "model", synthesisModel, "endpoint", synthesisEndpoint)

		// Default API key for local endpoints that don't validate it
		if synthesisAPIKey == "" {
			synthesisAPIKey = "sk-local-not-needed"
		}

		// Create eino OpenAI ChatModel config
		config := &openai.ChatModelConfig{
			Model:  synthesisModel,
			APIKey: synthesisAPIKey,
		}

		// Set custom BaseURL if provided (for LM Studio, Ollama with OpenAI compat, etc.)
		if synthesisEndpoint != "" {
			config.BaseURL = synthesisEndpoint
			setupLog.Info("Using custom OpenAI-compatible endpoint", "baseURL", synthesisEndpoint)
		}

		// Set temperature for consistent code generation
		temp := float32(0.3)
		config.Temperature = &temp

		// Set max tokens
		maxTokens := 4096
		config.MaxTokens = &maxTokens

		// Create ChatModel
		ctx := context.Background()
		chatModel, err := openai.NewChatModel(ctx, config)
		if err != nil {
			setupLog.Error(err, "failed to create synthesis ChatModel")
			os.Exit(1)
		}

		// Create synthesizer
		synthesizer := synthesis.NewSynthesizer(chatModel, ctrl.Log.WithName("synthesis"))
		agentReconciler.Synthesizer = synthesizer
		agentReconciler.SynthesisModel = synthesisModel
		setupLog.Info("Synthesis engine initialized successfully")
	} else {
		setupLog.Info("Synthesis engine disabled (SYNTHESIS_MODEL not set)")
	}

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

	// Setup LanguageClient controller
	if err = (&controllers.LanguageClientReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LanguageClient"),
	}).SetupWithManager(mgr, concurrency); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LanguageClient")
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

	// Setup LanguageCluster webhook
	if err = (&langopv1alpha1.LanguageCluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "LanguageCluster")
		os.Exit(1)
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
