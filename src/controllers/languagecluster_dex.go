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

package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
)

// dexConfig is the Dex YAML configuration structure.
type dexConfig struct {
	Issuer           string              `yaml:"issuer"`
	Storage          dexStorage          `yaml:"storage"`
	Web              dexWeb              `yaml:"web"`
	Connectors       []dexConnector      `yaml:"connectors,omitempty"`
	EnablePasswordDB bool                `yaml:"enablePasswordDB,omitempty"`
	StaticPasswords  []dexStaticPassword `yaml:"staticPasswords,omitempty"`
	StaticClients    []dexStaticClient   `yaml:"staticClients"`
	OAuth2           dexOAuth2           `yaml:"oauth2"`
}

type dexStorage struct {
	Type string `yaml:"type"`
}

type dexWeb struct {
	HTTP string `yaml:"http"`
}

type dexConnector struct {
	Type   string            `yaml:"type"`
	ID     string            `yaml:"id"`
	Name   string            `yaml:"name"`
	Config map[string]string `yaml:"config,omitempty"`
}

type dexStaticPassword struct {
	Email    string `yaml:"email"`
	Hash     string `yaml:"hash"`
	Username string `yaml:"username,omitempty"`
	UserID   string `yaml:"userID,omitempty"`
}

type dexStaticClient struct {
	ID           string   `yaml:"id"`
	Secret       string   `yaml:"secret"`
	RedirectURIs []string `yaml:"redirectURIs,omitempty"`
	Name         string   `yaml:"name"`
}

type dexOAuth2 struct {
	SkipApprovalScreen bool `yaml:"skipApprovalScreen"`
}

// reconcileDex creates or updates the Dex OIDC provider resources for a cluster.
// When auth is disabled or an external OIDC issuer is configured, existing Dex
// resources are deleted.
func (r *LanguageClusterReconciler) reconcileDex(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	auth := cluster.Spec.Auth
	namespace := cluster.Name

	// No embedded Dex when auth is off or an external issuer is supplied.
	if auth == nil || !auth.Enabled || (auth.OIDC != nil && auth.OIDC.ExternalIssuerURL != "") {
		return r.cleanupDexResources(ctx, namespace)
	}

	// 1. Ensure the shared client secret exists (auto-generate once, never rotate).
	clientSecret, err := r.reconcileDexClientSecret(ctx, cluster, namespace)
	if err != nil {
		return err
	}

	// 2. Collect redirect URIs from all auth-enabled LanguageAgents in this namespace.
	agentList := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list agents for dex redirect URIs: %w", err)
	}
	var redirectURIs []string
	for i := range agentList.Items {
		agent := &agentList.Items[i]
		if agentAuthEnabled(agent, cluster) {
			redirectURIs = append(redirectURIs, fmt.Sprintf("https://%s.%s/oauth2/callback", agent.Name, cluster.Spec.Domain))
		}
	}

	// 3. Build and reconcile the Dex ConfigMap.
	cfgYAML, err := r.buildDexConfigYAML(cluster, clientSecret, redirectURIs)
	if err != nil {
		return fmt.Errorf("failed to build dex config: %w", err)
	}
	if err := CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, cluster, DexConfigMapName, namespace, map[string]string{"config.yaml": cfgYAML}); err != nil {
		return fmt.Errorf("failed to reconcile dex ConfigMap: %w", err)
	}

	// 4. Reconcile Dex Deployment (pass config hash so a pod rollout fires when URIs change).
	cfgHash := fmt.Sprintf("%x", sha256.Sum256([]byte(cfgYAML)))
	if err := r.reconcileDexDeployment(ctx, cluster, namespace, auth, cfgHash); err != nil {
		return err
	}

	// 5. Reconcile Dex Service.
	if err := r.reconcileDexService(ctx, cluster, namespace); err != nil {
		return err
	}

	// 6. Reconcile Dex Ingress at auth.<domain>.
	if cluster.Spec.Domain != "" {
		if err := r.reconcileDexIngress(ctx, cluster, namespace); err != nil {
			return err
		}
	}

	log.Info("Reconciled Dex OIDC provider")
	return nil
}

