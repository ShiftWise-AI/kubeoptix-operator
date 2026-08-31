package operands

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

func labels(name, instance string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/instance":   instance,
		"app.kubernetes.io/part-of":    constants.AppName,
		"app.kubernetes.io/managed-by": constants.ManagedBy,
	}
}

func selectorLabels(name, instance string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     name,
		"app.kubernetes.io/instance": instance,
	}
}

func objectMeta(name, namespace string, ls map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    ls,
	}
}

func envVar(name, value string) corev1.EnvVar {
	return corev1.EnvVar{Name: name, Value: value}
}

func envFromSecret(name string) corev1.EnvFromSource {
	return corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: name}}}
}

func envFromConfigMap(name string) corev1.EnvFromSource {
	return corev1.EnvFromSource{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: name}}}
}

func secretEnv(name, secret, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secret},
				Key:                  key,
			},
		},
	}
}

func int32Ptr(v int32) *int32 { return &v }

func boolPtr(v bool) *bool { return &v }

func replicas(n int32) *int32 { return &n }

func httpProbe(path string, port int32, initial, period, timeout, failure int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(port),
			},
		},
		InitialDelaySeconds: initial,
		PeriodSeconds:       period,
		TimeoutSeconds:      timeout,
		FailureThreshold:    failure,
	}
}

func pvcVolume(volumeName, claim string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
		},
	}
}

func emptyDirVolume(name string) corev1.Volume {
	return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
}

func mount(name, path string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: name, MountPath: path}
}

func dropAllCaps() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: boolPtr(false),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}
}

func stsSelector(sts *appsv1.StatefulSet, name, instance string) {
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: selectorLabels(name, instance)}
	if sts.Spec.Template.Labels == nil {
		sts.Spec.Template.Labels = map[string]string{}
	}
	for k, v := range selectorLabels(name, instance) {
		sts.Spec.Template.Labels[k] = v
	}
}
