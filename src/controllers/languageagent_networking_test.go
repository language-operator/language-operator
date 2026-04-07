package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/cni"
	"github.com/language-operator/language-operator/pkg/events"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// --- Group 2: NetworkPolicy ---

func TestLanguageAgentController_NetworkPolicy(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "np-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:               fakeClient,
		Scheme:               scheme,
		Log:                  logr.Discard(),
		Recorder:             &record.FakeRecorder{},
		RegistryManager:      &mockRegistryManager{},
		NetworkPolicyTimeout: 30 * time.Second,
		NetworkPolicyRetries: 3,
	}

	ctx := context.Background()
	if err := reconciler.reconcileNetworkPolicy(ctx, agent); err != nil {
		t.Fatalf("reconcileNetworkPolicy failed: %v", err)
	}

	t.Run("network_policy_created", func(t *testing.T) {
		np := &networkingv1.NetworkPolicy{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np); err != nil {
			t.Fatalf("expected NetworkPolicy to exist: %v", err)
		}
	})

	t.Run("network_policy_has_ingress_rules", func(t *testing.T) {
		np := &networkingv1.NetworkPolicy{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np); err != nil {
			t.Fatalf("NetworkPolicy not found: %v", err)
		}
		if len(np.Spec.Ingress) == 0 {
			t.Error("expected ingress rules to be set on NetworkPolicy")
		}
		hasIngress := false
		for _, pt := range np.Spec.PolicyTypes {
			if pt == networkingv1.PolicyTypeIngress {
				hasIngress = true
			}
		}
		if !hasIngress {
			t.Error("expected PolicyTypeIngress to be in PolicyTypes")
		}
	})
}

func TestLanguageAgentController_NetworkPolicy_FromRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("from-rule-agent", "default",
		gen.SetAgentNetworkPolicies(&langopv1alpha1.AgentNetworkPolicies{
			Ingress: []langopv1alpha1.NetworkIngressRule{
				{
					From:  []langopv1alpha1.NetworkPeer{{Group: "monitoring"}},
					Ports: []langopv1alpha1.NetworkPort{{Port: 9090}},
				},
			},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:               fakeClient,
		Scheme:               scheme,
		Log:                  logr.Discard(),
		Recorder:             &record.FakeRecorder{},
		RegistryManager:      &mockRegistryManager{},
		NetworkPolicyTimeout: 30 * time.Second,
		NetworkPolicyRetries: 3,
	}

	ctx := context.Background()
	if err := reconciler.reconcileNetworkPolicy(ctx, agent); err != nil {
		t.Fatalf("reconcileNetworkPolicy failed: %v", err)
	}

	np := &networkingv1.NetworkPolicy{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np); err != nil {
		t.Fatalf("NetworkPolicy not found: %v", err)
	}

	t.Run("from_rule_appended_to_ingress", func(t *testing.T) {
		// Hardcoded rules: trigger, agent-to-agent = 2; user From rule = 1; total >= 3
		if len(np.Spec.Ingress) < 3 {
			t.Errorf("expected at least 3 ingress rules (2 default + 1 from spec), got %d", len(np.Spec.Ingress))
		}
		// Last rule should be the user-defined one
		last := np.Spec.Ingress[len(np.Spec.Ingress)-1]
		if len(last.From) == 0 {
			t.Fatal("expected From peers in user-defined ingress rule")
		}
		peer := last.From[0]
		if peer.PodSelector == nil {
			t.Fatal("expected PodSelector for Group-based From rule")
		}
		if peer.PodSelector.MatchLabels[langoplabels.LabelKeyLangopGroup] != "monitoring" {
			t.Errorf("expected langop.io/group=monitoring, got %v", peer.PodSelector.MatchLabels)
		}
		if len(last.Ports) == 0 || last.Ports[0].Port.IntVal != 9090 {
			t.Error("expected port 9090 in user-defined ingress rule")
		}
	})
}

// --- Group 3: Ingress ---

func TestLanguageAgentController_IngressCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}

	t.Run("ingress_with_class_name", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()

		reconciler := &LanguageAgentReconciler{
			Client:                  fakeClient,
			Scheme:                  scheme,
			Log:                     logr.Discard(),
			Recorder:                &record.FakeRecorder{},
			RegistryManager:         &mockRegistryManager{},
			DefaultIngressClassName: "nginx",
		}

		ctx := context.Background()
		if err := reconciler.reconcileIngress(ctx, agent, "agent.example.com"); err != nil {
			t.Fatalf("reconcileIngress failed: %v", err)
		}

		ing := &networkingv1.Ingress{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing); err != nil {
			t.Fatalf("expected Ingress to exist: %v", err)
		}
		if ing.Spec.IngressClassName == nil || *ing.Spec.IngressClassName != "nginx" {
			t.Errorf("expected IngressClassName 'nginx', got %v", ing.Spec.IngressClassName)
		}
	})

	t.Run("ingress_without_class_name", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()

		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}

		ctx := context.Background()
		if err := reconciler.reconcileIngress(ctx, agent, "agent.example.com"); err != nil {
			t.Fatalf("reconcileIngress failed: %v", err)
		}

		ing := &networkingv1.Ingress{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing); err != nil {
			t.Fatalf("expected Ingress to exist: %v", err)
		}
		if ing.Spec.IngressClassName != nil {
			t.Errorf("expected no IngressClassName, got %v", *ing.Spec.IngressClassName)
		}
	})

	t.Run("ingress_has_correct_host_and_path", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()

		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}

		ctx := context.Background()
		hostname := "my-agent.example.com"
		if err := reconciler.reconcileIngress(ctx, agent, hostname); err != nil {
			t.Fatalf("reconcileIngress failed: %v", err)
		}

		ing := &networkingv1.Ingress{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing); err != nil {
			t.Fatalf("expected Ingress to exist: %v", err)
		}
		if len(ing.Spec.Rules) == 0 {
			t.Fatal("expected at least one ingress rule")
		}
		if ing.Spec.Rules[0].Host != hostname {
			t.Errorf("expected host %q, got %q", hostname, ing.Spec.Rules[0].Host)
		}
		if ing.Spec.Rules[0].HTTP == nil || len(ing.Spec.Rules[0].HTTP.Paths) == 0 {
			t.Fatal("expected HTTP paths on ingress rule")
		}
		if ing.Spec.Rules[0].HTTP.Paths[0].Path != "/" {
			t.Errorf("expected path '/', got %q", ing.Spec.Rules[0].HTTP.Paths[0].Path)
		}
	})
}

