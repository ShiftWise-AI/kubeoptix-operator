package operands

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcileCoreAI(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	name := constants.CoreAIName
	ls := labels(name, name)
	if err := apply(ctx, c, scheme, owner, apiService(constants.CoreAIAPIService, s.Namespace, name, name, 8000, ls)); err != nil {
		return err
	}
	if err := applyUnstructured(ctx, c, imageStream(name, s.Namespace, ls, true)); err != nil {
		return err
	}
	bc := gitBuildConfig(name, s.Namespace, constants.CoreAIGitURI, constants.GitRef, s.GitSecret, constants.Containerfile, name+":"+constants.ImageTag, ls, "Serial", resourceQuantityMap("200m", "128Mi", "1", "1Gi"), configChangeTriggers())
	if err := applyUnstructured(ctx, c, bc); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, coreAISTS(s, ls))
}

func coreAISTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.CoreAIName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: constants.CoreAIAPIService,
			Replicas:    replicas(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken:  boolPtr(false),
					TerminationGracePeriodSeconds: int64Ptr(30),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot:   boolPtr(true),
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{{
						Name:            "api",
						Image:           s.CoreAIImage,
						ImagePullPolicy: corev1.PullAlways,
						Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: 8000, Protocol: corev1.ProtocolTCP}},
						Env: []corev1.EnvVar{
							envVar("HOME", "/tmp"),
							envVar("KUBECONFIG", "/tmp/.kube/config"),
							envVar("TZ", "America/Sao_Paulo"),
							envVar("LOG_LEVEL", "INFO"),
							envVar("PORT", "8000"),
							envVar("GRACEFUL_SHUTDOWN_TIMEOUT", "30"),
							envVar("MARK_DOWN_FILE", "/app/data/report.md"),
							envVar("KUBEOPTIX_METADATA_DIR", "/app/data/assessment"),
							envVar("KUBEOPTIX_OUTPUT_DIR", "/app/data/reports"),
						},
						Resources: cpuMem("500m", "1Gi", "2000m", "4Gi"),
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtr(false),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
						VolumeMounts: []corev1.VolumeMount{
							mount("tmp", "/tmp"),
							mount("logs", "/app/logs"),
							mount("data", "/app/data"),
						},
						StartupProbe:   httpProbe("/health/live", 8000, 10, 5, 2, 12),
						ReadinessProbe: httpProbe("/health/ready", 8000, 0, 10, 2, 3),
						LivenessProbe:  httpProbe("/health/live", 8000, 0, 10, 2, 3),
					}},
					Volumes: []corev1.Volume{
						emptyDirVolume("tmp"),
						emptyDirVolume("logs"),
						pvcVolume("data", s.ClaimName),
					},
				},
			},
		},
	}
	sts.Annotations = map[string]string{"kubeoptix.core-ai/single-instance": "true"}
	stsSelector(sts, name, name)
	return sts
}

func int64Ptr(v int64) *int64 { return &v }
