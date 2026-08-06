package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
)

const (
	conditionTypeAvailable   = "Available"
	conditionTypeProgressing = "Progressing"
	conditionReasonDeploying = "Deploying"
	conditionReasonReady     = "Ready"
)

// ShiftWiseAppReconciler reconciles a ShiftWiseApp object.
type ShiftWiseAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=shiftwise.ai,resources=shiftwiseapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=shiftwise.ai,resources=shiftwiseapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=shiftwise.ai,resources=shiftwiseapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile reads the state of a ShiftWiseApp object and reconciles the cluster state.
func (r *ShiftWiseAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	app := &shiftwisev1alpha1.ShiftWiseApp{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, app); err != nil {
		logger.Error(err, "failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, app); err != nil {
		logger.Error(err, "failed to reconcile Service")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, app); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ShiftWiseAppReconciler) reconcileDeployment(ctx context.Context, app *shiftwisev1alpha1.ShiftWiseApp) error {
	desired := r.desiredDeployment(app)
	if err := ctrl.SetControllerReference(app, desired, r.Scheme); err != nil {
		return err
	}

	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update if spec has changed
	if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) {
		updated := existing.DeepCopy()
		updated.Spec = desired.Spec
		return r.Update(ctx, updated)
	}
	return nil
}

func (r *ShiftWiseAppReconciler) reconcileService(ctx context.Context, app *shiftwisev1alpha1.ShiftWiseApp) error {
	desired := r.desiredService(app)
	if err := ctrl.SetControllerReference(app, desired, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update if ports or type changed
	if !equality.Semantic.DeepEqual(existing.Spec.Ports, desired.Spec.Ports) ||
		existing.Spec.Type != desired.Spec.Type {
		updated := existing.DeepCopy()
		updated.Spec.Ports = desired.Spec.Ports
		updated.Spec.Type = desired.Spec.Type
		return r.Update(ctx, updated)
	}
	return nil
}

func (r *ShiftWiseAppReconciler) updateStatus(ctx context.Context, app *shiftwisev1alpha1.ShiftWiseApp) error {
	// Fetch the current deployment to read ready replicas
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, dep); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	updated := app.DeepCopy()
	updated.Status.ReadyReplicas = dep.Status.ReadyReplicas

	desired := replicas(app)
	if dep.Status.ReadyReplicas == desired {
		setCondition(updated, conditionTypeAvailable, metav1.ConditionTrue, conditionReasonReady,
			fmt.Sprintf("All %d replicas are ready", desired))
		setCondition(updated, conditionTypeProgressing, metav1.ConditionFalse, conditionReasonReady, "Rollout complete")
	} else {
		setCondition(updated, conditionTypeAvailable, metav1.ConditionFalse, conditionReasonDeploying,
			fmt.Sprintf("%d/%d replicas ready", dep.Status.ReadyReplicas, desired))
		setCondition(updated, conditionTypeProgressing, metav1.ConditionTrue, conditionReasonDeploying, "Deployment in progress")
	}

	return r.Status().Update(ctx, updated)
}

// desiredDeployment returns the Deployment object for the given ShiftWiseApp.
func (r *ShiftWiseAppReconciler) desiredDeployment(app *shiftwisev1alpha1.ShiftWiseApp) *appsv1.Deployment {
	rep := replicas(app)
	labels := labelsForApp(app.Name)
	port := app.Spec.Port
	if port == 0 {
		port = 8080
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &rep,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  app.Name,
							Image: app.Spec.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: port, Protocol: corev1.ProtocolTCP},
							},
							Env:       app.Spec.Env,
							Resources: app.Spec.Resources,
						},
					},
				},
			},
		},
	}
}

// desiredService returns the Service object for the given ShiftWiseApp.
func (r *ShiftWiseAppReconciler) desiredService(app *shiftwisev1alpha1.ShiftWiseApp) *corev1.Service {
	labels := labelsForApp(app.Name)
	port := app.Spec.Port
	if port == 0 {
		port = 8080
	}
	svcType := app.Spec.ServiceType
	if svcType == "" {
		svcType = corev1.ServiceTypeClusterIP
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     svcType,
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstr.FromInt32(port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func labelsForApp(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "shiftwiseapp",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "kubeoptix-operator",
	}
}

func replicas(app *shiftwisev1alpha1.ShiftWiseApp) int32 {
	if app.Spec.Replicas != nil {
		return *app.Spec.Replicas
	}
	return 1
}

func setCondition(app *shiftwisev1alpha1.ShiftWiseApp, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range app.Status.Conditions {
		if c.Type == condType {
			if c.Status != status || c.Reason != reason {
				app.Status.Conditions[i].Status = status
				app.Status.Conditions[i].Reason = reason
				app.Status.Conditions[i].Message = message
				app.Status.Conditions[i].LastTransitionTime = now
			}
			return
		}
	}
	app.Status.Conditions = append(app.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShiftWiseAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&shiftwisev1alpha1.ShiftWiseApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
