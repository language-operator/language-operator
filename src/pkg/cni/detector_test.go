package cni

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDetectNetworkPolicySupport_Cilium(t *testing.T) {
	ctx := context.Background()

	// Create fake DaemonSet for Cilium
	ciliumDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cilium",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cilium-agent",
							Image: "quay.io/cilium/cilium:v1.18.0",
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(ciliumDS)

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	if caps.Name != "cilium" {
		t.Errorf("Expected CNI name 'cilium', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Cilium to support NetworkPolicy")
	}

	if caps.Version != "v1.18.0" {
		t.Errorf("Expected version 'v1.18.0', got '%s'", caps.Version)
	}
}

func TestDetectNetworkPolicySupport_Calico(t *testing.T) {
	ctx := context.Background()

	// Create fake DaemonSet for Calico
	calicoDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "calico-node",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "calico-node",
							Image: "docker.io/calico/node:v3.26.0",
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(calicoDS)

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	if caps.Name != "calico" {
		t.Errorf("Expected CNI name 'calico', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Calico to support NetworkPolicy")
	}
}

func TestDetectNetworkPolicySupport_WeaveNet(t *testing.T) {
	ctx := context.Background()

	weaveDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "weave-net",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "weave",
							Image: "weaveworks/weave-kube:2.8.1",
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(weaveDS)

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	if caps.Name != "weave" {
		t.Errorf("Expected CNI name 'weave', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Weave Net to support NetworkPolicy")
	}
}

func TestDetectNetworkPolicySupport_Flannel(t *testing.T) {
	ctx := context.Background()

	// Create fake DaemonSet for Flannel (with architecture suffix)
	flannelDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-flannel-ds-amd64",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kube-flannel",
							Image: "quay.io/coreos/flannel:v0.21.0",
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(flannelDS)

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	if caps.Name != "flannel" {
		t.Errorf("Expected CNI name 'flannel', got '%s'", caps.Name)
	}

	if caps.SupportsNetworkPolicy {
		t.Errorf("Expected Flannel to NOT support NetworkPolicy")
	}
}

func TestDetectNetworkPolicySupport_Antrea(t *testing.T) {
	ctx := context.Background()

	antreaDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "antrea-agent",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "antrea-agent",
							Image: "antrea/antrea-ubuntu:v1.13.0",
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(antreaDS)

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	if caps.Name != "antrea" {
		t.Errorf("Expected CNI name 'antrea', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Antrea to support NetworkPolicy")
	}
}

func TestDetectNetworkPolicySupport_NoCNI(t *testing.T) {
	ctx := context.Background()

	// Empty cluster with no CNI DaemonSets
	client := fake.NewSimpleClientset()

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err == nil {
		t.Error("Expected error when no CNI is detected, got nil")
	}

	if caps.Name != "none" {
		t.Errorf("Expected CNI name 'none', got '%s'", caps.Name)
	}

	if caps.SupportsNetworkPolicy {
		t.Errorf("Expected no NetworkPolicy support when no CNI detected")
	}
}

func TestDetectNetworkPolicySupport_NilClient(t *testing.T) {
	ctx := context.Background()

	_, err := DetectNetworkPolicySupport(ctx, nil)
	if err == nil {
		t.Error("Expected error with nil client, got nil")
	}

	if err.Error() != "kubernetes client is nil" {
		t.Errorf("Expected 'kubernetes client is nil' error, got '%s'", err.Error())
	}
}

func TestDetectNetworkPolicySupport_ConfigMapFallback(t *testing.T) {
	ctx := context.Background()

	// Create ConfigMap with Cilium configuration
	ciliumCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cilium-config",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"cni-conf": `{
				"name": "cilium",
				"type": "cilium-cni"
			}`,
		},
	}

	client := fake.NewSimpleClientset(ciliumCM)

	caps, err := DetectNetworkPolicySupport(ctx, client)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	if caps.Name != "cilium" {
		t.Errorf("Expected CNI name 'cilium' from ConfigMap, got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Cilium to support NetworkPolicy")
	}

	if caps.Version != "unknown" {
		t.Errorf("Expected version 'unknown' from ConfigMap detection, got '%s'", caps.Version)
	}
}

func TestMatchesCNI(t *testing.T) {
	tests := []struct {
		name         string
		dsName       string
		expectedName string
		want         bool
	}{
		{
			name:         "exact match",
			dsName:       "cilium",
			expectedName: "cilium",
			want:         true,
		},
		{
			name:         "prefix match with arch suffix",
			dsName:       "kube-flannel-ds-amd64",
			expectedName: "kube-flannel-ds",
			want:         true,
		},
		{
			name:         "no match",
			dsName:       "some-other-ds",
			expectedName: "cilium",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesCNI(tt.dsName, tt.expectedName)
			if got != tt.want {
				t.Errorf("matchesCNI(%q, %q) = %v, want %v", tt.dsName, tt.expectedName, got, tt.want)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name       string
		containers []corev1.Container
		want       string
	}{
		{
			name: "version from image tag",
			containers: []corev1.Container{
				{
					Image: "quay.io/cilium/cilium:v1.18.0",
				},
			},
			want: "v1.18.0",
		},
		{
			name: "skip latest tag",
			containers: []corev1.Container{
				{
					Image: "nginx:latest",
				},
			},
			want: "unknown",
		},
		{
			name:       "no containers",
			containers: []corev1.Container{},
			want:       "unknown",
		},
		{
			name: "multiple containers, first has version",
			containers: []corev1.Container{
				{
					Image: "calico/node:v3.26.0",
				},
				{
					Image: "calico/cni:latest",
				},
			},
			want: "v3.26.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVersion(tt.containers)
			if got != tt.want {
				t.Errorf("extractVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectCNIFromConfigMap(t *testing.T) {
	tests := []struct {
		name string
		cm   *corev1.ConfigMap
		want string
	}{
		{
			name: "cilium from name",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cilium-config",
				},
			},
			want: "cilium",
		},
		{
			name: "calico from name",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "calico-config",
				},
			},
			want: "calico",
		},
		{
			name: "flannel from data",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-flannel-cfg",
				},
				Data: map[string]string{
					"cni-conf": `{"type": "flannel"}`,
				},
			},
			want: "flannel",
		},
		{
			name: "no CNI detected",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-other-config",
				},
			},
			want: "",
		},
		{
			name: "weave from name",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "weave-net-config",
				},
			},
			want: "weave",
		},
		{
			name: "antrea from name",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "antrea-config",
				},
			},
			want: "antrea",
		},
		{
			name: "calico from data content",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cni-config",
				},
				Data: map[string]string{
					"cni-conf": `{"name": "calico-network"}`,
				},
			},
			want: "calico",
		},
		{
			name: "cilium from data content",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cni-config",
				},
				Data: map[string]string{
					"cni-conf": `{"type": "cilium-cni"}`,
				},
			},
			want: "cilium",
		},
		{
			name: "weave from data content",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cni-config",
				},
				Data: map[string]string{
					"cni-conf": `{"type": "weave-net"}`,
				},
			},
			want: "weave",
		},
		{
			name: "antrea from data content",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cni-config",
				},
				Data: map[string]string{
					"cni-conf": `{"type": "antrea"}`,
				},
			},
			want: "antrea",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCNIFromConfigMap(tt.cm)
			if got != tt.want {
				t.Errorf("detectCNIFromConfigMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
