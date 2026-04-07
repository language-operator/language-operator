package controllers

import (
	"context"
	"fmt"
	"maps"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// reconcileServiceMonitor creates or deletes a Prometheus Operator ServiceMonitor for the agent.
// When spec.monitoring.serviceMonitor.enabled is true, a ServiceMonitor is created that selects
// the agent's Service. When disabled or nil, any existing ServiceMonitor is deleted.
// If prometheus-operator is not installed, the CRD will be absent and all errors are silently
// suppressed (meta.IsNoMatchError).
func (r *LanguageAgentReconciler) reconcileServiceMonitor(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	sm := &unstructured.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})
	sm.SetName(agent.Name)
	sm.SetNamespace(agent.Namespace)

	smSpec := agent.Spec.Monitoring
	if smSpec == nil || smSpec.ServiceMonitor == nil || !smSpec.ServiceMonitor.Enabled {
		if err := r.Delete(ctx, sm); err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to delete ServiceMonitor: %w", err)
		}
		return nil
	}

	cfg := smSpec.ServiceMonitor
	port := cfg.Port
	if port == "" {
		port = "http"
		if len(agent.Spec.Ports) > 0 {
			port = agent.Spec.Ports[0].Name
		}
	}
	path := cfg.Path
	if path == "" {
		path = "/metrics"
	}

	endpoint := map[string]any{
		"port": port,
		"path": path,
	}
	if cfg.Interval != "" {
		endpoint["interval"] = cfg.Interval
	}
	if cfg.ScrapeTimeout != "" {
		endpoint["scrapeTimeout"] = cfg.ScrapeTimeout
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	maps.Copy(labels, cfg.Labels)

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, sm, func() error {
		sm.SetLabels(labels)
		return unstructured.SetNestedField(sm.Object, map[string]any{
			"selector": map[string]any{
				"matchLabels": map[string]any{
					LabelKeyK8sName: agent.Name,
				},
			},
			"endpoints": []any{endpoint},
		}, "spec")
	})
	if meta.IsNoMatchError(err) {
		return nil
	}
	return err
}

// reconcilePrometheusRule creates or deletes a Prometheus Operator PrometheusRule for the agent.
// When spec.monitoring.rules is non-empty, a PrometheusRule is created with the provided groups.
// When empty or nil, any existing PrometheusRule is deleted.
// If prometheus-operator is not installed the CRD will be absent and errors are silently suppressed.
func (r *LanguageAgentReconciler) reconcilePrometheusRule(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	pr := &unstructured.Unstructured{}
	pr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PrometheusRule",
	})
	pr.SetName(agent.Name)
	pr.SetNamespace(agent.Namespace)

	if agent.Spec.Monitoring == nil || len(agent.Spec.Monitoring.Rules) == 0 {
		if err := r.Delete(ctx, pr); err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to delete PrometheusRule: %w", err)
		}
		return nil
	}

	groups := make([]any, 0, len(agent.Spec.Monitoring.Rules))
	for _, g := range agent.Spec.Monitoring.Rules {
		rules := make([]any, 0, len(g.Rules))
		for _, rule := range g.Rules {
			rm := map[string]any{
				"expr": rule.Expr,
			}
			if rule.Alert != "" {
				rm["alert"] = rule.Alert
			}
			if rule.Record != "" {
				rm["record"] = rule.Record
			}
			if rule.For != "" {
				rm["for"] = rule.For
			}
			if len(rule.Labels) > 0 {
				lm := make(map[string]any, len(rule.Labels))
				for k, v := range rule.Labels {
					lm[k] = v
				}
				rm["labels"] = lm
			}
			if len(rule.Annotations) > 0 {
				am := make(map[string]any, len(rule.Annotations))
				for k, v := range rule.Annotations {
					am[k] = v
				}
				rm["annotations"] = am
			}
			rules = append(rules, rm)
		}
		group := map[string]any{
			"name":  g.Name,
			"rules": rules,
		}
		if g.Interval != "" {
			group["interval"] = g.Interval
		}
		groups = append(groups, group)
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, pr, func() error {
		pr.SetLabels(labels)
		return unstructured.SetNestedField(pr.Object, map[string]any{
			"groups": groups,
		}, "spec")
	})
	if meta.IsNoMatchError(err) {
		return nil
	}
	return err
}
