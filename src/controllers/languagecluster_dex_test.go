package controllers

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// authCluster returns a LanguageCluster with auth enabled and a domain set.
func authCluster(name string, mods ...gen.LanguageClusterModifier) *langopv1alpha1.LanguageCluster {
	c := gen.LanguageCluster(name,
		gen.SetClusterDomain("example.com"),
	)
	c.Status.Phase = "Ready"
	c.Spec.Auth = &langopv1alpha1.ClusterAuthSpec{
		Enabled: true,
		OIDC: &langopv1alpha1.ClusterOIDCSpec{
			Dex: &langopv1alpha1.DexSpec{
				EnablePasswordDB: true,
				StaticPasswords: []langopv1alpha1.DexStaticPassword{
					{Email: "user@example.com", Hash: "$2a$10$abc", Username: "user", UserID: "uid-1"},
				},
			},
		},
	}
	for _, mod := range mods {
		mod(c)
	}
	return c
}

// --- agentAuthEnabled ---

func TestAgentAuthEnabled_InheritsFromCluster(t *testing.T) {
	cluster := authCluster("test")
	agent := gen.LanguageAgent("a", "test")
	assert.True(t, agentAuthEnabled(agent, cluster))
}

func TestAgentAuthEnabled_ClusterDisabled(t *testing.T) {
	cluster := gen.ReadyCluster("test")
	agent := gen.LanguageAgent("a", "test")
	assert.False(t, agentAuthEnabled(agent, cluster))
}

func TestAgentAuthEnabled_AgentOverrideTrue(t *testing.T) {
	// cluster has auth disabled but agent forces it on
	cluster := gen.ReadyCluster("test")
	agent := gen.LanguageAgent("a", "test")
	agent.Spec.Auth = &langopv1alpha1.AgentAuthSpec{Enabled: ptr.To(true)}
	assert.True(t, agentAuthEnabled(agent, cluster))
}

func TestAgentAuthEnabled_AgentOverrideFalse(t *testing.T) {
	// cluster has auth enabled but agent opts out
	cluster := authCluster("test")
	agent := gen.LanguageAgent("a", "test")
	agent.Spec.Auth = &langopv1alpha1.AgentAuthSpec{Enabled: ptr.To(false)}
	assert.False(t, agentAuthEnabled(agent, cluster))
}

// --- dexIssuerURL ---

func TestDexIssuerURL_NilAuth(t *testing.T) {
	cluster := gen.ReadyCluster("test", gen.SetClusterDomain("example.com"))
	assert.Equal(t, "", dexIssuerURL(cluster))
}

func TestDexIssuerURL_Embedded_HTTPS(t *testing.T) {
	cluster := authCluster("test")
	assert.Equal(t, "https://auth.example.com", dexIssuerURL(cluster))
}

func TestDexIssuerURL_Embedded_HTTP_WhenTLSDisabled(t *testing.T) {
	cluster := authCluster("test", gen.SetClusterIngressTLS(&langopv1alpha1.IngressTLSConfig{Enabled: ptr.To(false)}))
	assert.Equal(t, "http://auth.example.com", dexIssuerURL(cluster))
}

func TestDexIssuerURL_External(t *testing.T) {
	cluster := authCluster("test")
	cluster.Spec.Auth.OIDC.ExternalIssuerURL = "https://idp.corp.com"
	assert.Equal(t, "https://idp.corp.com", dexIssuerURL(cluster))
}

// --- buildDexConfigYAML ---

func newDexReconciler(t *testing.T) *LanguageClusterReconciler {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	return &LanguageClusterReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
		Log:    logr.Discard(),
	}
}

func TestBuildDexConfigYAML_Issuer(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("test")
	yaml, err := r.buildDexConfigYAML(cluster, "secret", nil)
	require.NoError(t, err)
	assert.Contains(t, yaml, "issuer: https://auth.example.com")
}

func TestBuildDexConfigYAML_RedirectURIs(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("test")
	uris := []string{
		"https://agent1.example.com/oauth2/callback",
		"https://agent2.example.com/oauth2/callback",
	}
	yaml, err := r.buildDexConfigYAML(cluster, "secret", uris)
	require.NoError(t, err)
	assert.Contains(t, yaml, "https://agent1.example.com/oauth2/callback")
	assert.Contains(t, yaml, "https://agent2.example.com/oauth2/callback")
}

func TestBuildDexConfigYAML_EnablePasswordDB(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("test")
	yaml, err := r.buildDexConfigYAML(cluster, "secret", nil)
	require.NoError(t, err)
	assert.Contains(t, yaml, "enablePasswordDB: true")
	assert.Contains(t, yaml, "user@example.com")
}

func TestBuildDexConfigYAML_NoPasswordDB(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("test")
	cluster.Spec.Auth.OIDC.Dex.EnablePasswordDB = false
	cluster.Spec.Auth.OIDC.Dex.StaticPasswords = nil
	yaml, err := r.buildDexConfigYAML(cluster, "secret", nil)
	require.NoError(t, err)
	assert.NotContains(t, yaml, "enablePasswordDB")
	assert.NotContains(t, yaml, "staticPasswords")
}

