package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	unstructuredPkg "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// monitoringRESTMapper returns a REST mapper that knows about monitoring.coreos.com CRDs
// in addition to all types in the given scheme.
func monitoringRESTMapper(scheme *runtime.Scheme) apimeta.RESTMapper {
	base := testrestmapper.TestOnlyStaticRESTMapper(scheme)
	extra := apimeta.NewDefaultRESTMapper(nil)
	monGV := schema.GroupVersion{Group: "monitoring.coreos.com", Version: "v1"}
	extra.Add(monGV.WithKind("ServiceMonitor"), apimeta.RESTScopeNamespace)
	extra.Add(monGV.WithKind("PrometheusRule"), apimeta.RESTScopeNamespace)
	return apimeta.MultiRESTMapper{base, extra}
}

func TestLanguageAgentController_ServiceMonitor_CreatedWhenEnabled(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sm-agent", "default")
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		ServiceMonitor: &langopv1alpha1.AgentServiceMonitorSpec{
			Enabled:       true,
			Path:          "/metrics",
			Interval:      "30s",
			ScrapeTimeout: "10s",
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	sm := &unstructuredPkg.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, sm))

	endpoints, ok, err := unstructuredPkg.NestedSlice(sm.Object, "spec", "endpoints")
	require.NoError(t, err)
	require.True(t, ok, "spec.endpoints must be set")
	require.Len(t, endpoints, 1)

	ep := endpoints[0].(map[string]any)
	assert.Equal(t, "/metrics", ep["path"])
	assert.Equal(t, "30s", ep["interval"])
	assert.Equal(t, "10s", ep["scrapeTimeout"])

	selector, ok, err := unstructuredPkg.NestedStringMap(sm.Object, "spec", "selector", "matchLabels")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, agent.Name, selector[LabelKeyK8sName])
}

func TestLanguageAgentController_ServiceMonitor_PortDefaultsToFirstPort(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sm-port-agent", "default")
	agent.Spec.Ports = []langopv1alpha1.AgentPort{{Name: "grpc", Port: 9090}}
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		ServiceMonitor: &langopv1alpha1.AgentServiceMonitorSpec{Enabled: true},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	sm := &unstructuredPkg.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, sm))

	endpoints, _, _ := unstructuredPkg.NestedSlice(sm.Object, "spec", "endpoints")
	require.Len(t, endpoints, 1)
	ep := endpoints[0].(map[string]any)
	assert.Equal(t, "grpc", ep["port"], "port should default to first spec.ports entry")
}

func TestLanguageAgentController_ServiceMonitor_PortDefaultsToHTTPWhenNoPortsDefined(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sm-noport-agent", "default")
	// no spec.ports defined
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		ServiceMonitor: &langopv1alpha1.AgentServiceMonitorSpec{Enabled: true},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	sm := &unstructuredPkg.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, sm))

	endpoints, _, _ := unstructuredPkg.NestedSlice(sm.Object, "spec", "endpoints")
	require.Len(t, endpoints, 1)
	ep := endpoints[0].(map[string]any)
	assert.Equal(t, "http", ep["port"], "port should default to 'http' when no ports are defined")
}

func TestLanguageAgentController_ServiceMonitor_NotCreatedWhenDisabled(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sm-disabled-agent", "default")
	// monitoring nil — should not create ServiceMonitor
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	sm := &unstructuredPkg.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, sm)
	assert.True(t, errors.IsNotFound(err), "ServiceMonitor must not exist when monitoring is nil")
}

func TestLanguageAgentController_ServiceMonitor_DeletedWhenDisabled(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sm-rm-agent", "default")
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		ServiceMonitor: &langopv1alpha1.AgentServiceMonitorSpec{Enabled: true},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify ServiceMonitor exists
	sm := &unstructuredPkg.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, sm), "ServiceMonitor must exist after enabled reconcile")

	// Disable monitoring
	current := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, current))
	current.Spec.Monitoring.ServiceMonitor.Enabled = false
	require.NoError(t, fakeClient.Update(ctx, current))

	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, sm)
	assert.True(t, errors.IsNotFound(err), "ServiceMonitor must be deleted when monitoring is disabled")
}

