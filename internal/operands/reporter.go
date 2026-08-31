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

func ReconcileReporter(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	name := constants.ReporterName
	ls := labels(name, name)
	if err := reconcileSharedServiceAccount(ctx, c, scheme, owner, s); err != nil {
		return err
	}
	if err := reconcileClusterReaderBinding(ctx, c, scheme, name+"-cluster-reader", s, ls); err != nil {
		return err
	}
	if err := apply(ctx, c, scheme, owner, apiService(constants.ReporterAPIService, s.Namespace, name, name, 8000, ls)); err != nil {
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
	if err := applyUnstructured(ctx, c, imageStream(name, s.Namespace, ls, true)); err != nil {
		return err
	}
	bc := gitBuildConfig(name, s.Namespace, constants.ReporterGitURI, constants.GitRef, s.GitSecret, constants.Containerfile, name+":"+constants.ImageTag, ls, "SerialLatestOnly", resourceQuantityMap("200m", "128Mi", "1", "1Gi"), []interface{}{map[string]interface{}{"type": "ConfigChange"}})
	if err := applyUnstructured(ctx, c, bc); err != nil {
		return err
	}
	if err := applyExactReplicaPolicy(ctx, c, scheme, name, s.Namespace, name, "object.spec.replicas == 1", "KubeOptix Reporter must run with exactly one replica.", ls, false); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, reporterSTS(s, ls))
}

func reporterSTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.ReporterName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         constants.ReporterAPIService,
			Replicas:            replicas(1),
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: constants.SharedServiceAccount,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot:   boolPtr(true),
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{{
						Name:            name,
						Image:           s.ReporterImage,
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: dropAllCaps(),
						Env: []corev1.EnvVar{
							envVar("DATA_DIR", "/app/data/reports"),
							envVar("LOG_LEVEL", "INFO"),
							envVar("TZ", "America/Sao_Paulo"),
							envVar("HOME", "/tmp"),
							envVar("KUBECONFIG", "/tmp/.kube/config"),
						},
						Ports:          []corev1.ContainerPort{{Name: "http", ContainerPort: 8000, Protocol: corev1.ProtocolTCP}},
						StartupProbe:   httpProbe("/health", 8000, 10, 5, 2, 12),
						ReadinessProbe: httpProbe("/health", 8000, 0, 10, 2, 3),
						LivenessProbe:  httpProbe("/health", 8000, 0, 10, 2, 3),
						Resources:      cpuMem("500m", "1Gi", "2000m", "4Gi"),
						VolumeMounts:   []corev1.VolumeMount{mount("app-data", "/app/data")},
					}},
					Volumes: []corev1.Volume{pvcVolume("app-data", s.ClaimName)},
				},
			},
		},
	}
	stsSelector(sts, name, name)
	return sts
}
