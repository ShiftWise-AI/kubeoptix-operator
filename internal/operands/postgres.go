package operands

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcilePostgres(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	name := constants.PostgresName
	ls := labels(name, name)
	if s.CreatePostgres {
		if err := applyIfMissing(ctx, c, scheme, owner, postgresSecret(s, ls)); err != nil {
			return err
		}
	}
	sa := &corev1.ServiceAccount{ObjectMeta: objectMeta(name, s.Namespace, ls)}
	if err := apply(ctx, c, scheme, owner, sa); err != nil {
		return err
	}
	svc := &corev1.Service{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels(name, name),
			Ports: []corev1.ServicePort{{
				Name:       "postgresql",
				Port:       5432,
				TargetPort: intstr.FromString("postgresql"),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	if err := apply(ctx, c, scheme, owner, svc); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, postgresSTS(s, ls))
}

func postgresSecret(s Settings, ls map[string]string) *corev1.Secret {
	password := s.PostgresPassword
	if password == "" {
		password = "change-me-before-production"
	}
	return &corev1.Secret{
		ObjectMeta: objectMeta(s.PostgresSecret, s.Namespace, ls),
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"POSTGRESQL_USER":     s.PostgresUser,
			"POSTGRESQL_PASSWORD": password,
			"POSTGRESQL_DATABASE": s.PostgresDatabase,
		},
	}
}

func postgresSTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.PostgresName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    replicas(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					Containers: []corev1.Container{{
						Name:            "postgresql",
						Image:           s.PostgresImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							secretEnv("POSTGRESQL_USER", s.PostgresSecret, "POSTGRESQL_USER"),
							secretEnv("POSTGRESQL_PASSWORD", s.PostgresSecret, "POSTGRESQL_PASSWORD"),
							secretEnv("POSTGRESQL_DATABASE", s.PostgresSecret, "POSTGRESQL_DATABASE"),
						},
						Ports:        []corev1.ContainerPort{{Name: "postgresql", ContainerPort: 5432, Protocol: corev1.ProtocolTCP}},
						VolumeMounts: []corev1.VolumeMount{mount("data", "/var/lib/pgsql/data")},
						Resources:    cpuMem("100m", "256Mi", "", "512Mi"),
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "data", Labels: selectorLabels(name, name)},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(constants.PostgresStorageSize)},
					},
				},
			}},
		},
	}
	stsSelector(sts, name, name)
	return sts
}
