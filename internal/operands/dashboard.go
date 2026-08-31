package operands

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcileDashboard(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	name := constants.DashboardName
	ls := labels(name, constants.AppName)
	if err := apply(ctx, c, scheme, owner, dashboardConfigMap(s, ls)); err != nil {
		return err
	}
	svc := &corev1.Service{
		ObjectMeta: objectMeta(constants.DashboardService, s.Namespace, ls),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": name},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       8080,
				TargetPort: intstr.FromString("http"),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	if err := apply(ctx, c, scheme, owner, svc); err != nil {
		return err
	}
	if err := applyUnstructured(ctx, c, route(constants.DashboardRoute, s.Namespace, constants.DashboardService, "http", "edge", "Redirect", ls, map[string]string{
		"haproxy.router.openshift.io/timeout": "3m",
	})); err != nil {
		return err
	}
	if err := applyUnstructured(ctx, c, imageStream(name, s.Namespace, ls, true)); err != nil {
		return err
	}
	if err := applyUnstructured(ctx, c, binaryBuildConfig(name, s.Namespace, constants.Containerfile, name+":"+constants.ImageTag, ls)); err != nil {
		return err
	}
	if err := applyExactReplicaPolicy(ctx, c, scheme, name+"-single-replica", s.Namespace, name,
		"has(object.spec.replicas) && object.spec.replicas == 1",
		"The kubeoptix-dashboard StatefulSet must run with exactly one replica.",
		ls, true); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, dashboardSTS(s, ls))
}

func dashboardConfigMap(s Settings, ls map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: objectMeta(constants.DashboardConfigMap, s.Namespace, ls),
		Data: map[string]string{
			"ENV":               "openshift",
			"HARVESTER_API_URL": "http://" + constants.HarvesterAPIService + ":8000",
			"ANALYZER_API_URL":  "http://" + constants.AnalyzerAPIService + ":8000",
			"REPORTER_API_URL":  "http://" + constants.ReporterAPIService + ":8000",
			"CORE_AI_API_URL":   "http://" + constants.CoreAIAPIService + "." + s.Namespace + ".svc.cluster.local:8000",
			"SETTINGS_API_URL":  "http://" + constants.ConfigurationsAPIService + ":8000",
			"TZ":                "America/Sao_Paulo",
		},
	}
}

func dashboardSTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.DashboardName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: constants.DashboardService,
			Replicas:    replicas(1),
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":    name,
						"app.kubernetes.io/part-of": constants.AppName,
					},
					Annotations: map[string]string{
						"image.openshift.io/triggers": `[{"from":{"kind":"ImageStreamTag","name":"kubeoptix-dashboard:latest","namespace":"` + s.Namespace + `"},"fieldPath":"spec.template.spec.containers[?(@.name==\"dashboard\")].image"}]`,
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken:  boolPtr(false),
					TerminationGracePeriodSeconds: int64Ptr(30),
					Containers: []corev1.Container{{
						Name:            constants.DashboardContainer,
						Image:           s.DashboardImage,
						ImagePullPolicy: corev1.PullAlways,
						EnvFrom:         []corev1.EnvFromSource{envFromConfigMap(constants.DashboardConfigMap)},
						Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP}},
						ReadinessProbe:  httpProbe("/healthz", 8080, 3, 10, 2, 3),
						LivenessProbe:   httpProbe("/healthz", 8080, 10, 20, 2, 3),
						Resources:       cpuMem("50m", "64Mi", "500m", "256Mi"),
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtr(false),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							ReadOnlyRootFilesystem:   boolPtr(true),
							RunAsNonRoot:             boolPtr(true),
							SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
						},
					}},
				},
			},
		},
	}
	return sts
}
