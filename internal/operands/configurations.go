package operands

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcileConfigurations(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	if err := ReconcilePostgres(ctx, c, scheme, owner, s); err != nil {
		return err
	}
	name := constants.ConfigurationsName
	ls := labels(name, name)
	sa := &corev1.ServiceAccount{ObjectMeta: objectMeta(name, s.Namespace, ls)}
	if err := apply(ctx, c, scheme, owner, sa); err != nil {
		return err
	}
	if err := apply(ctx, c, scheme, owner, apiService(constants.ConfigurationsAPIService, s.Namespace, name, name, 8000, ls)); err != nil {
		return err
	}
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
			Selector:     &metav1.LabelSelector{MatchLabels: selectorLabels(name, name)},
		},
	}
	if err := apply(ctx, c, scheme, owner, pdb); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, configurationsSTS(s, ls))
}

func configurationsSTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.ConfigurationsName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: constants.ConfigurationsAPIService,
			Replicas:    replicas(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					Containers: []corev1.Container{{
						Name:            name,
						Image:           s.ConfigurationsImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						EnvFrom:         []corev1.EnvFromSource{envFromSecret(s.PostgresSecret)},
						Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: 8000, Protocol: corev1.ProtocolTCP}},
						LivenessProbe:   httpProbe("/q/health/live", 8000, 20, 10, 3, 3),
						ReadinessProbe:  httpProbe("/q/health/ready", 8000, 10, 10, 3, 3),
						StartupProbe:    httpProbe("/q/health/started", 8000, 5, 5, 3, 24),
						Resources:       cpuMem("100m", "256Mi", "", "768Mi"),
					}},
				},
			},
		},
	}
	stsSelector(sts, name, name)
	return sts
}
