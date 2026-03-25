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

package reconciler

import (
	"context"
	"errors"
	"testing"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := langopv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add langop scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	return scheme
}

func newHelper(c client.Client) *ReconcileHelper[*langopv1alpha1.LanguagePersona] {
	return &ReconcileHelper[*langopv1alpha1.LanguagePersona]{
		Client:       c,
		TracerName:   "test-tracer",
		ResourceType: "persona",
	}
}

func TestStartReconcile_ResourceFound(t *testing.T) {
	scheme := newTestScheme(t)
	persona := &langopv1alpha1.LanguagePersona{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(persona).Build()
	helper := newHelper(fakeClient)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test", Namespace: "default"}}
	result, err := helper.StartReconcile(context.Background(), req, &langopv1alpha1.LanguagePersona{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Resource.Name != "test" {
		t.Errorf("expected resource name %q, got %q", "test", result.Resource.Name)
	}
	if result.Span == nil {
		t.Error("expected non-nil span")
	}
	if result.Ctx == nil {
		t.Error("expected non-nil ctx")
	}
	// Verify CompleteReconcile does not panic
	result.CompleteReconcile(nil)
}

func TestStartReconcile_ResourceNotFound(t *testing.T) {
	scheme := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	helper := newHelper(fakeClient)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"}}
	result, err := helper.StartReconcile(context.Background(), req, &langopv1alpha1.LanguagePersona{})

	if err != nil {
		t.Fatalf("expected nil error for not-found, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for not-found, got non-nil")
	}
}

// errorClient wraps a real client and injects an error on Get.
type errorClient struct {
	client.Client
}

func (e *errorClient) Get(_ context.Context, _ types.NamespacedName, _ client.Object, _ ...client.GetOption) error {
	return errors.New("transient API server error")
}

func TestStartReconcile_GetError(t *testing.T) {
	scheme := newTestScheme(t)
	base := fake.NewClientBuilder().WithScheme(scheme).Build()
	helper := newHelper(&errorClient{Client: base})

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test", Namespace: "default"}}
	result, err := helper.StartReconcile(context.Background(), req, &langopv1alpha1.LanguagePersona{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
}

func TestCompleteReconcile_Success(t *testing.T) {
	scheme := newTestScheme(t)
	persona := &langopv1alpha1.LanguagePersona{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(persona).Build()
	helper := newHelper(fakeClient)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test", Namespace: "default"}}
	result, err := helper.StartReconcile(context.Background(), req, &langopv1alpha1.LanguagePersona{})
	if err != nil {
		t.Fatalf("StartReconcile failed: %v", err)
	}

	// Must not panic
	result.CompleteReconcile(nil)
}

func TestCompleteReconcile_WithError(t *testing.T) {
	scheme := newTestScheme(t)
	persona := &langopv1alpha1.LanguagePersona{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(persona).Build()
	helper := newHelper(fakeClient)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test", Namespace: "default"}}
	result, err := helper.StartReconcile(context.Background(), req, &langopv1alpha1.LanguagePersona{})
	if err != nil {
		t.Fatalf("StartReconcile failed: %v", err)
	}

	// Must not panic and must record error on span
	result.CompleteReconcile(errors.New("reconcile failed"))
}

func TestCompleteReconcile_NilSpan(t *testing.T) {
	// Directly construct a ReconcileResult with a nil span to exercise the nil guard.
	result := &ReconcileResult[*langopv1alpha1.LanguagePersona]{
		Ctx:      context.Background(),
		Span:     nil,
		Resource: &langopv1alpha1.LanguagePersona{},
	}
	// Must not panic
	result.CompleteReconcile(nil)
	result.CompleteReconcile(errors.New("some error"))
}
