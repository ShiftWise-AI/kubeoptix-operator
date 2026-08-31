package operands

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcileHarvester(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	name := constants.HarvesterName
	ls := labels(name, name)
	if err := reconcileSharedPVC(ctx, c, scheme, owner, s); err != nil {
		return err
	}
	if err := reconcileSharedServiceAccount(ctx, c, scheme, owner, s); err != nil {
		return err
	}
	if err := reconcileClusterReaderBinding(ctx, c, scheme, name+"-cluster-reader", s, ls); err != nil {
		return err
	}
	if err := apply(ctx, c, scheme, owner, apiService(constants.HarvesterAPIService, s.Namespace, name, name, 8000, ls)); err != nil {
		return err
	}
	if err := deleteRoute(ctx, c, constants.HarvesterRoute, s.Namespace); err != nil {
		return err
	}
	if err := applyMaxReplicaPolicy(ctx, c, scheme, name+"-max-replicas", s.Namespace, name, 1, ls); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, harvesterSTS(s, ls))
}

func harvesterSTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.HarvesterName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: constants.HarvesterAPIService,
			Replicas:    replicas(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName:           constants.SharedServiceAccount,
					AutomountServiceAccountToken: boolPtr(true),
					Containers: []corev1.Container{{
						Name:            name,
						Image:           s.HarvesterImage,
						ImagePullPolicy: corev1.PullAlways,
						Command:         []string{"/bin/sh", "-c"},
						Args:            []string{harvesterStartupScript},
						Env: []corev1.EnvVar{
							envVar("HOME", "/tmp"),
							envVar("KUBECONFIG", "/tmp/.kube/config"),
							envVar("TZ", "America/Sao_Paulo"),
						},
						Ports:          []corev1.ContainerPort{{Name: "http", ContainerPort: 8000, Protocol: corev1.ProtocolTCP}},
						StartupProbe:   httpProbe("/health", 8000, 10, 5, 2, 12),
						ReadinessProbe: httpProbe("/health", 8000, 0, 10, 2, 3),
						LivenessProbe:  httpProbe("/health", 8000, 0, 10, 2, 3),
						VolumeMounts:   []corev1.VolumeMount{mount("app-data", "/app/data")},
						Resources:      cpuMem("500m", "1Gi", "2000m", "4Gi"),
					}},
					Volumes: []corev1.Volume{pvcVolume("app-data", s.ClaimName)},
				},
			},
		},
	}
	stsSelector(sts, name, name)
	return sts
}

func apiService(svcName, namespace, app, instance string, port int32, ls map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: objectMeta(svcName, namespace, ls),
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels(app, instance),
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       port,
				TargetPort: intstr.FromInt32(port),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
}
