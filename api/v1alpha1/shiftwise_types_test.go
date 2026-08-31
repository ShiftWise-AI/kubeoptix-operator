package v1alpha1

import "testing"

func TestComponentSpec_IsEnabled(t *testing.T) {
	t.Parallel()

	enabled := true
	disabled := false

	cases := []struct {
		name string
		spec ComponentSpec
		want bool
	}{
		{name: "omitted defaults to enabled", spec: ComponentSpec{}, want: true},
		{name: "explicit true", spec: ComponentSpec{Enabled: &enabled}, want: true},
		{name: "explicit false", spec: ComponentSpec{Enabled: &disabled}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.spec.IsEnabled(); got != tc.want {
				t.Fatalf("IsEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShiftWiseSpec_Namespace(t *testing.T) {
	t.Parallel()

	if got := (ShiftWiseSpec{}).Namespace(); got != "shiftwise-ai" {
		t.Fatalf("default namespace = %q, want shiftwise-ai", got)
	}
	if got := (ShiftWiseSpec{TargetNamespace: "custom"}).Namespace(); got != "custom" {
		t.Fatalf("override namespace = %q, want custom", got)
	}
}
