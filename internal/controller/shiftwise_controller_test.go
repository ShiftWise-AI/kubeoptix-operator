package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func TestReconcileReadinessErrors(t *testing.T) {
	resource := schema.GroupResource{Group: "apps", Resource: "statefulsets"}
	for _, tc := range []struct {
		name                     string
		readinessErr, operandErr error
		wantPhase                string
	}{
		{name: "forbidden", readinessErr: apierrors.NewForbidden(resource, constants.HarvesterName, errors.New("denied")), wantPhase: "Error"},
		{name: "timeout", readinessErr: apierrors.NewTimeoutError("API timed out", 1), wantPhase: "Error"},
		{name: "combined errors", readinessErr: apierrors.NewServiceUnavailable("API unavailable"), operandErr: errors.New("PVC read failed"), wantPhase: "Error"},
		{name: "missing workload", readinessErr: apierrors.NewNotFound(resource, constants.HarvesterName), wantPhase: "Initializing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := runtime.NewScheme()
			if err := clientgoscheme.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := shiftwisev1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			sw := &shiftwisev1alpha1.ShiftWise{ObjectMeta: metav1.ObjectMeta{Name: "shiftwise", Namespace: "shiftwise-ai", UID: "test-uid", Generation: 1, Finalizers: []string{constants.Finalizer}}}
			reads := 0
			inject := true
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sw).WithStatusSubresource(sw).WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if inject {
						if _, ok := obj.(*corev1.PersistentVolumeClaim); ok && tc.operandErr != nil {
							return tc.operandErr
						}
						if _, ok := obj.(*appsv1.StatefulSet); ok && key.Name == constants.HarvesterName {
							reads++
							// Without an operand error, the first read belongs to workload creation;
							// the second is the readiness check. A PVC error skips workload creation.
							if reads == 2 || tc.operandErr != nil {
								return tc.readinessErr
							}
						}
					}
					return c.Get(ctx, key, obj, opts...)
				},
			}).Build()
			r := &ShiftWiseReconciler{Client: c, Scheme: scheme}
			key := client.ObjectKeyFromObject(sw)
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
			if tc.wantPhase == "Error" {
				if !errors.Is(err, tc.readinessErr) {
					t.Fatalf("readiness error not returned: %v", err)
				}
				if tc.operandErr != nil && !errors.Is(err, tc.operandErr) {
					t.Fatalf("operand error lost: %v", err)
				}
				if result != (ctrl.Result{}) {
					t.Fatalf("API errors should use controller retry: %+v", result)
				}
			} else if err != nil || result.RequeueAfter != requeueAfter {
				t.Fatalf("missing workload should use normal polling: result=%+v, err=%v", result, err)
			}
			got := &shiftwisev1alpha1.ShiftWise{}
			if err := c.Get(ctx, key, got); err != nil {
				t.Fatal(err)
			}
			if got.Status.Phase != tc.wantPhase || got.Status.ReadyComponents != "0/7" {
				t.Fatalf("unexpected status: %+v", got.Status)
			}
			if tc.wantPhase == "Error" {
				if !strings.Contains(got.Status.Message, tc.readinessErr.Error()) || !strings.Contains(got.Status.Message, "shiftwise-ai/"+constants.HarvesterName) {
					t.Fatalf("status lacks API error context: %s", got.Status.Message)
				}
				if tc.operandErr != nil && !strings.Contains(got.Status.Message, tc.operandErr.Error()) {
					t.Fatalf("status lost operand error: %s", got.Status.Message)
				}
			}
			// A successful later reconciliation must clear the previous API failure.
			inject = false
			result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
			if err != nil || result.RequeueAfter != requeueAfter {
				t.Fatalf("recovery failed: result=%+v, err=%v", result, err)
			}
			if err := c.Get(ctx, key, got); err != nil {
				t.Fatal(err)
			}
			if got.Status.Phase != "Initializing" || got.Status.Message != "reconciled KubeOptix inventory" {
				t.Fatalf("stale error after recovery: %+v", got.Status)
			}
		})
	}
}
