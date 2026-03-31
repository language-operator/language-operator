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
	"fmt"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newWebhookTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return s
}

func makeClusterClient(t *testing.T, ns string) client.Client {
	t.Helper()
	s := newWebhookTestScheme(t)
	cluster := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	return fake.NewClientBuilder().WithScheme(s).WithObjects(cluster).Build()
}

func makeEmptyClient(t *testing.T) client.Client {
	t.Helper()
	s := newWebhookTestScheme(t)
	return fake.NewClientBuilder().WithScheme(s).Build()
}

func makeErrorClient(t *testing.T, injectedErr error) client.Client {
	t.Helper()
	s := newWebhookTestScheme(t)
	return fake.NewClientBuilder().WithScheme(s).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
			return injectedErr
		},
	}).Build()
}

func TestValidateClusterMembership(t *testing.T) {
	const ns = "test-ns"
	transientErr := fmt.Errorf("internal server error")

	tests := []struct {
		name    string
		client  client.Client
		wantErr bool
		errMsg  string
	}{
		{
			name:    "cluster exists",
			client:  makeClusterClient(t, ns),
			wantErr: false,
		},
		{
			name:    "cluster not found",
			client:  makeEmptyClient(t),
			wantErr: true,
			errMsg:  "no cluster",
		},
		{
			name:    "transient API error",
			client:  makeErrorClient(t, transientErr),
			wantErr: true,
			errMsg:  "failed to check LanguageCluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClusterMembership(context.Background(), tt.client, ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateClusterMembership() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateClusterMembership() error = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}
