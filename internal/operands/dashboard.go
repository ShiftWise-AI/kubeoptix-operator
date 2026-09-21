package operands

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	if err := apply(ctx, c, scheme, owner, dashboardServiceAccount(s, ls)); err != nil {
		return err
	}
	if err := reconcileDashboardAuthDelegator(ctx, c, scheme, name+"-auth-delegator", s, ls); err != nil {
		return err
	}
	if err := applyIfMissing(ctx, c, scheme, owner, dashboardOAuthSecret(s, ls)); err != nil {
		return err
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.DashboardService,
			Namespace: s.Namespace,
			Labels:    ls,
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": constants.DashboardTLSSecret,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": name},
			Ports: []corev1.ServicePort{{
				Name:       "public",
				Port:       8443,
				TargetPort: intstr.FromString("public"),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	if err := apply(ctx, c, scheme, owner, svc); err != nil {
		return err
	}
	if err := applyUnstructured(ctx, c, route(constants.DashboardRoute, s.Namespace, constants.DashboardService, "public", "reencrypt", "Redirect", ls, map[string]string{
		"haproxy.router.openshift.io/timeout": "3m",
	})); err != nil {
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

// dashboardServiceAccount is the identity the oauth-proxy sidecar authenticates as.
// The redirect reference annotation lets the OpenShift OAuth server validate that the
// dashboard Route is an allowed redirect target for this service account's OAuth client.
func dashboardServiceAccount(s Settings, ls map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.DashboardServiceAccount,
			Namespace: s.Namespace,
			Labels:    ls,
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": fmt.Sprintf(
					`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":%q}}`,
					constants.DashboardRoute,
				),
			},
		},
	}
}

// reconcileDashboardAuthDelegator grants the dashboard service account permission to
// perform TokenReview/SubjectAccessReview calls, which oauth-proxy needs to validate
// bearer tokens issued by the cluster's OAuth server.
func reconcileDashboardAuthDelegator(ctx context.Context, c client.Client, scheme *runtime.Scheme, name string, s Settings, ls map[string]string) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: clusterMeta(name, ls),
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     constants.AuthDelegatorRole,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      constants.DashboardServiceAccount,
			Namespace: s.Namespace,
		}},
	}
	return apply(ctx, c, scheme, nil, crb)
}

// dashboardOAuthSecret holds the oauth-proxy cookie secret used to sign session cookies.
// It is created once (applyIfMissing) so existing sessions are not invalidated on every
// reconcile.
func dashboardOAuthSecret(s Settings, ls map[string]string) *corev1.Secret {
	cookieSecret, err := randomAlphanum(24)
	if err != nil {
		cookieSecret = fmt.Sprintf("kubeoptix-oauth-%s!", s.Instance)[:24]
	}
	return &corev1.Secret{
		ObjectMeta: objectMeta(constants.DashboardOAuthSecret, s.Namespace, ls),
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"session_secret": cookieSecret,
		},
	}
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
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            constants.DashboardServiceAccount,
					AutomountServiceAccountToken:  boolPtr(true),
					TerminationGracePeriodSeconds: int64Ptr(30),
					Containers: []corev1.Container{
						{
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
						},
						{
							// Authenticates incoming requests against the cluster's OAuth
							// server and forwards the resolved identity to the dashboard
							// container via X-Forwarded-User/-Email/-Preferred-Username.
							Name:            constants.DashboardOAuthContainer,
							Image:           constants.OAuthProxyImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"--https-address=:8443",
								"--provider=openshift",
								"--openshift-service-account=" + constants.DashboardServiceAccount,
								"--upstream=http://localhost:8080",
								"--tls-cert=/etc/tls/private/tls.crt",
								"--tls-key=/etc/tls/private/tls.key",
								"--cookie-secret-file=/etc/proxy/secrets/session_secret",
								"--cookie-secure=true",
								`--openshift-sar={"resource":"namespaces","verb":"get","name":"` + s.Namespace + `"}`,
								"--pass-user-headers=true",
							},
							Ports: []corev1.ContainerPort{{Name: "public", ContainerPort: 8443, Protocol: corev1.ProtocolTCP}},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "tls", MountPath: "/etc/tls/private", ReadOnly: true},
								{Name: "oauth-secret", MountPath: "/etc/proxy/secrets", ReadOnly: true},
							},
							Resources: cpuMem("50m", "64Mi", "200m", "128Mi"),
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
								ReadOnlyRootFilesystem:   boolPtr(true),
								RunAsNonRoot:             boolPtr(true),
								SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "tls", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: constants.DashboardTLSSecret}}},
						{Name: "oauth-secret", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: constants.DashboardOAuthSecret}}},
					},
				},
			},
		},
	}
	return sts
}