func TestLanguageAgentController_IngressTLS(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
		},
	}
	hostname := "tls-agent.example.com"

	t.Run("explicit_secret_name", func(t *testing.T) {
		cluster := gen.ReadyCluster("default", gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{
			SecretName: "my-tls-secret",
		}))
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, agent).
			WithStatusSubresource(agent).
			Build()
		r := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}

		require.NoError(t, r.reconcileIngress(context.Background(), agent, hostname))

		ing := &networkingv1.Ingress{}
		require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing))
		require.Len(t, ing.Spec.TLS, 1)
		assert.Equal(t, "my-tls-secret", ing.Spec.TLS[0].SecretName)
		assert.Equal(t, []string{hostname}, ing.Spec.TLS[0].Hosts)
	})

	t.Run("cert_manager_annotation", func(t *testing.T) {
		cluster := gen.ReadyCluster("default", gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{}))
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, agent).
			WithStatusSubresource(agent).
			Build()
		r := &LanguageAgentReconciler{
			Client:               fakeClient,
			Scheme:               scheme,
			Log:                  logr.Discard(),
			Recorder:             &record.FakeRecorder{},
			RegistryManager:      &mockRegistryManager{},
			DefaultTLSIssuerName: "letsencrypt",
		}

		require.NoError(t, r.reconcileIngress(context.Background(), agent, hostname))

		ing := &networkingv1.Ingress{}
		require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing))
		assert.Equal(t, "letsencrypt", ing.Annotations["cert-manager.io/cluster-issuer"])
		require.Len(t, ing.Spec.TLS, 1)
		assert.Equal(t, agent.Name+"-tls", ing.Spec.TLS[0].SecretName)
	})

	t.Run("cluster_classname_overrides_default", func(t *testing.T) {
		cluster := gen.ReadyCluster("default", gen.SetClusterIngressClassName("traefik"))
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, agent).
			WithStatusSubresource(agent).
			Build()
		r := &LanguageAgentReconciler{
			Client:                  fakeClient,
			Scheme:                  scheme,
			Log:                     logr.Discard(),
			Recorder:                &record.FakeRecorder{},
			RegistryManager:         &mockRegistryManager{},
			DefaultIngressClassName: "nginx",
		}

		require.NoError(t, r.reconcileIngress(context.Background(), agent, hostname))

		ing := &networkingv1.Ingress{}
		require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, ing))
		require.NotNil(t, ing.Spec.IngressClassName)
		assert.Equal(t, "traefik", *ing.Spec.IngressClassName)
	})
}

