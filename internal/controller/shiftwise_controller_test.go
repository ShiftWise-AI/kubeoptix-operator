package controller

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func TestReconcileInvalidStorageReportsError(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, appsv1.AddToScheme, shiftwisev1alpha1.AddToScheme} {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}
	instance := &shiftwisev1alpha1.ShiftWise{
		ObjectMeta: metav1.ObjectMeta{Name: "shiftwise", Namespace: "shiftwise-ai", Generation: 1, Finalizers: []string{constants.Finalizer}},
		Spec:       shiftwisev1alpha1.ShiftWiseSpec{Storage: shiftwisev1alpha1.StorageSpec{Size: "invalid"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(instance).WithObjects(instance).Build()
	r := &ShiftWiseReconciler{Client: c, Scheme: scheme}
	key := client.ObjectKeyFromObject(instance)
	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 {
		t.Fatal("expected retry after invalid storage")
	}
	got := &shiftwisev1alpha1.ShiftWise{}
	if err := c.Get(ctx, key, got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != "Error" || !strings.Contains(got.Status.Message, "spec.storage.size") {
		t.Fatalf("unexpected status: %+v", got.Status)
	}
	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := c.List(ctx, pvcs); err != nil {
		t.Fatal(err)
	}
	if len(pvcs.Items) != 0 {
		t.Fatal("invalid storage created a PVC")
	}
}
