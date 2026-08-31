package operands

import (
	"context"
	"strconv"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func applyMaxReplicaPolicy(ctx context.Context, c client.Client, scheme *runtime.Scheme, name, namespace, stsName string, maxReplicas int32, ls map[string]string) error {
	expr := "!has(object.spec.replicas) || object.spec.replicas <= " + strconv.Itoa(int(maxReplicas))
	msg := "Scaling is restricted. Maximum replicas allowed for this workload is " + strconv.Itoa(int(maxReplicas)) + "."
	return applyAdmissionPolicy(ctx, c, scheme, name, name+"-binding", namespace, stsName, expr, msg, ls, false)
}

func applyExactReplicaPolicy(ctx context.Context, c client.Client, scheme *runtime.Scheme, name, namespace, stsName, expr, msg string, ls map[string]string, namespaceSelector bool) error {
	return applyAdmissionPolicy(ctx, c, scheme, name, name, namespace, stsName, expr, msg, ls, namespaceSelector)
}

func applyAdmissionPolicy(ctx context.Context, c client.Client, scheme *runtime.Scheme, policyName, bindingName, namespace, stsName, expr, msg string, ls map[string]string, namespaceSelector bool) error {
	fail := admissionv1.Fail
	deny := admissionv1.Deny
	policy := &admissionv1.ValidatingAdmissionPolicy{
		ObjectMeta: clusterMeta(policyName, ls),
		Spec: admissionv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: &fail,
			MatchConstraints: &admissionv1.MatchResources{
				ResourceRules: []admissionv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionv1.RuleWithOperations{
							Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update},
							Rule: admissionv1.Rule{
								APIGroups:   []string{"apps"},
								APIVersions: []string{"v1"},
								Resources:   []string{"statefulsets", "statefulsets/scale"},
							},
						},
					},
				},
			},
			MatchConditions: []admissionv1.MatchCondition{
				{
					Name:       "target-workload",
					Expression: "request.namespace == '" + namespace + "' && request.name == '" + stsName + "'",
				},
			},
			Validations: []admissionv1.Validation{
				{Expression: expr, Message: msg},
			},
		},
	}
	if err := apply(ctx, c, scheme, nil, policy); err != nil {
		return err
	}

	binding := &admissionv1.ValidatingAdmissionPolicyBinding{
		ObjectMeta: clusterMeta(bindingName, ls),
		Spec: admissionv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName:        policyName,
			ValidationActions: []admissionv1.ValidationAction{deny},
		},
	}
	if namespaceSelector {
		binding.Spec.MatchResources = &admissionv1.MatchResources{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"kubernetes.io/metadata.name": namespace},
			},
		}
	}
	return apply(ctx, c, scheme, nil, binding)
}