func TestBuildDexConfigYAML_ClientSecret(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("test")
	yaml, err := r.buildDexConfigYAML(cluster, "my-secret-value", nil)
	require.NoError(t, err)
	assert.Contains(t, yaml, "my-secret-value")
}

func TestBuildDexConfigYAML_SkipApprovalScreen(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("test")
	yaml, err := r.buildDexConfigYAML(cluster, "s", nil)
	require.NoError(t, err)
	assert.Contains(t, yaml, "skipApprovalScreen: true")
}

// --- reconcileDex: ConfigMap, Secret, Deployment, Service, Ingress ---

func TestReconcileDex_CreatesSecret(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	secret := &corev1.Secret{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexClientSecretName, Namespace: cluster.Name}, secret))
	// Fake client stores StringData as-is (no base64 encoding like the real API server).
	val := secret.StringData[DexClientSecretKey]
	if val == "" {
		val = string(secret.Data[DexClientSecretKey])
	}
	assert.NotEmpty(t, val, "client secret must be non-empty")
}

func TestReconcileDex_SecretStable(t *testing.T) {
	// Calling reconcileDex twice must not change the secret value.
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()

	require.NoError(t, r.reconcileDex(ctx, cluster))

	secret1 := &corev1.Secret{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: DexClientSecretName, Namespace: cluster.Name}, secret1))
	first := string(secret1.Data[DexClientSecretKey])

	require.NoError(t, r.reconcileDex(ctx, cluster))

	secret2 := &corev1.Secret{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: DexClientSecretName, Namespace: cluster.Name}, secret2))
	assert.Equal(t, first, string(secret2.Data[DexClientSecretKey]), "secret must not be rotated")
}

func TestReconcileDex_CreatesConfigMap(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexConfigMapName, Namespace: cluster.Name}, cm))
	cfgYAML := cm.Data["config.yaml"]
	assert.Contains(t, cfgYAML, "issuer: https://auth.example.com")
	assert.Contains(t, cfgYAML, "enablePasswordDB: true")
	assert.Contains(t, cfgYAML, "user@example.com")
}

func TestReconcileDex_ConfigMapContainsAgentRedirectURIs(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	agent := gen.LanguageAgent("my-agent", "mycluster")
	// auth is inherited from cluster (enabled)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(cluster).
		Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexConfigMapName, Namespace: cluster.Name}, cm))
	assert.Contains(t, cm.Data["config.yaml"], "https://my-agent.example.com/oauth2/callback")
}

func TestReconcileDex_OptedOutAgentNotInRedirectURIs(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	agent := gen.LanguageAgent("opted-out", "mycluster")
	agent.Spec.Auth = &langopv1alpha1.AgentAuthSpec{Enabled: ptr.To(false)}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(cluster).
		Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexConfigMapName, Namespace: cluster.Name}, cm))
	assert.NotContains(t, cm.Data["config.yaml"], "opted-out")
}

func TestReconcileDex_CreatesDeployment(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, dep))
	assert.NotEmpty(t, dep.Spec.Template.Annotations["langop.io/dex-config-hash"], "pod template must carry config hash")
	containers := dep.Spec.Template.Spec.Containers
	require.Len(t, containers, 1)
	assert.Contains(t, containers[0].Image, "dex")
}

func TestReconcileDex_ConfigHashChangesOnURIUpdate(t *testing.T) {
	// Adding an agent should produce a different config hash, causing Dex to roll out.
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()

	require.NoError(t, r.reconcileDex(ctx, cluster))
	dep1 := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, dep1))
	hash1 := dep1.Spec.Template.Annotations["langop.io/dex-config-hash"]

	// Add an agent that will be included in redirect URIs.
	agent := gen.LanguageAgent("new-agent", "mycluster")
	require.NoError(t, fakeClient.Create(ctx, agent))

	require.NoError(t, r.reconcileDex(ctx, cluster))
	dep2 := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, dep2))
	hash2 := dep2.Spec.Template.Annotations["langop.io/dex-config-hash"]

	assert.NotEqual(t, hash1, hash2, "config hash must change when redirect URIs change")
}

func TestReconcileDex_CreatesService(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	svc := &corev1.Service{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, svc))
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, DexPort, svc.Spec.Ports[0].Port)
}

func TestReconcileDex_CreatesIngress(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	ing := &networkingv1.Ingress{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, ing))
	require.Len(t, ing.Spec.Rules, 1)
	assert.Equal(t, "auth.example.com", ing.Spec.Rules[0].Host)
}

func TestReconcileDex_SkipsIngressWhenNoDomain(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	cluster.Spec.Domain = ""
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	ing := &networkingv1.Ingress{}
	err := fakeClient.Get(context.Background(), types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, ing)
	assert.True(t, strings.Contains(err.Error(), "not found"), "ingress must not be created when domain is empty")
}

