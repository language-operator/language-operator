// Package cni provides CNI plugin detection and NetworkPolicy capability checking
package cni

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CNICapabilities represents the detected CNI plugin and its capabilities
type CNICapabilities struct {
	Name                  string
	SupportsNetworkPolicy bool
	Version               string
}

// cniInfo holds CNI detection metadata
type cniInfo struct {
	daemonSetName string
	supportsNP    bool
}

// Known CNI plugins and their NetworkPolicy support
var knownCNIs = map[string]cniInfo{
	"cilium": {
		daemonSetName: "cilium",
		supportsNP:    true,
	},
	"calico": {
		daemonSetName: "calico-node",
		supportsNP:    true,
	},
	"weave": {
		daemonSetName: "weave-net",
		supportsNP:    true,
	},
	"antrea": {
		daemonSetName: "antrea-agent",
		supportsNP:    true,
	},
	"flannel": {
		daemonSetName: "kube-flannel-ds",
		supportsNP:    false,
	},
}

// DetectNetworkPolicySupport probes the cluster to determine if NetworkPolicy is enforced
func DetectNetworkPolicySupport(ctx context.Context, client kubernetes.Interface) (*CNICapabilities, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	// Query DaemonSets in kube-system namespace
	daemonSets, err := client.AppsV1().DaemonSets("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets in kube-system: %w", err)
	}

	// Try to detect CNI by matching known DaemonSet names
	for cniName, info := range knownCNIs {
		for _, ds := range daemonSets.Items {
			if matchesCNI(ds.Name, info.daemonSetName) {
				version := extractVersion(ds.Spec.Template.Spec.Containers)
				return &CNICapabilities{
					Name:                  cniName,
					SupportsNetworkPolicy: info.supportsNP,
					Version:               version,
				}, nil
			}
		}
	}

	// If no known CNI found, check ConfigMaps for CNI configuration
	configMaps, err := client.CoreV1().ConfigMaps("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps in kube-system: %w", err)
	}

	for _, cm := range configMaps.Items {
		if cniName := detectCNIFromConfigMap(&cm); cniName != "" {
			if info, ok := knownCNIs[cniName]; ok {
				return &CNICapabilities{
					Name:                  cniName,
					SupportsNetworkPolicy: info.supportsNP,
					Version:               "unknown",
				}, nil
			}
		}
	}

	// No CNI detected
	return &CNICapabilities{
		Name:                  "none",
		SupportsNetworkPolicy: false,
		Version:               "",
	}, fmt.Errorf("no known CNI plugin detected in cluster")
}

// matchesCNI checks if a DaemonSet name matches the expected CNI DaemonSet pattern
func matchesCNI(dsName, expectedName string) bool {
	// Exact match
	if dsName == expectedName {
		return true
	}

	// Prefix match with architecture suffix (e.g., kube-flannel-ds-amd64)
	if strings.HasPrefix(dsName, expectedName) {
		return true
	}

	return false
}

// extractVersion attempts to extract version from container images
func extractVersion(containers []corev1.Container) string {
	for _, container := range containers {
		// Extract version from image tag
		// Example: quay.io/cilium/cilium:v1.18.0 -> v1.18.0
		parts := strings.Split(container.Image, ":")
		if len(parts) >= 2 {
			tag := parts[len(parts)-1]
			// Skip "latest" and return actual version tags
			if tag != "latest" {
				return tag
			}
		}
	}
	return "unknown"
}

// detectCNIFromConfigMap attempts to detect CNI from ConfigMap data
func detectCNIFromConfigMap(cm *corev1.ConfigMap) string {
	// Check common CNI ConfigMap names
	cmName := strings.ToLower(cm.Name)

	if strings.Contains(cmName, "cilium") {
		return "cilium"
	}
	if strings.Contains(cmName, "calico") {
		return "calico"
	}
	if strings.Contains(cmName, "weave") {
		return "weave"
	}
	if strings.Contains(cmName, "antrea") {
		return "antrea"
	}
	if strings.Contains(cmName, "flannel") {
		return "flannel"
	}

	// Check ConfigMap data for CNI type
	for key, value := range cm.Data {
		if strings.Contains(key, "cni-conf") || key == "10-calico.conflist" {
			valueLower := strings.ToLower(value)
			if strings.Contains(valueLower, "cilium") {
				return "cilium"
			}
			if strings.Contains(valueLower, "calico") {
				return "calico"
			}
			if strings.Contains(valueLower, "weave") {
				return "weave"
			}
			if strings.Contains(valueLower, "antrea") {
				return "antrea"
			}
			if strings.Contains(valueLower, "flannel") {
				return "flannel"
			}
		}
	}

	return ""
}
