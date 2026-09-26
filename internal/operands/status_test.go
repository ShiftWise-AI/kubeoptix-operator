package operands

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReadyCount(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		settings               Settings
		replicas               map[string]int32
		wantReady, wantDesired int
	}{
		{name: "no components", settings: Settings{}, wantDesired: 0},
		{name: "all missing", settings: FromCR(newShiftWise()), wantDesired: 7},
		{name: "all ready", settings: FromCR(newShiftWise()), replicas: map[string]int32{"kubeoptix-harvester": 1, "kubeoptix-db": 1, "kubeoptix-configurations": 1, "kubeoptix-analyzer": 1, "kubeoptix-core-ai": 1, "kubeoptix-reporter": 1, "kubeoptix-dashboard": 1}, wantReady: 7, wantDesired: 7},
		{name: "mixed missing unready and ready", settings: Settings{Namespace: "test", Harvester: true, Analyzer: true, Reporter: true}, replicas: map[string]int32{"kubeoptix-analyzer": 0, "kubeoptix-reporter": 1}, wantReady: 1, wantDesired: 3},
		{name: "disabled workloads ignored", settings: Settings{Namespace: "test", Harvester: true}, replicas: map[string]int32{"kubeoptix-harvester": 2, "kubeoptix-reporter": 1}, wantReady: 1, wantDesired: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := appsv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			var objects []client.Object
			for name, ready := range tc.replicas {
				objects = append(objects, &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tc.settings.Namespace}, Status: appsv1.StatefulSetStatus{ReadyReplicas: ready}})
			}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			ready, desired, err := ReadyCount(context.Background(), c, tc.settings)
			if err != nil || ready != tc.wantReady || desired != tc.wantDesired {
				t.Fatalf("ReadyCount = %d/%d, %v; want %d/%d, nil", ready, desired, err, tc.wantReady, tc.wantDesired)
			}
		})
	}
}

func TestReadyCountAPIErrors(t *testing.T) {
	resource := schema.GroupResource{Group: "apps", Resource: "statefulsets"}
	for _, failure := range []error{
		apierrors.NewForbidden(resource, "workload", errors.New("denied")),
		apierrors.NewTimeoutError("API timed out", 1),
		apierrors.NewServiceUnavailable("API unavailable"),
		context.Canceled,
	} {
		t.Run(failure.Error(), func(t *testing.T) {
			for _, failureIndex := range []int{0, 3} {
				s := FromCR(newShiftWise())
				names := enabledComponents(s)
				scheme := runtime.NewScheme()
				if err := appsv1.AddToScheme(scheme); err != nil {
					t.Fatal(err)
				}
				reads := 0
				c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
					Get: func(_ context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
						reads++
						if key.Name == names[failureIndex] {
							return failure
						}
						obj.(*appsv1.StatefulSet).Status.ReadyReplicas = 1
						return nil
					},
				}).Build()
				ready, desired, err := ReadyCount(context.Background(), c, s)
				if !errors.Is(err, failure) {
					t.Fatalf("error = %v, want wrapped %v", err, failure)
				}
				if !strings.Contains(err.Error(), s.Namespace+"/"+names[failureIndex]) {
					t.Fatalf("error lacks workload context: %v", err)
				}
				if ready != failureIndex || desired != len(names) {
					t.Fatalf("count = %d/%d, want %d/%d", ready, desired, failureIndex, len(names))
				}
				if reads != failureIndex+1 {
					t.Fatalf("continued reading after API error: %d reads", reads)
				}
			}
		})
	}
}
