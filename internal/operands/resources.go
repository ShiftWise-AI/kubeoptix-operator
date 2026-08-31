package operands

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func cpuMem(reqCPU, reqMem, limCPU, limMem string) corev1.ResourceRequirements {
	out := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	if reqCPU != "" {
		out.Requests[corev1.ResourceCPU] = resource.MustParse(reqCPU)
	}
	if reqMem != "" {
		out.Requests[corev1.ResourceMemory] = resource.MustParse(reqMem)
	}
	if limCPU != "" {
		out.Limits[corev1.ResourceCPU] = resource.MustParse(limCPU)
	}
	if limMem != "" {
		out.Limits[corev1.ResourceMemory] = resource.MustParse(limMem)
	}
	if len(out.Requests) == 0 {
		out.Requests = nil
	}
	if len(out.Limits) == 0 {
		out.Limits = nil
	}
	return out
}

func accessModes(modes []string) []corev1.PersistentVolumeAccessMode {
	out := make([]corev1.PersistentVolumeAccessMode, 0, len(modes))
	for _, m := range modes {
		out = append(out, corev1.PersistentVolumeAccessMode(m))
	}
	return out
}
