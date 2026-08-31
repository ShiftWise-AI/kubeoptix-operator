package operands

import (
	"context"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcileNamespace(ctx context.Context, c client.Client, s Settings) error {
	existing := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: s.Namespace}, existing)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: s.Namespace,
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": constants.ManagedBy,
			"app.kubernetes.io/part-of":    constants.AppName,
			"app.kubernetes.io/name":       constants.AppName,
		},
	}}
	return c.Create(ctx, ns)
}

func reconcileSharedPVC(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	existing := &corev1.PersistentVolumeClaim{}
	err := c.Get(ctx, client.ObjectKey{Name: s.ClaimName, Namespace: s.Namespace}, existing)
	if err == nil || !apierrors.IsNotFound(err) {
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: objectMeta(s.ClaimName, s.Namespace, labels("data", s.Instance)),
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes(s.AccessModes),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(s.StorageSize),
				},
			},
		},
	}
	if s.StorageClass != "" {
		pvc.Spec.StorageClassName = &s.StorageClass
	}
	return applyIfMissing(ctx, c, scheme, owner, pvc)
}

func reconcileSharedServiceAccount(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: objectMeta(constants.SharedServiceAccount, s.Namespace, labels(constants.HarvesterName, s.Instance)),
	}
	return applyIfMissing(ctx, c, scheme, owner, sa)
}

func reconcileClusterReaderBinding(ctx context.Context, c client.Client, scheme *runtime.Scheme, name string, s Settings, ls map[string]string) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: clusterMeta(name, ls),
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     constants.ClusterReaderRole,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      constants.SharedServiceAccount,
			Namespace: s.Namespace,
		}},
	}
	return apply(ctx, c, scheme, nil, crb)
}

func ClusterScopedFor(s Settings) []client.Object {
	var objs []client.Object
	if s.Harvester {
		objs = append(objs,
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: constants.HarvesterName + "-cluster-reader"}},
			&admissionv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: constants.HarvesterName + "-max-replicas"}},
			&admissionv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: constants.HarvesterName + "-max-replicas-binding"}},
		)
	}
	if s.Analyzer {
		objs = append(objs,
			&admissionv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: constants.AnalyzerName + "-max-replicas"}},
			&admissionv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: constants.AnalyzerName + "-max-replicas-binding"}},
		)
	}
	if s.Reporter {
		objs = append(objs,
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: constants.ReporterName + "-cluster-reader"}},
			&admissionv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: constants.ReporterName}},
			&admissionv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: constants.ReporterName}},
		)
	}
	if s.Dashboard {
		objs = append(objs,
			&admissionv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: constants.DashboardName + "-single-replica"}},
			&admissionv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: constants.DashboardName + "-single-replica"}},
		)
	}
	return objs
}