func TestLanguageAgentController_CheckIngressReadiness(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("not_found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "no-ingress", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ready {
			t.Error("expected not ready when ingress not found")
		}
	})

	t.Run("no_lb_ip", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "default"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ing).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "my-ing", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ready {
			t.Error("expected not ready when no LB ingress points")
		}
	})

	t.Run("lb_with_ip", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "default"},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{
						{IP: "10.0.0.1"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ing).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "my-ing", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ready {
			t.Error("expected ready when LB has IP")
		}
	})

	t.Run("lb_with_hostname", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "default"},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{
						{Hostname: "lb.example.com"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ing).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		ready, _, err := r.checkIngressReadiness(context.Background(), "my-ing", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ready {
			t.Error("expected ready when LB has hostname")
		}
	})
}

// TestLanguageAgentController_WebhookConditions_LBNotReady verifies that when a cluster
// has spec.domain set but the Ingress LB is not yet assigned, reconcileWebhooks sets
// ConditionWebhookRouteCreated=True, ConditionWebhookRouteReady=False, and leaves
// WebhookURLs empty.
func TestLanguageAgentController_WebhookConditions_LBNotReady(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("default", gen.SetClusterDomain("example.com"))
	agent := gen.LanguageAgent("hook-agent", "default")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	require.NoError(t, reconciler.reconcileWebhooks(ctx, agent))

	var routeCreated, routeReady *metav1.Condition
	for i := range agent.Status.Conditions {
		switch agent.Status.Conditions[i].Type {
		case langopv1alpha1.ConditionWebhookRouteCreated:
			routeCreated = &agent.Status.Conditions[i]
		case langopv1alpha1.ConditionWebhookRouteReady:
			routeReady = &agent.Status.Conditions[i]
		}
	}

	require.NotNil(t, routeCreated, "ConditionWebhookRouteCreated must be set")
	assert.Equal(t, metav1.ConditionTrue, routeCreated.Status)
	assert.Equal(t, langopv1alpha1.ReasonIngressCreated, routeCreated.Reason)

	require.NotNil(t, routeReady, "ConditionWebhookRouteReady must be set")
	assert.Equal(t, metav1.ConditionFalse, routeReady.Status)

	assert.Empty(t, agent.Status.WebhookURLs, "WebhookURLs must be empty when LB is not ready")
}

// TestLanguageAgentController_WebhookConditions_LBReady verifies that when the Ingress
// LB is assigned, reconcileWebhooks sets ConditionWebhookRouteReady=True and populates
// agent.Status.WebhookURLs with the expected URL.
func TestLanguageAgentController_WebhookConditions_LBReady(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := gen.LanguageCluster("default", gen.SetClusterDomain("example.com"))
	agent := gen.LanguageAgent("hook-agent", "default")

	// Pre-create the Ingress with an LB IP already assigned so checkIngressReadiness returns true.
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hook-agent", Namespace: "default"},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{{IP: "10.0.0.1"}},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent, ing).
		WithStatusSubresource(agent, ing).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	require.NoError(t, reconciler.reconcileWebhooks(ctx, agent))

	var routeCreated, routeReady *metav1.Condition
	for i := range agent.Status.Conditions {
		switch agent.Status.Conditions[i].Type {
		case langopv1alpha1.ConditionWebhookRouteCreated:
			routeCreated = &agent.Status.Conditions[i]
		case langopv1alpha1.ConditionWebhookRouteReady:
			routeReady = &agent.Status.Conditions[i]
		}
	}

	require.NotNil(t, routeCreated, "ConditionWebhookRouteCreated must be set")
	assert.Equal(t, metav1.ConditionTrue, routeCreated.Status)
	assert.Equal(t, langopv1alpha1.ReasonIngressCreated, routeCreated.Reason)

	require.NotNil(t, routeReady, "ConditionWebhookRouteReady must be set")
	assert.Equal(t, metav1.ConditionTrue, routeReady.Status)
	assert.Equal(t, langopv1alpha1.ReasonWebhookRouteReady, routeReady.Reason)

	require.Len(t, agent.Status.WebhookURLs, 1, "WebhookURLs must contain exactly one entry")
	assert.Equal(t, "https://hook-agent.example.com", agent.Status.WebhookURLs[0])
}

