package operands

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func ReconcileAnalyzer(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner client.Object, s Settings) error {
	name := constants.AnalyzerName
	ls := labels(name, name)
	if s.CreateLLM {
		if err := applyIfMissing(ctx, c, scheme, owner, llmSecret(s, ls)); err != nil {
			return err
		}
	}
	if err := apply(ctx, c, scheme, owner, apiService(constants.AnalyzerAPIService, s.Namespace, name, name, 8000, ls)); err != nil {
		return err
	}
	if err := applyUnstructured(ctx, c, route(constants.AnalyzerRoute, s.Namespace, constants.AnalyzerAPIService, "http", "edge", "Redirect", ls, nil)); err != nil {
		return err
	}
	if err := applyMaxReplicaPolicy(ctx, c, scheme, name+"-max-replicas", s.Namespace, name, 1, ls); err != nil {
		return err
	}
	return apply(ctx, c, scheme, owner, analyzerSTS(s, ls))
}

func llmSecret(s Settings, ls map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: objectMeta(s.LLMSecret, s.Namespace, ls),
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"CURSOR_API_KEY": s.CursorAPIKey,
			"CURSOR_MODEL":   s.CursorModel,
			"LLM_API_KEY":    s.LLMAPIKey,
			"LLM_BASE_URL":   s.LLMBaseURL,
			"LLM_MODEL":      s.LLMModel,
		},
	}
}

func analyzerSTS(s Settings, ls map[string]string) *appsv1.StatefulSet {
	name := constants.AnalyzerName
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(name, s.Namespace, ls),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: constants.AnalyzerAPIService,
			Replicas:    replicas(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            name,
						Image:           s.AnalyzerImage,
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							envVar("HOME", "/tmp"),
							envVar("KUBECONFIG", "/tmp/kubeconfig"),
							envVar("ENV", "openshift"),
							envVar("TZ", "America/Sao_Paulo"),
						},
						EnvFrom:      []corev1.EnvFromSource{envFromSecret(s.LLMSecret)},
						Ports:        []corev1.ContainerPort{{Name: "http", ContainerPort: 8000, Protocol: corev1.ProtocolTCP}},
						VolumeMounts: []corev1.VolumeMount{mount("app-data", "/app/data")},
						Resources:    cpuMem("250m", "512Mi", "1000m", "2Gi"),
					}},
					Volumes: []corev1.Volume{pvcVolume("app-data", s.ClaimName)},
				},
			},
		},
	}
	stsSelector(sts, name, name)
	return sts
}
