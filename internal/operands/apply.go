package operands

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func setOwner(scheme *runtime.Scheme, owner client.Object, obj client.Object) error {
	if owner == nil || owner.GetNamespace() == "" {
		return nil
	}
	if obj.GetNamespace() == "" || obj.GetNamespace() != owner.GetNamespace() {
		return nil
	}
	return controllerutil.SetControllerReference(owner, obj, scheme)
}

func apply(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, obj client.Object) error {
	if err := setOwner(scheme, owner, obj); err != nil {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		existing := obj.DeepCopyObject().(client.Object)
		err := c.Get(ctx, client.ObjectKeyFromObject(obj), existing)
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, obj); err != nil {
				return fmt.Errorf("create %s/%s: %w", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
			}
			return nil
		}
		if err != nil {
			return err
		}
		obj.SetResourceVersion(existing.GetResourceVersion())
		if svc, ok := obj.(*corev1.Service); ok {
			if current, ok := existing.(*corev1.Service); ok {
				svc.Spec.ClusterIP = current.Spec.ClusterIP
				svc.Spec.ClusterIPs = current.Spec.ClusterIPs
				svc.Spec.IPFamilies = current.Spec.IPFamilies
				svc.Spec.IPFamilyPolicy = current.Spec.IPFamilyPolicy
			}
		}
		if err := c.Update(ctx, obj); err != nil {
			return fmt.Errorf("update %s/%s: %w", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
		}
		return nil
	})
}

func applyIfMissing(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, obj client.Object) error {
	if err := setOwner(scheme, owner, obj); err != nil {
		return err
	}
	err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj.DeepCopyObject().(client.Object))
	if apierrors.IsNotFound(err) {
		if err := c.Create(ctx, obj); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create %s/%s: %w", obj.GetName(), obj.GetNamespace(), err)
		}
		return nil
	}
	return err
}

func applyUnstructured(ctx context.Context, c client.Client, u *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(u.GroupVersionKind())
	err := c.Get(ctx, client.ObjectKeyFromObject(u), existing)
	if apierrors.IsNotFound(err) {
		if err := c.Create(ctx, u); err != nil {
			return fmt.Errorf("create %s/%s: %w", u.GetKind(), u.GetName(), err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	u.SetResourceVersion(existing.GetResourceVersion())
	if uid := existing.GetUID(); uid != "" {
		u.SetUID(uid)
	}
	if err := c.Update(ctx, u); err != nil {
		return fmt.Errorf("update %s/%s: %w", u.GetKind(), u.GetName(), err)
	}
	return nil
}

func DeleteIgnoreMissing(ctx context.Context, c client.Client, obj client.Object) error {
	err := c.Delete(ctx, obj)
	return client.IgnoreNotFound(err)
}

func clusterMeta(name string, ls map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Labels: ls}
}

func gvk(group, version, kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
}