func TestLanguageAgentController_ServiceTypeAndAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		serviceType     corev1.ServiceType
		annotations     map[string]string
		wantServiceType corev1.ServiceType
	}{
		{
			name:            "defaults to ClusterIP when unset",
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
		{
			name:            "NodePort applied",
			serviceType:     corev1.ServiceTypeNodePort,
			wantServiceType: corev1.ServiceTypeNodePort,
		},
		{
			name:            "annotations propagated",
			annotations:     map[string]string{"example.com/ann": "value"},
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			mods := []gen.LanguageAgentModifier{}
			if tt.serviceType != "" {
				mods = append(mods, gen.SetAgentServiceType(tt.serviceType))
			}
			if tt.annotations != nil {
				mods = append(mods, gen.SetAgentServiceAnnotations(tt.annotations))
			}
			agent := gen.LanguageAgent("test-agent", "default", mods...)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), agent).
				WithStatusSubresource(agent).
				Build()

			reconciler := &LanguageAgentReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				Log:             logr.Discard(),
				Recorder:        &record.FakeRecorder{},
				RegistryManager: &mockRegistryManager{},
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			svc := &corev1.Service{}
			if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc); err != nil {
				t.Fatalf("expected service to exist: %v", err)
			}
			if svc.Spec.Type != tt.wantServiceType {
				t.Errorf("service type = %q, want %q", svc.Spec.Type, tt.wantServiceType)
			}
			for k, v := range tt.annotations {
				if svc.Annotations[k] != v {
					t.Errorf("annotation %q = %q, want %q", k, svc.Annotations[k], v)
				}
			}
		})
	}
}

// --- Group 5: CNI detection ---

func TestLanguageAgentController_DetectNetworkPolicySupport(t *testing.T) {
	// CNI detection is no longer performed per-reconcile.
	// The startup-cached CNICapabilities field is used instead.
	// Coverage is provided by TestLanguageAgentController_ConditionNetworkPolicyEnforced_*.
}

func TestLanguageAgentController_ConditionNetworkPolicyEnforced_Supported(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("np-enforced-agent", "default")
	agent.Finalizers = []string{FinalizerName}

	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                recorder,
		EventManager:            events.NewEventManager(recorder),
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
		NetworkPolicyTimeout:    30 * time.Second,
		NetworkPolicyRetries:    3,
		CNICapabilities:         &cni.CNICapabilities{Name: "cilium", SupportsNetworkPolicy: true},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionNetworkPolicyEnforced {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "expected ConditionNetworkPolicyEnforced to be set")
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, langopv1alpha1.ReasonEnforced, cond.Reason)
	assert.Contains(t, cond.Message, "cilium")
}

func TestLanguageAgentController_ConditionNetworkPolicyEnforced_NotSupported(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// CNICapabilities nil — falls through to "unknown"/unsupported.
	agent := gen.LanguageAgent("np-not-enforced-agent", "default")
	agent.Finalizers = []string{FinalizerName}

	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                recorder,
		EventManager:            events.NewEventManager(recorder),
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
		NetworkPolicyTimeout:    30 * time.Second,
		NetworkPolicyRetries:    3,
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionNetworkPolicyEnforced {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "expected ConditionNetworkPolicyEnforced to be set")
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, langopv1alpha1.ReasonCNINotSupported, cond.Reason)
}

