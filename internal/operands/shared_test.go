package operands

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func TestReconcileSharedPVCStorageSize(t *testing.T) {
	for _, tc := range []struct {
		size    string
		invalid bool
	}{
		{"20Gi", false}, {"1.5Gi", false}, {"1e9", false}, {"500M", false}, {"", false},
		{"invalid", true}, {"20GB", true}, {"1GiB", true}, {"0", true}, {"0Gi", true}, {"-1Gi", true}, {" ", true},
	} {
		t.Run(tc.size, func(t *testing.T) {
			ctx := context.Background()
			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			sw := newShiftWise()
			sw.Spec.Storage.Size = tc.size
			s := FromCR(sw)
			err := reconcileSharedPVC(ctx, c, scheme, nil, s)
			pvc := &corev1.PersistentVolumeClaim{}
			getErr := c.Get(ctx, client.ObjectKey{Name: s.ClaimName, Namespace: s.Namespace}, pvc)
			if tc.invalid {
				if err == nil || !strings.Contains(err.Error(), "spec.storage.size") {
					t.Fatalf("expected storage validation error, got %v", err)
				}
				if !apierrors.IsNotFound(getErr) {
					t.Fatalf("invalid size created PVC: %v", getErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if getErr != nil {
				t.Fatal(getErr)
			}
			want := tc.size
			if want == "" {
				want = constants.DefaultStorageSize
			}
			got := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			if got.Cmp(resource.MustParse(want)) != 0 {
				t.Fatalf("storage = %s, want %s", got.String(), want)
			}
		})
	}
}

func TestReconcileSharedPVCPreservesExistingClaim(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	sw := newShiftWise()
	sw.Spec.Storage.ExistingClaim = "existing-data"
	sw.Spec.Storage.Size = "invalid"
	s := FromCR(sw)
	existing := &corev1.PersistentVolumeClaim{ObjectMeta: objectMeta(s.ClaimName, s.Namespace, nil)}
	existing.Spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	if err := reconcileSharedPVC(ctx, c, scheme, nil, s); err != nil {
		t.Fatal(err)
	}
	got := &corev1.PersistentVolumeClaim{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(existing), got); err != nil {
		t.Fatal(err)
	}
	quantity := got.Spec.Resources.Requests[corev1.ResourceStorage]
	if quantity.Cmp(resource.MustParse("10Gi")) != 0 {
		t.Fatalf("existing storage changed to %s", quantity.String())
	}
}
