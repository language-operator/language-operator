//go:build integration

package controllers_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Every resource in the gateway examples must be accepted by the API server: the
// CRD schema (enums, patterns, unknown fields) and its CEL rules. The webhook-level
// checks are covered by a unit test in api/v1alpha1.
func TestGatewayExamplesApplyCleanly(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "components", "model-gateway", "examples", "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples found: %v", err)
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	applied := 0
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, doc := range strings.Split(string(raw), "\n---") {
			obj := &unstructured.Unstructured{}
			if err := yaml.Unmarshal([]byte(doc), &obj.Object); err != nil {
				t.Fatalf("%s document %d: %v", filepath.Base(file), i, err)
			}
			if len(obj.Object) == 0 {
				continue
			}
			// Each example is applied on its own, so give it a unique name.
			obj.SetName(strings.TrimSuffix(filepath.Base(file), ".yaml") + "-" + obj.GetName())
			if err := k8sClient.Create(ctx, obj); err != nil {
				t.Errorf("%s document %d (%s): rejected by the API server: %v", filepath.Base(file), i, obj.GetKind(), err)
				continue
			}
			applied++
		}
	}
	if applied == 0 {
		t.Fatal("no example resources were applied")
	}
}
