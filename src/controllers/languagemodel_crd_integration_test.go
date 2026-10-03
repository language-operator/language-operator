//go:build integration

package controllers_test

import (
	"strings"
	"testing"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// The LanguageModel schema rules are enforced by the API server itself, so they
// are checked against envtest rather than a fake client.
func TestLanguageModelCRDSchema(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "model-schema"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	model := func(name string, spec langopv1alpha1.LanguageModelSpec) *langopv1alpha1.LanguageModel {
		return &langopv1alpha1.LanguageModel{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns.Name}, Spec: spec}
	}

	for _, tc := range []struct {
		name    string
		spec    langopv1alpha1.LanguageModelSpec
		wantErr string
	}{
		{"provider only", langopv1alpha1.LanguageModelSpec{Provider: "gemini", ModelName: "gemini-2.5-pro"}, ""},
		{"litellmProvider only", langopv1alpha1.LanguageModelSpec{LiteLLMProvider: "deepseek", ModelName: "deepseek-chat"}, ""},
		{"neither", langopv1alpha1.LanguageModelSpec{ModelName: "m"}, "exactly one of provider or litellmProvider"},
		{"both", langopv1alpha1.LanguageModelSpec{Provider: "openai", LiteLLMProvider: "deepseek", ModelName: "m"}, "exactly one of provider or litellmProvider"},
		{"litellmProvider pattern", langopv1alpha1.LanguageModelSpec{LiteLLMProvider: "Deep-Seek", ModelName: "m"}, "litellmProvider"},
		{"custom still accepted", langopv1alpha1.LanguageModelSpec{Provider: "custom", ModelName: "m", Endpoint: "http://x"}, ""},
		{"typed params round-trip", langopv1alpha1.LanguageModelSpec{Provider: "bedrock", ModelName: "m", Region: "us-east-1",
			Params: map[string]apiextensionsv1.JSON{"extra_headers": {Raw: []byte(`{"X-Team":"a"}`)}, "use_chat_completions_api": {Raw: []byte(`true`)}}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := model(strings.ToLower(strings.ReplaceAll(tc.name, " ", "-")), tc.spec)
			err := k8sClient.Create(ctx, m)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("expected the API server to accept it, got %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("expected rejection mentioning %q, got %v", tc.wantErr, err)
			}
		})
	}

	got := &langopv1alpha1.LanguageModel{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: "typed-params-round-trip"}, got); err != nil {
		t.Fatal(err)
	}
	if string(got.Spec.Params["use_chat_completions_api"].Raw) != "true" {
		t.Fatalf("params lost their type: %s", got.Spec.Params["use_chat_completions_api"].Raw)
	}
}
