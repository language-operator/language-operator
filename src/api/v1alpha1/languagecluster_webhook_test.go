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

package v1alpha1

import (
	"context"
	"strings"
	"testing"

	"github.com/language-operator/language-operator/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLanguageClusterValidateSpec(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty domain is valid",
			domain:  "",
			wantErr: false,
		},
		{
			name:    "valid simple domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			domain:  "ai.example.com",
			wantErr: false,
		},
		{
			name:    "valid multi-level subdomain",
			domain:  "agents.ai.example.com",
			wantErr: false,
		},
		{
			name:    "domain with spaces",
			domain:  "ai example.com",
			wantErr: true,
			errMsg:  "not a valid DNS subdomain",
		},
		{
			name:    "domain with underscores",
			domain:  "ai_example.com",
			wantErr: true,
			errMsg:  "not a valid DNS subdomain",
		},
		{
			name:    "domain starting with hyphen",
			domain:  "-bad.example.com",
			wantErr: true,
			errMsg:  "not a valid DNS subdomain",
		},
		{
			name:    "domain ending with hyphen",
			domain:  "bad-.example.com",
			wantErr: true,
			errMsg:  "not a valid DNS subdomain",
		},
		{
			name:    "domain with uppercase letters",
			domain:  "AI.Example.com",
			wantErr: true,
			errMsg:  "not a valid DNS subdomain",
		},
		{
			name:    "domain too long",
			domain:  strings.Repeat("a", 254),
			wantErr: true,
			errMsg:  "not a valid DNS subdomain",
		},
		{
			name:    "single label",
			domain:  "localhost",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &LanguageCluster{
				Spec: LanguageClusterSpec{Domain: tt.domain},
			}
			err := lc.validateSpec()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSpec() error = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLanguageClusterWebhookValidateCreate(t *testing.T) {
	h := &LanguageClusterWebhook{}

	t.Run("valid cluster passes", func(t *testing.T) {
		lc := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec:       LanguageClusterSpec{Domain: "ai.example.com"},
		}
		_, err := h.ValidateCreate(context.Background(), lc)
		if err != nil {
			t.Errorf("ValidateCreate() unexpected error: %v", err)
		}
	})

	t.Run("invalid domain rejected", func(t *testing.T) {
		lc := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec:       LanguageClusterSpec{Domain: "invalid domain"},
		}
		_, err := h.ValidateCreate(context.Background(), lc)
		if err == nil {
			t.Error("ValidateCreate() expected error for invalid domain, got nil")
		}
	})
}

func TestLanguageClusterWebhookValidateUpdate(t *testing.T) {
	h := &LanguageClusterWebhook{}

	t.Run("valid update passes", func(t *testing.T) {
		old := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec:       LanguageClusterSpec{Domain: "old.example.com"},
		}
		lc := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec:       LanguageClusterSpec{Domain: "new.example.com"},
		}
		_, err := h.ValidateUpdate(context.Background(), old, lc)
		if err != nil {
			t.Errorf("ValidateUpdate() unexpected error: %v", err)
		}
	})

	t.Run("invalid domain rejected on update", func(t *testing.T) {
		old := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec:       LanguageClusterSpec{Domain: "valid.example.com"},
		}
		lc := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec:       LanguageClusterSpec{Domain: "bad domain!"},
		}
		_, err := h.ValidateUpdate(context.Background(), old, lc)
		if err == nil {
			t.Error("ValidateUpdate() expected error for invalid domain, got nil")
		}
	})

	t.Run("update during deletion is skipped", func(t *testing.T) {
		now := metav1.Now()
		old := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}}
		lc := &LanguageCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-cluster",
				DeletionTimestamp: &now,
			},
			Spec: LanguageClusterSpec{Domain: "bad domain!"},
		}
		_, err := h.ValidateUpdate(context.Background(), old, lc)
		if err != nil {
			t.Errorf("ValidateUpdate() during deletion should skip validation, got error: %v", err)
		}
	})
}

func TestLanguageClusterWebhookValidateDelete(t *testing.T) {
	h := &LanguageClusterWebhook{}
	lc := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}}
	_, err := h.ValidateDelete(context.Background(), lc)
	if err != nil {
		t.Errorf("ValidateDelete() unexpected error: %v", err)
	}
}

func newFakeWebhook(objs ...runtime.Object) *LanguageClusterWebhook {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = AddToScheme(scheme)
	b := fake.NewClientBuilder().WithScheme(scheme)
	for _, o := range objs {
		b = b.WithRuntimeObjects(o)
	}
	return &LanguageClusterWebhook{Client: b.Build()}
}

