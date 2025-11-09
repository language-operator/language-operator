package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/based/language-operator/pkg/cni"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestCNIDetection_Cilium tests CNI detection for Cilium CNI plugin
func TestCNIDetection_Cilium(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create Cilium DaemonSet
	ciliumDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cilium",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "cilium"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "cilium"},
				},
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

	err := env.k8sClient.Create(env.ctx, ciliumDS)
	if err != nil {
		t.Fatalf("Failed to create Cilium DaemonSet: %v", err)
	}

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	// Verify Cilium was detected
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

// TestCNIDetection_Calico tests CNI detection for Calico CNI plugin
func TestCNIDetection_Calico(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create Calico DaemonSet
	calicoDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "calico-node",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "calico-node"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "calico-node"},
				},
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

	err := env.k8sClient.Create(env.ctx, calicoDS)
	if err != nil {
		t.Fatalf("Failed to create Calico DaemonSet: %v", err)
	}

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	// Verify Calico was detected
	if caps.Name != "calico" {
		t.Errorf("Expected CNI name 'calico', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Calico to support NetworkPolicy")
	}

	if caps.Version != "v3.26.0" {
		t.Errorf("Expected version 'v3.26.0', got '%s'", caps.Version)
	}
}

// TestCNIDetection_Flannel tests CNI detection for Flannel CNI plugin (no NetworkPolicy support)
func TestCNIDetection_Flannel(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create Flannel DaemonSet with architecture suffix
	flannelDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-flannel-ds-amd64",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "flannel"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "flannel"},
				},
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

	err := env.k8sClient.Create(env.ctx, flannelDS)
	if err != nil {
		t.Fatalf("Failed to create Flannel DaemonSet: %v", err)
	}

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	// Verify Flannel was detected but doesn't support NetworkPolicy
	if caps.Name != "flannel" {
		t.Errorf("Expected CNI name 'flannel', got '%s'", caps.Name)
	}

	if caps.SupportsNetworkPolicy {
		t.Errorf("Expected Flannel to NOT support NetworkPolicy")
	}

	if caps.Version != "v0.21.0" {
		t.Errorf("Expected version 'v0.21.0', got '%s'", caps.Version)
	}
}

// TestCNIDetection_NoCNI tests CNI detection when no CNI plugin is present
func TestCNIDetection_NoCNI(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace exists but no CNI DaemonSets will be created

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
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

// TestCNIDetection_Timeout tests CNI detection with context timeout
func TestCNIDetection_Timeout(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Run CNI detection with expired context
	_, err := cni.DetectNetworkPolicySupport(ctx, env.clientset)
	if err == nil {
		t.Error("Expected error with timeout context, got nil")
	}

	// The error should be related to context cancellation
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got %v", ctx.Err())
	}
}

// TestCNIDetection_ConfigMapFallback tests CNI detection via ConfigMap when DaemonSet is not found
func TestCNIDetection_ConfigMapFallback(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create ConfigMap with Cilium configuration (no DaemonSet)
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

	env.CreateConfigMap(t, ciliumCM)

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	// Verify Cilium was detected from ConfigMap
	if caps.Name != "cilium" {
		t.Errorf("Expected CNI name 'cilium' from ConfigMap, got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Cilium to support NetworkPolicy")
	}

	// Version should be unknown when detected from ConfigMap
	if caps.Version != "unknown" {
		t.Errorf("Expected version 'unknown' from ConfigMap detection, got '%s'", caps.Version)
	}
}

// TestCNIDetection_WeaveNet tests CNI detection for Weave Net CNI plugin
func TestCNIDetection_WeaveNet(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create Weave Net DaemonSet
	weaveDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "weave-net",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "weave-net"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "weave-net"},
				},
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

	err := env.k8sClient.Create(env.ctx, weaveDS)
	if err != nil {
		t.Fatalf("Failed to create Weave Net DaemonSet: %v", err)
	}

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	// Verify Weave Net was detected
	if caps.Name != "weave" {
		t.Errorf("Expected CNI name 'weave', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Weave Net to support NetworkPolicy")
	}

	if caps.Version != "2.8.1" {
		t.Errorf("Expected version '2.8.1', got '%s'", caps.Version)
	}
}

// TestCNIDetection_Antrea tests CNI detection for Antrea CNI plugin
func TestCNIDetection_Antrea(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Note: kube-system namespace is automatically created by envtest

	// Create Antrea DaemonSet
	antreaDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "antrea-agent",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "antrea"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "antrea"},
				},
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

	err := env.k8sClient.Create(env.ctx, antreaDS)
	if err != nil {
		t.Fatalf("Failed to create Antrea DaemonSet: %v", err)
	}

	// Run CNI detection
	caps, err := cni.DetectNetworkPolicySupport(env.ctx, env.clientset)
	if err != nil {
		t.Fatalf("DetectNetworkPolicySupport() error = %v", err)
	}

	// Verify Antrea was detected
	if caps.Name != "antrea" {
		t.Errorf("Expected CNI name 'antrea', got '%s'", caps.Name)
	}

	if !caps.SupportsNetworkPolicy {
		t.Errorf("Expected Antrea to support NetworkPolicy")
	}

	if caps.Version != "v1.13.0" {
		t.Errorf("Expected version 'v1.13.0', got '%s'", caps.Version)
	}
}
