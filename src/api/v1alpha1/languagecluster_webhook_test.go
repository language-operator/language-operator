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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
