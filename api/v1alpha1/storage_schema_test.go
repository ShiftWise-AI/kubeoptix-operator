package v1alpha1

import (
	"os"
	"regexp"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestStorageSizeSchema(t *testing.T) {
	for _, path := range []string{
		"../../config/crd/bases/shiftwise.ai_shiftwises.yaml",
		"../../bundle/manifests/shiftwise.ai_shiftwises.yaml",
		"../../config/manifests/shiftwise-operator.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(data)), 4096)
			var crd apiextensionsv1.CustomResourceDefinition
			for crd.Kind == "" {
				if err := decoder.Decode(&crd); err != nil {
					t.Fatal(err)
				}
			}
			if crd.Kind != "CustomResourceDefinition" {
				t.Fatalf("expected CRD, got %s", crd.Kind)
			}
			storage := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["storage"]
			for _, required := range storage.Required {
				if required == "size" {
					t.Fatal("size must remain optional")
				}
			}
			pattern := storage.Properties["size"].Pattern
			if pattern == "" {
				t.Fatal("missing size validation")
			}
			re := regexp.MustCompile(pattern)
			// An explicit empty size uses the default, just like an omitted field.
			if !re.MatchString("") {
				t.Error("empty size must be accepted to preserve default storage behavior")
			}
			for _, value := range []string{"20Gi", "1.5Gi", "500M", "1e9", "1E+9", "1e-3", "1", ".5Gi", "+20Gi", "01Gi", "1.Gi", "1m", "1u", "1n", "0.01Gi"} {
				if !re.MatchString(value) {
					t.Errorf("valid quantity rejected: %q", value)
				}
				q, err := resource.ParseQuantity(value)
				if err != nil || q.Sign() <= 0 {
					t.Fatalf("invalid test quantity %q: %v", value, err)
				}
			}
			for _, value := range []string{" ", "\n", "invalid", "20GB", "1GiB", "0", "0Gi", "0.0Gi", "0e9", "-1Gi", " 20Gi", "20Gi\n", ".", "+", "1e", "1ki", "1K", "1.2.3Gi"} {
				if re.MatchString(value) {
					t.Errorf("invalid quantity accepted: %q", value)
				}
			}
		})
	}
}
