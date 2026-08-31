package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
	"github.com/ShiftWise-AI/kubeoptix-operator/internal/operands"
)

const requeueAfter = time.Duration(constants.ReconcileInterval) * time.Second

// ShiftWiseReconciler reconciles a ShiftWise object.
type ShiftWiseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=shiftwise.ai,resources=shiftwises,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=shiftwise.ai,resources=shiftwises/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=shiftwise.ai,resources=shiftwises/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services;serviceaccounts;secrets;configmaps;persistentvolumeclaims;events;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingadmissionpolicies;validatingadmissionpolicybindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.openshift.io,resources=buildconfigs,verbs=get;list;watch;create;update;patch;delete

func (r *ShiftWiseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	instance := &shiftwisev1alpha1.ShiftWise{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	settings := operands.FromCR(instance)

	if !instance.DeletionTimestamp.IsZero() {
		if err := r.cleanup(ctx, settings); err != nil {
			return ctrl.Result{}, err
		}
		if controllerutil.RemoveFinalizer(instance, constants.Finalizer) {
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(instance, constants.Finalizer) {
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if err := operands.ReconcileNamespace(ctx, r.Client, settings); err != nil {
		return r.fail(ctx, instance, fmt.Errorf("namespace: %w", err))
	}

	var recErr error
	if settings.Harvester {
		recErr = operands.ReconcileHarvester(ctx, r.Client, r.Scheme, instance, settings)
	}
	if recErr == nil && settings.Configurations {
		recErr = operands.ReconcileConfigurations(ctx, r.Client, r.Scheme, instance, settings)
	}
	if recErr == nil && settings.Analyzer {
		recErr = operands.ReconcileAnalyzer(ctx, r.Client, r.Scheme, instance, settings)
	}
	if recErr == nil && settings.CoreAI {
		recErr = operands.ReconcileCoreAI(ctx, r.Client, r.Scheme, instance, settings)
	}
	if recErr == nil && settings.Reporter {
		recErr = operands.ReconcileReporter(ctx, r.Client, r.Scheme, instance, settings)
	}
	if recErr == nil && settings.Dashboard {
		recErr = operands.ReconcileDashboard(ctx, r.Client, r.Scheme, instance, settings)
	}

	ready, desired, _ := operands.ReadyCount(ctx, r.Client, settings)
	phase := operands.Phase(ready, desired, recErr)
	message := "reconciled KubeOptix inventory"
	if recErr != nil {
		message = recErr.Error()
		log.Error(recErr, "reconciliation failed")
	} else {
		log.Info("reconciled ShiftWise", "namespace", settings.Namespace, "phase", phase, "ready", operands.ReadyString(ready, desired))
	}
	if err := r.patchStatus(ctx, instance, phase, operands.ReadyString(ready, desired), message); err != nil {
		return ctrl.Result{}, err
	}
	if recErr != nil {
		return ctrl.Result{RequeueAfter: requeueAfter}, recErr
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *ShiftWiseReconciler) fail(ctx context.Context, instance *shiftwisev1alpha1.ShiftWise, recErr error) (ctrl.Result, error) {
	_ = r.patchStatus(ctx, instance, "Error", "0/0", recErr.Error())
	return ctrl.Result{RequeueAfter: requeueAfter}, recErr
}

func (r *ShiftWiseReconciler) cleanup(ctx context.Context, settings operands.Settings) error {
	for _, obj := range operands.ClusterScopedFor(settings) {
		if err := operands.DeleteIgnoreMissing(ctx, r.Client, obj); err != nil {
			return err
		}
	}
	return nil
}

func (r *ShiftWiseReconciler) patchStatus(ctx context.Context, instance *shiftwisev1alpha1.ShiftWise, phase, ready, message string) error {
	latest := &shiftwisev1alpha1.ShiftWise{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(instance), latest); err != nil {
		return err
	}
	latest.Status.Phase = phase
	latest.Status.ReadyComponents = ready
	latest.Status.Message = message
	latest.Status.ObservedGeneration = latest.Generation
	return r.Status().Update(ctx, latest)
}

func (r *ShiftWiseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&shiftwisev1alpha1.ShiftWise{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Named("shiftwise").
		Complete(r)
}