// reconcileDexClientSecret creates the shared oauth2-proxy client secret if it doesn't exist.
// The secret is stable — it is never rotated automatically.
func (r *LanguageClusterReconciler) reconcileDexClientSecret(ctx context.Context, cluster *langopv1alpha1.LanguageCluster, namespace string) (string, error) {
	key := types.NamespacedName{Name: DexClientSecretName, Namespace: namespace}
	existing := &corev1.Secret{}
	err := r.Get(ctx, key, existing)
	if err == nil {
		return string(existing.Data[DexClientSecretKey]), nil
	}
	if !apierrors.IsNotFound(err) {
		return "", fmt.Errorf("failed to get dex client secret: %w", err)
	}

	// Generate a random 32-byte secret.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate dex client secret: %w", err)
	}
	secret := base64.URLEncoding.EncodeToString(raw)

	obj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DexClientSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{DexClientSecretKey: secret},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, obj, func() error { return nil }); err != nil {
		return "", fmt.Errorf("failed to create dex client secret: %w", err)
	}
	return secret, nil
}

// buildDexConfigYAML returns the Dex configuration YAML for the given cluster.
func (r *LanguageClusterReconciler) buildDexConfigYAML(cluster *langopv1alpha1.LanguageCluster, clientSecret string, redirectURIs []string) (string, error) {
	scheme := "https"
	if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil &&
		cluster.Spec.Ingress.TLS.Enabled != nil && !*cluster.Spec.Ingress.TLS.Enabled {
		scheme = "http"
	}
	issuer := fmt.Sprintf("%s://auth.%s", scheme, cluster.Spec.Domain)

	// The client secret in the Dex config is the literal value — the ConfigMap is
	// namespace-scoped so access is already gated by RBAC.
	cfg := dexConfig{
		Issuer:  issuer,
		Storage: dexStorage{Type: "memory"},
		Web:     dexWeb{HTTP: fmt.Sprintf("0.0.0.0:%d", DexPort)},
		OAuth2:  dexOAuth2{SkipApprovalScreen: true},
		StaticClients: []dexStaticClient{
			{
				ID:           "langop-agents",
				Secret:       clientSecret,
				RedirectURIs: redirectURIs,
				Name:         "Language Operator",
			},
		},
	}

	if cluster.Spec.Auth.OIDC != nil && cluster.Spec.Auth.OIDC.Dex != nil {
		dex := cluster.Spec.Auth.OIDC.Dex
		for _, c := range dex.Connectors {
			cfg.Connectors = append(cfg.Connectors, dexConnector{
				Type:   c.Type,
				ID:     c.ID,
				Name:   c.Name,
				Config: c.Config,
			})
		}
		cfg.EnablePasswordDB = dex.EnablePasswordDB
		for _, sp := range dex.StaticPasswords {
			cfg.StaticPasswords = append(cfg.StaticPasswords, dexStaticPassword{
				Email:    sp.Email,
				Hash:     sp.Hash,
				Username: sp.Username,
				UserID:   sp.UserID,
			})
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal dex config: %w", err)
	}
	return string(data), nil
}

// reconcileDexDeployment creates or updates the Dex Deployment.
func (r *LanguageClusterReconciler) reconcileDexDeployment(ctx context.Context, cluster *langopv1alpha1.LanguageCluster, namespace string, auth *langopv1alpha1.ClusterAuthSpec, cfgHash string) error {
	image := r.DexImage
	if auth.OIDC != nil && auth.OIDC.Dex != nil && auth.OIDC.Dex.Image != "" {
		image = auth.OIDC.Dex.Image
	}
	if image == "" {
		image = "ghcr.io/dexidp/dex:latest"
	}

	labels := r.dexLabels(cluster)
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: DexResourceName, Namespace: namespace},
	}
	return CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, deployment, func() error {
		deployment.Labels = labels
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: map[string]string{"langop.io/dex-config-hash": cfgHash},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  DexResourceName,
							Image: image,
							Args: []string{
								"dex",
								"serve",
								"/etc/dex/config.yaml",
							},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: DexPort, Protocol: corev1.ProtocolTCP},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config", MountPath: "/etc/dex", ReadOnly: true},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt32(DexPort),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt32(DexPort),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       30,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: DexConfigMapName},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						RunAsUser:    ptr.To(int64(1001)),
					},
				},
			},
		}
		return nil
	})
}

// reconcileDexService creates or updates the Dex ClusterIP Service.
func (r *LanguageClusterReconciler) reconcileDexService(ctx context.Context, cluster *langopv1alpha1.LanguageCluster, namespace string) error {
	labels := r.dexLabels(cluster)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: DexResourceName, Namespace: namespace},
	}
	return CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, svc, func() error {
		svc.Labels = labels
		svc.Spec = corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       DexPort,
					TargetPort: intstr.FromInt32(DexPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}
		return nil
	})
}

