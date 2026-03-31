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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLanguageClusterWebhookValidateCreate(t *testing.T) {
	h := &LanguageClusterWebhook{}
	cluster := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	warns, err := h.ValidateCreate(context.Background(), cluster)
	if err != nil {
		t.Errorf("ValidateCreate() unexpected error: %v", err)
	}
	if warns != nil {
		t.Errorf("ValidateCreate() unexpected warnings: %v", warns)
	}
}

func TestLanguageClusterWebhookValidateUpdate(t *testing.T) {
	h := &LanguageClusterWebhook{}
	cluster := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	warns, err := h.ValidateUpdate(context.Background(), cluster, cluster)
	if err != nil {
		t.Errorf("ValidateUpdate() unexpected error: %v", err)
	}
	if warns != nil {
		t.Errorf("ValidateUpdate() unexpected warnings: %v", warns)
	}
}

func TestLanguageClusterWebhookValidateDelete(t *testing.T) {
	h := &LanguageClusterWebhook{}
	cluster := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	warns, err := h.ValidateDelete(context.Background(), cluster)
	if err != nil {
		t.Errorf("ValidateDelete() unexpected error: %v", err)
	}
	if warns != nil {
		t.Errorf("ValidateDelete() unexpected warnings: %v", warns)
	}
}