func TestReconcileDex_ImageFallsBackToDefault(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	// DexImage not set on reconciler — must fall back to hardcoded default
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, dep))
	assert.NotEmpty(t, dep.Spec.Template.Spec.Containers[0].Image)
}

// --- branded Dex templates ---

func TestBuildDexConfigYAML_FrontendPointsAtMountedTemplates(t *testing.T) {
	r := newDexReconciler(t)
	cluster := authCluster("mycluster")
	yaml, err := r.buildDexConfigYAML(cluster, "secret", nil)
	require.NoError(t, err)
	assert.Contains(t, yaml, "frontend:")
	assert.Contains(t, yaml, "dir: "+DexTemplatesMountPath)
	assert.Contains(t, yaml, "issuer: Language Operator")
	// Cluster name is surfaced to templates via frontend.extra so the login
	// header can render an environment chip ({{ extra "clusterName" }}).
	assert.Contains(t, yaml, "clusterName: mycluster")
}

func TestDexWebAssets_RequiredFilesAreEmbedded(t *testing.T) {
	// The required set mirrors what Dex's loader insists on at startup (see
	// dex/server/templates.go requiredTmpls + the static/themes/robots.txt
	// directory contract). If any of these go missing the Dex pod won't start.
	required := []string{
		"templates/header.html",
		"templates/footer.html",
		"templates/login.html",
		"templates/password.html",
		"templates/approval.html",
		"templates/oob.html",
		"templates/error.html",
		"templates/device.html",
		"templates/device_success.html",
		"templates/totp_verify.html",
		"templates/webauthn_verify.html",
		"templates/home.html",
		"templates/logout.html",
		"static/main.css",
		"static/img/logo.svg",
		"static/img/favicon.svg",
		"themes/light/styles.css",
		"robots.txt",
	}
	for _, rel := range required {
		key := strings.ReplaceAll(rel, "/", dexWebPathSeparator)
		assert.NotEmpty(t, dexWebData[key], "embedded asset %q must be present and non-empty", rel)
	}
	assert.NotEmpty(t, dexWebHash, "asset hash must be computed at init")
}

func TestReconcileDex_CreatesTemplatesConfigMap(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexTemplatesConfigMapName, Namespace: cluster.Name}, cm))

	loginKey := strings.ReplaceAll("templates/login.html", "/", dexWebPathSeparator)
	cssKey := strings.ReplaceAll("static/main.css", "/", dexWebPathSeparator)
	assert.Contains(t, cm.Data[loginKey], "Sign in to")
	assert.Contains(t, cm.Data[cssKey], "Marfa")
}

func TestReconcileDex_DeploymentMountsTemplatesVolume(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

	require.NoError(t, r.reconcileDex(context.Background(), cluster))

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: DexResourceName, Namespace: cluster.Name}, dep))

	assert.NotEmpty(t, dep.Spec.Template.Annotations["langop.io/dex-templates-hash"], "pod template must carry templates hash so assets roll out")

	require.Len(t, dep.Spec.Template.Spec.Containers, 1)
	mounts := dep.Spec.Template.Spec.Containers[0].VolumeMounts
	var sawTemplatesMount bool
	for _, m := range mounts {
		if m.Name == "templates" && m.MountPath == DexTemplatesMountPath && m.ReadOnly {
			sawTemplatesMount = true
		}
	}
	assert.True(t, sawTemplatesMount, "dex container must mount the templates ConfigMap read-only at %s", DexTemplatesMountPath)

	var templatesVol *corev1.Volume
	for i := range dep.Spec.Template.Spec.Volumes {
		v := &dep.Spec.Template.Spec.Volumes[i]
		if v.Name == "templates" {
			templatesVol = v
		}
	}
	require.NotNil(t, templatesVol, "templates volume must be defined on the pod")
	require.NotNil(t, templatesVol.ConfigMap, "templates volume must be backed by a ConfigMap")
	assert.Equal(t, DexTemplatesConfigMapName, templatesVol.ConfigMap.Name)
	// Items recreate the templates/static/themes subdirectory tree under the mount.
	assert.NotEmpty(t, templatesVol.ConfigMap.Items, "items must remap ConfigMap keys back to relative paths")
}

func TestCleanupDexResources_DeletesTemplatesConfigMap(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cluster := authCluster("mycluster")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()

	require.NoError(t, r.reconcileDex(ctx, cluster))

	// Sanity check: templates CM exists before cleanup.
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: DexTemplatesConfigMapName, Namespace: cluster.Name}, &corev1.ConfigMap{}))

	require.NoError(t, r.cleanupDexResources(ctx, cluster.Name))

	err := fakeClient.Get(ctx, types.NamespacedName{Name: DexTemplatesConfigMapName, Namespace: cluster.Name}, &corev1.ConfigMap{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
