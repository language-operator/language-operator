package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/language-operator/language-operator/pkg/cni"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestCNIDetection tests CNI detection logic with different CNI plugins
// This is a fast unit test using fake Kubernetes client - no real cluster needed
func TestCNIDetection(t *testing.T) {
	tests := []struct {
		name                  string
		daemonSets            []*appsv1.DaemonSet
		configMaps            []*corev1.ConfigMap
		expectedCNI           string
		expectedNetworkPolicy bool
		expectedVersion       string
		expectError           bool
	}{
		{
			name: "Cilium detected",
			daemonSets: []*appsv1.DaemonSet{
				makeDaemonSet("cilium", "kube-system", "quay.io/cilium/cilium:v1.18.0"),
			},
			expectedCNI:           "cilium",
			expectedNetworkPolicy: true,
			expectedVersion:       "v1.18.0",
		},
		{
			name: "Calico detected",
			daemonSets: []*appsv1.DaemonSet{
				makeDaemonSet("calico-node", "kube-system", "docker.io/calico/node:v3.26.0"),
			},
			expectedCNI:           "calico",
			expectedNetworkPolicy: true,
			expectedVersion:       "v3.26.0",
		},
		{
			name: "Flannel detected (no NetworkPolicy)",
			daemonSets: []*appsv1.DaemonSet{
				makeDaemonSet("kube-flannel-ds-amd64", "kube-system", "quay.io/coreos/flannel:v0.21.0"),
			},
			expectedCNI:           "flannel",
			expectedNetworkPolicy: false,
			expectedVersion:       "v0.21.0",
		},
		{
			name: "Weave Net detected",
			daemonSets: []*appsv1.DaemonSet{
				makeDaemonSet("weave-net", "kube-system", "weaveworks/weave-kube:2.8.1"),
			},
			expectedCNI:           "weave",
			expectedNetworkPolicy: true,
			expectedVersion:       "2.8.1",
		},
		{
			name: "Antrea detected",
			daemonSets: []*appsv1.DaemonSet{
				makeDaemonSet("antrea-agent", "kube-system", "antrea/antrea-ubuntu:v1.13.0"),
			},
			expectedCNI:           "antrea",
			expectedNetworkPolicy: true,
			expectedVersion:       "v1.13.0",
		},
		{
			name:                  "No CNI detected",
			daemonSets:            []*appsv1.DaemonSet{},
			expectedCNI:           "none",
			expectedNetworkPolicy: false,
			expectError:           true,
		},
		{
			name:       "Cilium ConfigMap fallback",
			daemonSets: []*appsv1.DaemonSet{},
			configMaps: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cilium-config",
						Namespace: "kube-system",
					},
					Data: map[string]string{
						"cni-conf": `{"name": "cilium", "type": "cilium-cni"}`,
					},
				},
			},
			expectedCNI:           "cilium",
			expectedNetworkPolicy: true,
			expectedVersion:       "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake Kubernetes client with test objects
			objects := []runtime.Object{}
			for _, ds := range tt.daemonSets {
				objects = append(objects, ds)
			}
			for _, cm := range tt.configMaps {
				objects = append(objects, cm)
			}

			clientset := fake.NewSimpleClientset(objects...)
			ctx := context.Background()

			// Run CNI detection
			caps, err := cni.DetectNetworkPolicySupport(ctx, clientset)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err, "Should return error for: %s", tt.name)
			} else {
				assert.NoError(t, err, "Should not return error for: %s", tt.name)
			}

			// Verify detected CNI
			assert.Equal(t, tt.expectedCNI, caps.Name,
				"CNI name should match for: %s", tt.name)

			// Verify NetworkPolicy support
			assert.Equal(t, tt.expectedNetworkPolicy, caps.SupportsNetworkPolicy,
				"NetworkPolicy support should match for: %s", tt.name)

			// Verify version
			assert.Equal(t, tt.expectedVersion, caps.Version,
				"Version should match for: %s", tt.name)

			t.Logf("✓ CNI detection correct for: %s (detected: %s)", tt.name, caps.Name)
		})
	}
}

// TestCNIVersionExtraction tests version extraction from image names
func TestCNIVersionExtraction(t *testing.T) {
	tests := []struct {
		image           string
		expectedVersion string
	}{
		{"quay.io/cilium/cilium:v1.18.0", "v1.18.0"},
		{"docker.io/calico/node:v3.26.0", "v3.26.0"},
		{"weaveworks/weave-kube:2.8.1", "2.8.1"},
		{"antrea/antrea-ubuntu:v1.13.0", "v1.13.0"},
		{"image-without-version", "unknown"},
		{"image:latest", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			version := extractVersionFromImage(tt.image)
			assert.Equal(t, tt.expectedVersion, version,
				"Version extraction should match for: %s", tt.image)
		})
	}
}

// makeDaemonSet creates a test DaemonSet with the given name and image
func makeDaemonSet(name, namespace, image string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cni-container",
							Image: image,
						},
					},
				},
			},
		},
	}
}

// extractVersionFromImage is a helper to extract version from container image
func extractVersionFromImage(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return "unknown"
	}
	return parts[len(parts)-1]
}
