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
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func testModel(name, modelName string) *LanguageModel {
	return &LanguageModel{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team"},
		Spec:       LanguageModelSpec{Provider: "openai-compatible", ModelName: modelName},
	}
}

func newModelWebhook(t *testing.T, funcs *interceptor.Funcs, objs ...client.Object) *LanguageModelWebhook {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	objs = append(objs, &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "team"}})
	b := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...)
	if funcs != nil {
		b = b.WithInterceptorFuncs(*funcs)
	}
	return &LanguageModelWebhook{Client: b.Build()}
}

// Two LanguageModels with the same modelName become one load-balanced group in
// the gateway. That is allowed, but admission says so.
func TestLanguageModelWebhook_SharedModelNameWarns(t *testing.T) {
	ctx := context.Background()
	h := newModelWebhook(t, nil, testModel("ollama-a", "llama3.2"), testModel("other", "qwen3"))

	warnings, err := h.ValidateCreate(ctx, testModel("ollama-b", "llama3.2"))
	if err != nil {
		t.Fatalf("a shared modelName must be allowed, got %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "ollama-a") || !strings.Contains(warnings[0], "load-balances") {
		t.Fatalf("expected one load-balancing warning naming ollama-a, got %q", warnings)
	}

	// Updating a model into an existing name warns too.
	warnings, err = h.ValidateUpdate(ctx, testModel("other", "qwen3"), testModel("other", "llama3.2"))
	if err != nil || len(warnings) != 1 {
		t.Fatalf("update into a shared modelName: warnings=%q err=%v", warnings, err)
	}
}

func TestLanguageModelWebhook_UniqueModelNameIsQuiet(t *testing.T) {
	ctx := context.Background()
	self := testModel("ollama-a", "llama3.2")
	h := newModelWebhook(t, nil, self, testModel("other", "qwen3"))

	if warnings, err := h.ValidateCreate(ctx, testModel("new", "mistral")); err != nil || len(warnings) != 0 {
		t.Fatalf("unique modelName: warnings=%q err=%v", warnings, err)
	}
	// The stored copy of the model being updated is not a collision with itself.
	if warnings, err := h.ValidateUpdate(ctx, self, self); err != nil || len(warnings) != 0 {
		t.Fatalf("updating a model against its own stored copy: warnings=%q err=%v", warnings, err)
	}
}

// The warning is advisory: if the lookup fails, admission proceeds without it.
func TestLanguageModelWebhook_ListFailureAddsNoWarning(t *testing.T) {
	funcs := &interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*LanguageModelList); ok {
				return errors.New("api server unavailable")
			}
			return c.List(ctx, list, opts...)
		},
	}
	h := newModelWebhook(t, funcs, testModel("ollama-a", "llama3.2"))

	warnings, err := h.ValidateCreate(context.Background(), testModel("ollama-b", "llama3.2"))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("failed lookup must neither block nor warn: warnings=%q err=%v", warnings, err)
	}
}
