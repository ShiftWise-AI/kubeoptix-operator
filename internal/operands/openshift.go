package operands

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func imageStream(name, namespace string, ls map[string]string, lookupLocal bool) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk("image.openshift.io", "v1", "ImageStream"))
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(ls)
	_ = unstructured.SetNestedField(u.Object, lookupLocal, "spec", "lookupPolicy", "local")
	return u
}

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

func gitBuildConfig(name, namespace, uri, ref, secret, dockerfile, outputTag string, ls map[string]string, runPolicy string, resources map[string]interface{}, triggers []interface{}) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk("build.openshift.io", "v1", "BuildConfig"))
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(ls)
	source := map[string]interface{}{
		"type": "Git",
		"git": map[string]interface{}{
			"uri": uri,
			"ref": ref,
		},
	}
	if secret != "" {
		source["sourceSecret"] = map[string]interface{}{"name": secret}
	}
	spec := map[string]interface{}{
		"runPolicy": runPolicy,
		"source":    source,
		"strategy": map[string]interface{}{
			"type": "Docker",
			"dockerStrategy": map[string]interface{}{
				"dockerfilePath": dockerfile,
			},
		},
		"output": map[string]interface{}{
			"to": map[string]interface{}{
				"kind": "ImageStreamTag",
				"name": outputTag,
			},
		},
	}
	if resources != nil {
		spec["resources"] = resources
	}
	if triggers != nil {
		spec["triggers"] = triggers
	}
	_ = unstructured.SetNestedMap(u.Object, spec, "spec")
	return u
}

func binaryBuildConfig(name, namespace, dockerfile, outputTag string, ls map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk("build.openshift.io", "v1", "BuildConfig"))
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(ls)
	spec := map[string]interface{}{
		"runPolicy": "Serial",
		"source": map[string]interface{}{
			"type":   "Binary",
			"binary": map[string]interface{}{},
		},
		"strategy": map[string]interface{}{
			"type": "Docker",
			"dockerStrategy": map[string]interface{}{
				"dockerfilePath": dockerfile,
			},
		},
		"output": map[string]interface{}{
			"to": map[string]interface{}{
				"kind": "ImageStreamTag",
				"name": outputTag,
			},
		},
		"triggers": []interface{}{},
	}
	_ = unstructured.SetNestedMap(u.Object, spec, "spec")
	return u
}

func resourceQuantityMap(reqCPU, reqMem, limCPU, limMem string) map[string]interface{} {
	requests := map[string]interface{}{}
	limits := map[string]interface{}{}
	if reqCPU != "" {
		requests["cpu"] = reqCPU
	}
	if reqMem != "" {
		requests["memory"] = reqMem
	}
	if limCPU != "" {
		limits["cpu"] = limCPU
	}
	if limMem != "" {
		limits["memory"] = limMem
	}
	out := map[string]interface{}{}
	if len(requests) > 0 {
		out["requests"] = requests
	}
	if len(limits) > 0 {
		out["limits"] = limits
	}
	return out
}

func configChangeTriggers() []interface{} {
	return []interface{}{
		map[string]interface{}{"type": "ConfigChange"},
		map[string]interface{}{"type": "ImageChange"},
	}
}
