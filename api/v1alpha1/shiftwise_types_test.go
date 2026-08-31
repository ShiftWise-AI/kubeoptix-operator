package v1alpha1

import "testing"

func TestShiftWiseSpec_Namespace(t *testing.T) {
	t.Parallel()

	if got := (ShiftWiseSpec{}).Namespace(); got != "shiftwise-ai" {
		t.Fatalf("namespace = %q, want shiftwise-ai", got)
	}
	if got := (ShiftWiseSpec{Storage: StorageSpec{Size: "20Gi"}}).Namespace(); got != "shiftwise-ai" {
		t.Fatalf("namespace with storage = %q, want shiftwise-ai", got)
	}
}