// TestLanguageAgentController_CustomPortService verifies that spec.ports flows through
// to both port and targetPort of the reconciled Service.
func TestLanguageAgentController_CustomPortService(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("custom-port-agent", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "http", Port: 9090, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	svc := &corev1.Service{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc))
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(9090), svc.Spec.Ports[0].Port, "Service.Spec.Ports[0].Port must match spec.ports[0].port")
	assert.Equal(t, int32(9090), svc.Spec.Ports[0].TargetPort.IntVal, "Service.Spec.Ports[0].TargetPort must match spec.ports[0].port")
}

// TestLanguageAgentController_CustomPortNetworkPolicy verifies that spec.ports flows
// through to all NetworkPolicy ingress rule ports.
func TestLanguageAgentController_CustomPortNetworkPolicy(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := gen.LanguageAgent("custom-port-np-agent", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "http", Port: 9090, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                &record.FakeRecorder{},
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
		NetworkPolicyTimeout:    30 * time.Second,
		NetworkPolicyRetries:    3,
	}

	ctx := context.Background()
	require.NoError(t, reconciler.reconcileNetworkPolicy(ctx, agent))

	np := &networkingv1.NetworkPolicy{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np))
	require.NotEmpty(t, np.Spec.Ingress, "expected ingress rules on NetworkPolicy")

	for i, rule := range np.Spec.Ingress {
		for j, p := range rule.Ports {
			assert.Equal(t, int32(9090), p.Port.IntVal,
				"NetworkPolicy ingress rule %d port %d must match spec.ports[0].port", i, j)
		}
	}
}

// TestAgentPorts_DefaultsTo8080 verifies that agentPorts returns http:8080 when spec.ports is unset.
func TestAgentPorts_DefaultsTo8080(t *testing.T) {
	agent := gen.LanguageAgent("default-port", "default")
	ports := agentPorts(agent)
	require.Len(t, ports, 1)
	assert.Equal(t, "http", ports[0].Name)
	assert.Equal(t, int32(8080), ports[0].Port)
	assert.Equal(t, ptr.To(true), ports[0].Expose)
}

// TestAgentPorts_SpecPortsUsed verifies that spec.ports is returned as-is.
func TestAgentPorts_SpecPortsUsed(t *testing.T) {
	agent := gen.LanguageAgent("multi", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "http", Port: 3000, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
			{Name: "ws", Port: 4000, Protocol: corev1.ProtocolTCP},
		}),
	)
	ports := agentPorts(agent)
	require.Len(t, ports, 2)
	assert.Equal(t, int32(3000), ports[0].Port)
	assert.Equal(t, int32(4000), ports[1].Port)
}

// TestAgentIngressPort_ExposeFlagWins verifies that agentIngressPort returns the
// port marked expose:true even when it is not the first in the list.
func TestAgentIngressPort_ExposeFlagWins(t *testing.T) {
	agent := gen.LanguageAgent("expose-test", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "ws", Port: 4000, Protocol: corev1.ProtocolTCP},
			{Name: "http", Port: 3000, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		}),
	)
	assert.Equal(t, int32(3000), agentIngressPort(agent))
}

// TestAgentIngressPort_FallbackToFirst verifies that agentIngressPort falls back
// to the first port when no port has expose:true.
func TestAgentIngressPort_FallbackToFirst(t *testing.T) {
	agent := gen.LanguageAgent("fallback-test", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "ws", Port: 4000, Protocol: corev1.ProtocolTCP},
			{Name: "http", Port: 3000, Protocol: corev1.ProtocolTCP},
		}),
	)
	assert.Equal(t, int32(4000), agentIngressPort(agent))
}

// TestLanguageAgentController_MultiPortService verifies that reconcileService
// creates one ServicePort per AgentPort.
func TestLanguageAgentController_MultiPortService(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("multi-port-svc", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "http", Port: 3000, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
			{Name: "ws", Port: 4000, Protocol: corev1.ProtocolTCP},
		}),
	)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	require.NoError(t, reconciler.reconcileService(ctx, agent))

	svc := &corev1.Service{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc))
	require.Len(t, svc.Spec.Ports, 2, "expected one ServicePort per AgentPort")
	assert.Equal(t, "http", svc.Spec.Ports[0].Name)
	assert.Equal(t, int32(3000), svc.Spec.Ports[0].Port)
	assert.Equal(t, "ws", svc.Spec.Ports[1].Name)
	assert.Equal(t, int32(4000), svc.Spec.Ports[1].Port)
}