// reconcileDexIngress creates or updates the Ingress exposing Dex at auth.<domain>.
func (r *LanguageClusterReconciler) reconcileDexIngress(ctx context.Context, cluster *langopv1alpha1.LanguageCluster, namespace string) error {
	hostname := "auth." + cluster.Spec.Domain

	ingressClass := r.DefaultIngressClassName
	if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.ClassName != "" {
		ingressClass = cluster.Spec.Ingress.ClassName
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: DexResourceName, Namespace: namespace},
	}
	return CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, ingress, func() error {
		pathType := networkingv1.PathTypePrefix
		ingress.Labels = r.dexLabels(cluster)
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: hostname,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: DexResourceName,
											Port: networkingv1.ServiceBackendPort{Number: DexPort},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		tlsEnabled := r.DefaultTLSIssuerName != ""
		if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil {
			if cluster.Spec.Ingress.TLS.Enabled != nil {
				tlsEnabled = *cluster.Spec.Ingress.TLS.Enabled
			} else {
				tlsEnabled = true
			}
		}
		if tlsEnabled {
			if r.DefaultTLSIssuerName != "" {
				if ingress.Annotations == nil {
					ingress.Annotations = make(map[string]string)
				}
				kind := r.DefaultTLSIssuerKind
				if kind == "" {
					kind = "ClusterIssuer"
				}
				ingress.Annotations["cert-manager.io/"+certManagerIssuerAnnotationSuffix(kind)] = r.DefaultTLSIssuerName
			}
			ingress.Spec.TLS = []networkingv1.IngressTLS{
				{Hosts: []string{hostname}, SecretName: "auth-tls"},
			}
		}
		if ingressClass != "" {
			ingress.Spec.IngressClassName = &ingressClass
		}
		return nil
	})
}

// cleanupDexResources deletes Dex resources when auth is disabled or switched to external OIDC.
func (r *LanguageClusterReconciler) cleanupDexResources(ctx context.Context, namespace string) error {
	log := log.FromContext(ctx)

	toDelete := []client.Object{
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: DexResourceName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: DexResourceName, Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: DexResourceName, Namespace: namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: DexConfigMapName, Namespace: namespace}},
	}
	for _, obj := range toDelete {
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete dex resource %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
		}
	}

	log.V(1).Info("Cleaned up Dex OIDC provider resources")
	return nil
}

// dexLabels returns the standard labels for Dex resources.
func (r *LanguageClusterReconciler) dexLabels(cluster *langopv1alpha1.LanguageCluster) map[string]string {
	labels := GetCommonLabels("language-operator", DexResourceName)
	labels[langoplabels.LabelKeyK8sComponent] = DexResourceName
	labels[langoplabels.LabelKeyLangopCluster] = cluster.Name
	return labels
}

// agentAuthEnabled reports whether OIDC auth is active for the given agent.
// The agent's own spec.auth.enabled takes precedence; when nil, it inherits
// the cluster-level setting from spec.auth.enabled.
func agentAuthEnabled(agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster) bool {
	if agent.Spec.Auth != nil && agent.Spec.Auth.Enabled != nil {
		return *agent.Spec.Auth.Enabled
	}
	return cluster.Spec.Auth != nil && cluster.Spec.Auth.Enabled
}

// dexIssuerURL returns the Dex OIDC issuer URL for the given cluster.
// Falls back to the external issuer URL if configured.
func dexIssuerURL(cluster *langopv1alpha1.LanguageCluster) string {
	if cluster.Spec.Auth == nil {
		return ""
	}
	if cluster.Spec.Auth.OIDC != nil && cluster.Spec.Auth.OIDC.ExternalIssuerURL != "" {
		return cluster.Spec.Auth.OIDC.ExternalIssuerURL
	}
	if cluster.Spec.Domain == "" {
		return ""
	}
	scheme := "https"
	if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil &&
		cluster.Spec.Ingress.TLS.Enabled != nil && !*cluster.Spec.Ingress.TLS.Enabled {
		scheme = "http"
	}
	return fmt.Sprintf("%s://auth.%s", scheme, cluster.Spec.Domain)
}

// oauth2ProxyClientID returns the OIDC client ID to use for oauth2-proxy.
// For embedded Dex this is always "langop-agents"; for external OIDC it is
// taken from spec.auth.oidc.clientID.
func oauth2ProxyClientID(cluster *langopv1alpha1.LanguageCluster) string {
	if cluster.Spec.Auth == nil || cluster.Spec.Auth.OIDC == nil {
		return "langop-agents"
	}
	if cluster.Spec.Auth.OIDC.ExternalIssuerURL != "" && cluster.Spec.Auth.OIDC.ClientID != "" {
		return cluster.Spec.Auth.OIDC.ClientID
	}
	return "langop-agents"
}
