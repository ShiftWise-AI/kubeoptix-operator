package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = shiftwisev1alpha1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	return s
}

func TestReconcile_CreatesDeploymentAndService(t *testing.T) {
	replicas := int32(2)
	app := &shiftwisev1alpha1.ShiftWiseApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: shiftwisev1alpha1.ShiftWiseAppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
		},
	}

	scheme := newScheme()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(app).WithStatusSubresource(app).Build()

	r := &ShiftWiseAppReconciler{Client: c, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-app", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check Deployment was created
	dep := &appsv1.Deployment{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "test-app", Namespace: "default"}, dep); err != nil {
		t.Fatalf("expected Deployment to exist: %v", err)
	}
	if *dep.Spec.Replicas != 2 {
		t.Errorf("expected 2 replicas, got %d", *dep.Spec.Replicas)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "nginx:latest" {
		t.Errorf("unexpected image: %s", dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Check Service was created
	svc := &corev1.Service{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "test-app", Namespace: "default"}, svc); err != nil {
		t.Fatalf("expected Service to exist: %v", err)
	}
	if svc.Spec.Ports[0].Port != 8080 {
		t.Errorf("expected port 8080, got %d", svc.Spec.Ports[0].Port)
	}
}

func TestReconcile_DefaultReplicas(t *testing.T) {
	app := &shiftwisev1alpha1.ShiftWiseApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-replicas-app",
			Namespace: "default",
		},
		Spec: shiftwisev1alpha1.ShiftWiseAppSpec{
			Image: "myapp:v1",
		},
	}

	scheme := newScheme()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(app).WithStatusSubresource(app).Build()

	r := &ShiftWiseAppReconciler{Client: c, Scheme: scheme}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-replicas-app", Namespace: "default"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dep := &appsv1.Deployment{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "no-replicas-app", Namespace: "default"}, dep); err != nil {
		t.Fatalf("expected Deployment to exist: %v", err)
	}
	if *dep.Spec.Replicas != 1 {
		t.Errorf("expected default 1 replica, got %d", *dep.Spec.Replicas)
	}
}

func TestReconcile_NotFound(t *testing.T) {
	scheme := newScheme()
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &ShiftWiseAppReconciler{Client: c, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("expected no error for missing object, got: %v", err)
	}
}

func TestLabelsForApp(t *testing.T) {
	labels := labelsForApp("myapp")
	if labels["app.kubernetes.io/instance"] != "myapp" {
		t.Errorf("unexpected instance label: %s", labels["app.kubernetes.io/instance"])
	}
	if labels["app.kubernetes.io/managed-by"] != "kubeoptix-operator" {
		t.Errorf("unexpected managed-by label: %s", labels["app.kubernetes.io/managed-by"])
	}
}