// TestLanguageAgentController_MultiPortNetworkPolicy verifies that all three
// built-in ingress rules carry one NetworkPolicyPort per AgentPort.
func TestLanguageAgentController_MultiPortNetworkPolicy(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("multi-port-np", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "http", Port: 3000, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
			{Name: "ws", Port: 4000, Protocol: corev1.ProtocolTCP},
		}),
	)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		Recorder:                &record.FakeRecorder{},
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
		NetworkPolicyTimeout:    30 * time.Second,
		NetworkPolicyRetries:    3,
	}
	ctx := context.Background()
	require.NoError(t, reconciler.reconcileNetworkPolicy(ctx, agent))

	np := &networkingv1.NetworkPolicy{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np))
	// First 2 rules are the built-in trigger/agent-to-agent rules.
	require.GreaterOrEqual(t, len(np.Spec.Ingress), 2)
	for i := 0; i < 2; i++ {
		rule := np.Spec.Ingress[i]
		require.Len(t, rule.Ports, 2, "rule %d should have 2 ports", i)
		assert.Equal(t, int32(3000), rule.Ports[0].Port.IntVal, "rule %d port 0", i)
		assert.Equal(t, int32(4000), rule.Ports[1].Port.IntVal, "rule %d port 1", i)
	}
}

// TestLanguageAgentController_IngressControllerNamespace verifies that when
// IngressControllerNamespace is set, a third ingress rule is added to allow
// the ingress controller namespace to reach agent ports.
func TestLanguageAgentController_IngressControllerNamespace(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("ingress-ns-agent", "default",
		gen.SetAgentPorts([]langopv1alpha1.AgentPort{
			{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		}),
	)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:                     fakeClient,
		Scheme:                     scheme,
		Log:                        logr.Discard(),
		Recorder:                   &record.FakeRecorder{},
		RegistryManager:            &mockRegistryManager{},
		NetworkIsolationEnabled:    true,
		NetworkPolicyTimeout:       30 * time.Second,
		NetworkPolicyRetries:       3,
		IngressControllerNamespace: "traefik",
	}
	ctx := context.Background()
	require.NoError(t, reconciler.reconcileNetworkPolicy(ctx, agent))

	np := &networkingv1.NetworkPolicy{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np))
	// 2 built-in rules + 1 ingress-controller rule
	require.Len(t, np.Spec.Ingress, 3, "expected 3 ingress rules (2 built-in + ingress controller)")
	ingressNsRule := np.Spec.Ingress[2]
	require.Len(t, ingressNsRule.From, 1)
	require.NotNil(t, ingressNsRule.From[0].NamespaceSelector)
	assert.Equal(t, "traefik", ingressNsRule.From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
	require.Len(t, ingressNsRule.Ports, 1)
	assert.Equal(t, int32(8080), ingressNsRule.Ports[0].Port.IntVal)
}

// TestLanguageAgentController_NoIngressControllerNamespace verifies that without
// IngressControllerNamespace set, only the 2 built-in ingress rules are created.
func TestLanguageAgentController_NoIngressControllerNamespace(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("no-ingress-ns-agent", "default")
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:               fakeClient,
		Scheme:               scheme,
		Log:                  logr.Discard(),
		Recorder:             &record.FakeRecorder{},
		RegistryManager:      &mockRegistryManager{},
		NetworkPolicyTimeout: 30 * time.Second,
		NetworkPolicyRetries: 3,
		// IngressControllerNamespace intentionally not set
	}
	ctx := context.Background()
	require.NoError(t, reconciler.reconcileNetworkPolicy(ctx, agent))

	np := &networkingv1.NetworkPolicy{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, np))
	assert.Len(t, np.Spec.Ingress, 2, "expected exactly 2 built-in ingress rules")
}