func TestLanguageClusterWebhookNamespaceConflict(t *testing.T) {
	lc := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"}}

	t.Run("no existing namespace — accepted", func(t *testing.T) {
		h := newFakeWebhook()
		_, err := h.ValidateCreate(context.Background(), lc)
		if err != nil {
			t.Errorf("ValidateCreate() unexpected error: %v", err)
		}
	})

	t.Run("langop-owned namespace — accepted", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-cluster",
				Labels: map[string]string{labels.LabelKeyLangopCluster: "my-cluster"},
			},
		}
		h := newFakeWebhook(ns)
		_, err := h.ValidateCreate(context.Background(), lc)
		if err != nil {
			t.Errorf("ValidateCreate() unexpected error: %v", err)
		}
	})

	t.Run("unmanaged namespace with same name — rejected", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
		}
		h := newFakeWebhook(ns)
		_, err := h.ValidateCreate(context.Background(), lc)
		if err == nil {
			t.Error("ValidateCreate() expected error for unmanaged namespace conflict, got nil")
		} else if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("ValidateCreate() error = %q, want to contain %q", err.Error(), "already exists")
		}
	})

	t.Run("namespace owned by different cluster — rejected", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-cluster",
				Labels: map[string]string{labels.LabelKeyLangopCluster: "other-cluster"},
			},
		}
		h := newFakeWebhook(ns)
		_, err := h.ValidateCreate(context.Background(), lc)
		if err == nil {
			t.Error("ValidateCreate() expected error for namespace owned by different cluster, got nil")
		}
	})
}

func TestLanguageClusterWebhookAdoptNamespace(t *testing.T) {
	adopting := &LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
		Spec:       LanguageClusterSpec{AdoptExistingNamespace: true},
	}

	t.Run("adoptExistingNamespace=true with unmanaged namespace — accepted", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
		}
		h := newFakeWebhook(ns)
		_, err := h.ValidateCreate(context.Background(), adopting)
		if err != nil {
			t.Errorf("ValidateCreate() unexpected error: %v", err)
		}
	})

	t.Run("adoptExistingNamespace=false with unmanaged namespace — rejected", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
		}
		notAdopting := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"}}
		h := newFakeWebhook(ns)
		_, err := h.ValidateCreate(context.Background(), notAdopting)
		if err == nil {
			t.Error("ValidateCreate() expected error when adopt flag is false, got nil")
		}
	})

	t.Run("adoptExistingNamespace=true with namespace owned by different cluster — rejected", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-cluster",
				Labels: map[string]string{labels.LabelKeyLangopCluster: "other-cluster"},
			},
		}
		h := newFakeWebhook(ns)
		_, err := h.ValidateCreate(context.Background(), adopting)
		if err == nil {
			t.Error("ValidateCreate() expected error when namespace is owned by a different cluster, got nil")
		}
	})
}

func TestLanguageClusterWebhookAdoptImmutable(t *testing.T) {
	old := &LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
		Spec:       LanguageClusterSpec{AdoptExistingNamespace: false},
	}
	updated := old.DeepCopy()
	updated.Spec.AdoptExistingNamespace = true

	h := newFakeWebhook()
	_, err := h.ValidateUpdate(context.Background(), old, updated)
	if err == nil {
		t.Error("ValidateUpdate() expected error when changing adoptExistingNamespace, got nil")
	} else if !strings.Contains(err.Error(), "immutable") {
		t.Errorf("ValidateUpdate() error = %q, want to contain %q", err.Error(), "immutable")
	}
}

func TestValidateNetworkPortPolicies(t *testing.T) {
	mkIngress := func(port int32) *AgentNetworkPolicies {
		return &AgentNetworkPolicies{
			Ingress: []NetworkIngressRule{{
				Ports: []NetworkPort{{Port: port}},
			}},
		}
	}
	mkEgress := func(port int32) *AgentNetworkPolicies {
		return &AgentNetworkPolicies{
			Egress: []NetworkEgressRule{{
				Ports: []NetworkPort{{Port: port}},
			}},
		}
	}

	tests := []struct {
		name     string
		policies *AgentNetworkPolicies
		wantErr  bool
	}{
		{"nil policies", nil, false},
		{"no ports", &AgentNetworkPolicies{}, false},
		{"ingress port 1 (min)", mkIngress(1), false},
		{"ingress port 65535 (max)", mkIngress(65535), false},
		{"ingress port 8080", mkIngress(8080), false},
		{"egress port 443", mkEgress(443), false},
		{"ingress port 0 — invalid", mkIngress(0), true},
		{"ingress port -1 — invalid", mkIngress(-1), true},
		{"egress port 65536 — invalid", mkEgress(65536), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNetworkPortPolicies(tt.policies)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNetworkPortPolicies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
