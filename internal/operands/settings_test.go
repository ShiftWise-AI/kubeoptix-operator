package operands

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
)

func TestFromCRDefaults(t *testing.T) {
	t.Parallel()
	s := FromCR(newShiftWise())
	if s.Namespace != "shiftwise-ai" {
		t.Fatalf("namespace = %q", s.Namespace)
	}
	if s.ClaimName != "harvester-app-data" {
		t.Fatalf("claim = %q", s.ClaimName)
	}
	if s.PostgresSecret != "kubeoptix-db" {
		t.Fatalf("postgres secret = %q", s.PostgresSecret)
	}
	wantImage := "quay.io/parraes/kubeoptix-harvester:0.2.1"
	if s.HarvesterImage != wantImage {
		t.Fatalf("harvester image = %q, want %q", s.HarvesterImage, wantImage)
	}
	if !s.Harvester || !s.Configurations || !s.Analyzer || !s.CoreAI || !s.Reporter || !s.Dashboard {
		t.Fatal("all components should be enabled by default")
	}
	if got := enabledComponents(s); len(got) != 7 {
		t.Fatalf("enabled workloads = %d, want 7 (6 apps + postgres)", len(got))
	}
}

func TestFromCRStorage(t *testing.T) {
	t.Parallel()
	sw := newShiftWise()
	sw.Spec.Storage.Size = "50Gi"
	sw.Spec.Storage.Name = "custom-data"
	s := FromCR(sw)
	if s.StorageSize != "50Gi" {
		t.Fatalf("size = %q", s.StorageSize)
	}
	if s.ClaimName != "custom-data" {
		t.Fatalf("claim = %q", s.ClaimName)
	}
}

func TestPhase(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		Phase(0, 6, nil): "Initializing",
		Phase(3, 6, nil): "Progressing",
		Phase(6, 6, nil): "Ready",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
}

func TestRandomAlphanum(t *testing.T) {
	t.Parallel()
	a, err := randomAlphanum(24)
	if err != nil {
		t.Fatal(err)
	}
	b, err := randomAlphanum(24)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 24 || len(b) != 24 {
		t.Fatalf("len a=%d b=%d", len(a), len(b))
	}
	if a == b {
		t.Fatal("generated identical passwords")
	}
}

func newShiftWise() *shiftwisev1alpha1.ShiftWise {
	return &shiftwisev1alpha1.ShiftWise{
		ObjectMeta: metav1.ObjectMeta{Name: "shiftwise", Namespace: "shiftwise-ai"},
	}
}
