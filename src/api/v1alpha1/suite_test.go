//go:build integration

package v1alpha1_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	k8sClient  client.Client
	testEnv    *envtest.Environment
	testScheme *runtime.Scheme
	ctx        context.Context
	cancel     context.CancelFunc
)

func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	testScheme = buildScheme()

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{
				filepath.Join("..", "..", "config", "webhook"),
			},
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic("failed to start envtest: " + err.Error())
	}

	webhookOpts := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: testScheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookOpts.LocalServingHost,
			Port:    webhookOpts.LocalServingPort,
			CertDir: webhookOpts.LocalServingCertDir,
		}),
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	})
	if err != nil {
		panic("failed to create manager: " + err.Error())
	}

	// Register all webhooks against the manager's client (which has cluster membership access)
	if err := langopv1alpha1.SetupLanguageClusterWebhookWithManager(mgr); err != nil {
		panic("failed to setup LanguageCluster webhook: " + err.Error())
	}
	if err := langopv1alpha1.SetupLanguageAgentWebhookWithManager(mgr); err != nil {
		panic("failed to setup LanguageAgent webhook: " + err.Error())
	}
	if err := langopv1alpha1.SetupLanguageModelWebhookWithManager(mgr); err != nil {
		panic("failed to setup LanguageModel webhook: " + err.Error())
	}
	if err := langopv1alpha1.SetupLanguageToolWebhookWithManager(mgr); err != nil {
		panic("failed to setup LanguageTool webhook: " + err.Error())
	}
	if err := langopv1alpha1.SetupLanguagePersonaWebhookWithManager(mgr); err != nil {
		panic("failed to setup LanguagePersona webhook: " + err.Error())
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic("manager exited with error: " + err.Error())
		}
	}()

	// Wait for webhook server to be ready (up to 10s)
	addr := fmt.Sprintf("%s:%d", webhookOpts.LocalServingHost, webhookOpts.LocalServingPort)
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		panic("failed to create k8s client: " + err.Error())
	}

	code := m.Run()

	cancel()
	if err := testEnv.Stop(); err != nil {
		panic("failed to stop envtest: " + err.Error())
	}

	os.Exit(code)
}

func buildScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = langopv1alpha1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	_ = rbacv1.AddToScheme(s)
	return s
}

// TestWebhookClusterMembership verifies that the admission webhook enforces
// LanguageCluster namespace membership at create time.
func TestWebhookClusterMembership(t *testing.T) {
	// Cluster-managed namespace
	clusterNs := &corev1.Namespace{}
	clusterNs.Name = "webhook-test-cluster"
	if err := k8sClient.Create(ctx, clusterNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create cluster namespace: %v", err)
	}

	// Plain namespace — no LanguageCluster backing it
	plainNs := &corev1.Namespace{}
	plainNs.Name = "webhook-test-plain"
	if err := k8sClient.Create(ctx, plainNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create plain namespace: %v", err)
	}

	// LanguageCluster matching the cluster namespace
	cluster := gen.LanguageCluster("webhook-test-cluster", gen.SetClusterDomain("agents.example.com"))
	if err := k8sClient.Create(ctx, cluster); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create LanguageCluster: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, cluster) })

	t.Run("agent in cluster namespace is accepted", func(t *testing.T) {
		agent := gen.LanguageAgent("accepted-agent", "webhook-test-cluster",
			gen.SetAgentImage("ghcr.io/test/agent:latest"),
		)
		if err := k8sClient.Create(ctx, agent); err != nil {
			t.Errorf("expected success, got: %v", err)
		} else {
			t.Cleanup(func() { _ = k8sClient.Delete(ctx, agent) })
		}
	})

	t.Run("agent in non-cluster namespace is rejected", func(t *testing.T) {
		agent := gen.LanguageAgent("rejected-agent", "webhook-test-plain",
			gen.SetAgentImage("ghcr.io/test/agent:latest"),
		)
		err := k8sClient.Create(ctx, agent)
		if err == nil {
			t.Error("expected admission to reject the agent, but it was created")
			_ = k8sClient.Delete(ctx, agent)
		}
	})

	t.Run("webhook defaults workspace when not set", func(t *testing.T) {
		agent := gen.LanguageAgent("ws-defaulted-agent", "webhook-test-cluster",
			gen.SetAgentImage("ghcr.io/test/agent:latest"),
		)
		agent.Spec.Workspace = nil

		if err := k8sClient.Create(ctx, agent); err != nil {
			t.Fatalf("create agent: %v", err)
		}
		t.Cleanup(func() { _ = k8sClient.Delete(ctx, agent) })

		created := &langopv1alpha1.LanguageAgent{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name: "ws-defaulted-agent", Namespace: "webhook-test-cluster",
		}, created); err != nil {
			t.Fatalf("get agent: %v", err)
		}

		if created.Spec.Workspace == nil {
			t.Fatal("expected workspace to be defaulted, got nil")
		}
		if created.Spec.Workspace.Enabled != nil && !*created.Spec.Workspace.Enabled {
			t.Error("expected workspace.enabled=true")
		}
		if created.Spec.Workspace.Size == "" {
			t.Error("expected workspace.size to be set")
		}
	})
}

