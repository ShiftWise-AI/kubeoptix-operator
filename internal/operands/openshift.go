package operands

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func route(name, namespace, service, targetPort, termination, insecure string, ls map[string]string, extraAnnotations map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk("route.openshift.io", "v1", "Route"))
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(ls)
	if extraAnnotations != nil {
		u.SetAnnotations(extraAnnotations)
	}
	spec := map[string]interface{}{
		"to": map[string]interface{}{
			"kind":   "Service",
			"name":   service,
			"weight": int64(100),
		},
		"port": map[string]interface{}{
			"targetPort": targetPort,
		},
		"tls": map[string]interface{}{
			"termination":                   termination,
			"insecureEdgeTerminationPolicy": insecure,
		},
		"wildcardPolicy": "None",
	}
	_ = unstructured.SetNestedMap(u.Object, spec, "spec")
	return u
}
