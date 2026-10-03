package v1alpha1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// The LanguageModel examples shipped with the gateway must be valid: every field
// known to the CRD, and accepted by the admission webhook's checks. Nothing else
// keeps documentation examples honest as the API moves.
func TestGatewayExamplesAreValidLanguageModels(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "components", "model-gateway", "examples", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no examples found; has the directory moved?")
	}

	models := 0
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, doc := range strings.Split(string(raw), "\n---") {
			var meta struct{ Kind string }
			if err := yaml.Unmarshal([]byte(doc), &meta); err != nil {
				t.Errorf("%s document %d: not YAML: %v", filepath.Base(file), i, err)
				continue
			}
			if meta.Kind != "LanguageModel" {
				continue
			}
			models++
			var m LanguageModel
			if err := yaml.UnmarshalStrict([]byte(doc), &m); err != nil {
				t.Errorf("%s document %d: %v", filepath.Base(file), i, err)
				continue
			}
			name := filepath.Base(file) + " " + m.Name
			if (m.Spec.Provider == "") == (m.Spec.LiteLLMProvider == "") {
				t.Errorf("%s: set exactly one of provider or litellmProvider", name)
			}
			if err := validateModelSpec(&m.Spec); err != nil {
				t.Errorf("%s: %v", name, err)
			}
			if m.Namespace == "langop-system" {
				t.Errorf("%s: namespace langop-system is not a LanguageCluster namespace; use the cluster's", name)
			}
		}
	}
	if models == 0 {
		t.Fatal("no LanguageModel documents found in the examples")
	}
}