// TestWebhookClusterMembershipTool verifies that the admission webhook enforces
// LanguageCluster namespace membership for LanguageTool resources.
func TestWebhookClusterMembershipTool(t *testing.T) {
	clusterNs := &corev1.Namespace{}
	clusterNs.Name = "webhook-tool-cluster"
	if err := k8sClient.Create(ctx, clusterNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create cluster namespace: %v", err)
	}

	plainNs := &corev1.Namespace{}
	plainNs.Name = "webhook-tool-plain"
	if err := k8sClient.Create(ctx, plainNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create plain namespace: %v", err)
	}

	cluster := gen.LanguageCluster("webhook-tool-cluster", gen.SetClusterDomain("tools.example.com"))
	if err := k8sClient.Create(ctx, cluster); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create LanguageCluster: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, cluster) })

	t.Run("tool in cluster namespace is accepted", func(t *testing.T) {
		tool := gen.LanguageTool("accepted-tool", "webhook-tool-cluster",
			gen.SetToolImage("ghcr.io/test/tool:latest"),
		)
		if err := k8sClient.Create(ctx, tool); err != nil {
			t.Errorf("expected success, got: %v", err)
		} else {
			t.Cleanup(func() { _ = k8sClient.Delete(ctx, tool) })
		}
	})

	t.Run("tool in non-cluster namespace is rejected", func(t *testing.T) {
		tool := gen.LanguageTool("rejected-tool", "webhook-tool-plain",
			gen.SetToolImage("ghcr.io/test/tool:latest"),
		)
		err := k8sClient.Create(ctx, tool)
		if err == nil {
			t.Error("expected admission to reject the tool, but it was created")
			_ = k8sClient.Delete(ctx, tool)
		}
	})

}

// TestWebhookClusterMembershipModel verifies that the admission webhook enforces
// LanguageCluster namespace membership for LanguageModel resources.
func TestWebhookClusterMembershipModel(t *testing.T) {
	clusterNs := &corev1.Namespace{}
	clusterNs.Name = "webhook-model-cluster"
	if err := k8sClient.Create(ctx, clusterNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create cluster namespace: %v", err)
	}

	plainNs := &corev1.Namespace{}
	plainNs.Name = "webhook-model-plain"
	if err := k8sClient.Create(ctx, plainNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create plain namespace: %v", err)
	}

	cluster := gen.LanguageCluster("webhook-model-cluster", gen.SetClusterDomain("models.example.com"))
	if err := k8sClient.Create(ctx, cluster); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create LanguageCluster: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, cluster) })

	t.Run("model in cluster namespace is accepted", func(t *testing.T) {
		model := gen.LanguageModel("accepted-model", "webhook-model-cluster",
			gen.SetModelProvider("anthropic"),
		)
		if err := k8sClient.Create(ctx, model); err != nil {
			t.Errorf("expected success, got: %v", err)
		} else {
			t.Cleanup(func() { _ = k8sClient.Delete(ctx, model) })
		}
	})

	t.Run("model in non-cluster namespace is rejected", func(t *testing.T) {
		model := gen.LanguageModel("rejected-model", "webhook-model-plain",
			gen.SetModelProvider("anthropic"),
		)
		err := k8sClient.Create(ctx, model)
		if err == nil {
			t.Error("expected admission to reject the model, but it was created")
			_ = k8sClient.Delete(ctx, model)
		}
	})

}

// TestWebhookClusterMembershipPersona verifies that the admission webhook enforces
// LanguageCluster namespace membership for LanguagePersona resources.
func TestWebhookClusterMembershipPersona(t *testing.T) {
	clusterNs := &corev1.Namespace{}
	clusterNs.Name = "webhook-persona-cluster"
	if err := k8sClient.Create(ctx, clusterNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create cluster namespace: %v", err)
	}

	plainNs := &corev1.Namespace{}
	plainNs.Name = "webhook-persona-plain"
	if err := k8sClient.Create(ctx, plainNs); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create plain namespace: %v", err)
	}

	cluster := gen.LanguageCluster("webhook-persona-cluster", gen.SetClusterDomain("personas.example.com"))
	if err := k8sClient.Create(ctx, cluster); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create LanguageCluster: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, cluster) })

	t.Run("persona in cluster namespace is accepted", func(t *testing.T) {
		persona := gen.LanguagePersona("accepted-persona", "webhook-persona-cluster",
			gen.SetPersonaPersonality("You are a helpful assistant."),
		)
		if err := k8sClient.Create(ctx, persona); err != nil {
			t.Errorf("expected success, got: %v", err)
		} else {
			t.Cleanup(func() { _ = k8sClient.Delete(ctx, persona) })
		}
	})

	t.Run("persona in non-cluster namespace is rejected", func(t *testing.T) {
		persona := gen.LanguagePersona("rejected-persona", "webhook-persona-plain",
			gen.SetPersonaPersonality("You are a helpful assistant."),
		)
		err := k8sClient.Create(ctx, persona)
		if err == nil {
			t.Error("expected admission to reject the persona, but it was created")
			_ = k8sClient.Delete(ctx, persona)
		}
	})

}