func TestLanguageAgentController_ServiceMonitor_GracefulWhenCRDAbsent(t *testing.T) {
	// No monitoring CRDs in REST mapper → controller must not fail.
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("sm-nocrd-agent", "default")
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		ServiceMonitor: &langopv1alpha1.AgentServiceMonitorSpec{Enabled: true},
	}
	// Use default scheme-based mapper only (no monitoring CRDs)
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	assert.NoError(t, err, "reconcile must succeed even when prometheus-operator CRDs are absent")
}

func TestLanguageAgentController_PrometheusRule_CreatedWhenRulesDefined(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("pr-agent", "default")
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		Rules: []langopv1alpha1.PrometheusRuleGroup{
			{
				Name:     "pr-agent.alerts",
				Interval: "1m",
				Rules: []langopv1alpha1.PrometheusAlertingRule{
					{
						Alert:       "AgentDown",
						Expr:        `up{job="pr-agent"} == 0`,
						For:         "5m",
						Labels:      map[string]string{"severity": "critical"},
						Annotations: map[string]string{"summary": "Agent is down"},
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pr := &unstructuredPkg.Unstructured{}
	pr.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusRule"})
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pr))

	groups, ok, err := unstructuredPkg.NestedSlice(pr.Object, "spec", "groups")
	require.NoError(t, err)
	require.True(t, ok, "spec.groups must be set")
	require.Len(t, groups, 1)

	group := groups[0].(map[string]any)
	assert.Equal(t, "pr-agent.alerts", group["name"])
	assert.Equal(t, "1m", group["interval"])

	rules := group["rules"].([]any)
	require.Len(t, rules, 1)
	rule := rules[0].(map[string]any)
	assert.Equal(t, "AgentDown", rule["alert"])
	assert.Equal(t, `up{job="pr-agent"} == 0`, rule["expr"])
	assert.Equal(t, "5m", rule["for"])
	assert.Equal(t, "critical", rule["labels"].(map[string]any)["severity"])
	assert.Equal(t, "Agent is down", rule["annotations"].(map[string]any)["summary"])
}

func TestLanguageAgentController_PrometheusRule_DeletedWhenRulesRemoved(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("pr-rm-agent", "default")
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		Rules: []langopv1alpha1.PrometheusRuleGroup{
			{
				Name:  "pr-rm-agent.alerts",
				Rules: []langopv1alpha1.PrometheusAlertingRule{{Alert: "Test", Expr: "up == 0"}},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(monitoringRESTMapper(scheme)).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	pr := &unstructuredPkg.Unstructured{}
	pr.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusRule"})
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pr), "PrometheusRule must exist initially")

	// Remove rules
	current := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, current))
	current.Spec.Monitoring.Rules = nil
	require.NoError(t, fakeClient.Update(ctx, current))

	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, pr)
	assert.True(t, errors.IsNotFound(err), "PrometheusRule must be deleted when rules are removed")
}

func TestLanguageAgentController_PrometheusRule_GracefulWhenCRDAbsent(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("pr-nocrd-agent", "default")
	agent.Spec.Monitoring = &langopv1alpha1.AgentMonitoringSpec{
		Rules: []langopv1alpha1.PrometheusRuleGroup{
			{
				Name:  "alerts",
				Rules: []langopv1alpha1.PrometheusAlertingRule{{Alert: "Test", Expr: "up == 0"}},
			},
		},
	}
	// Default mapper only — no monitoring CRDs
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
		EventManager:    events.NewEventManager(&record.FakeRecorder{}),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	assert.NoError(t, err, "reconcile must succeed even when prometheus-operator CRDs are absent")
}
