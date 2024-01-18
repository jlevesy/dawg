package controller

import (
	"context"
	"net/url"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	dawgv1 "github.com/jlevesy/dawg/api/v1"
	"github.com/jlevesy/dawg/generator"
	"github.com/jlevesy/dawg/pkg/grafana"
)

const finalizer = "dashboard.dawg.urcloud.cc/finalizer"

// DashboardReconciler reconciles a Dashboard object
type DashboardReconciler struct {
	k8sClient      client.Client
	generatorStore generator.Reader
	runtime        generator.Runtime
	grafana        *grafana.Client
}

func NewDashboardReconciller(store generator.Reader, runtime generator.Runtime, grafana *grafana.Client) *DashboardReconciler {
	return &DashboardReconciler{
		generatorStore: store,
		runtime:        runtime,
		grafana:        grafana,
	}
}

//+kubebuilder:rbac:groups=dawg.urcloud.cc,resources=dashboards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dawg.urcloud.cc,resources=dashboards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dawg.urcloud.cc,resources=dashboards/finalizers,verbs=update

// Reconcile handles dashboard reconciliation.
func (r *DashboardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dashboard dawgv1.Dashboard

	if err := r.k8sClient.Get(ctx, req.NamespacedName, &dashboard); err != nil {
		logger.Error(err, "Could not fetch the dashboard")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !dashboard.DeletionTimestamp.IsZero() {
		return r.deleteDashboard(ctx, &dashboard, logger)
	}

	return r.applyDashboard(ctx, &dashboard, logger)
}

func (r *DashboardReconciler) applyDashboard(ctx context.Context, dashboard *dawgv1.Dashboard, logger logr.Logger) (ctrl.Result, error) {
	logger = logger.WithValues("generator", dashboard.Spec.Generator)

	logger.V(1).Info("Applying Dashboard")

	if !controllerutil.ContainsFinalizer(dashboard, finalizer) {
		controllerutil.AddFinalizer(dashboard, finalizer)
		if err := r.k8sClient.Update(ctx, dashboard); err != nil {
			r.setFailureStatus(
				ctx,
				dashboard,
				"Could set finalizer",
				err,
				logger,
			)
			return ctrl.Result{}, err
		}
	}

	generatorURL, err := url.Parse(dashboard.Spec.Generator)
	if err != nil {
		r.setFailureStatus(
			ctx,
			dashboard,
			"Could not parse generator refererence as an URL",
			err,
			logger,
		)
		return ctrl.Result{}, err
	}

	generator, err := r.generatorStore.Load(ctx, generatorURL)
	if err != nil {
		r.setFailureStatus(
			ctx,
			dashboard,
			"Could not retrieve referenced generator",
			err,
			logger,
		)
		return ctrl.Result{}, err
	}

	genResult, err := r.runtime.Execute(ctx, generator, []byte(dashboard.Spec.Config))
	if err != nil {
		r.setFailureStatus(
			ctx,
			dashboard,
			"Could not execute generator",
			err, logger,
		)
		return ctrl.Result{}, err
	}

	dashboardResult, err := r.grafana.CreateDashboard(
		ctx,
		&grafana.CreateDashboardRequest{
			Dashboard: genResult.Payload,
			Overwrite: true,
		},
	)
	if err != nil {
		r.setFailureStatus(
			ctx,
			dashboard,
			"Could not create or update the dashboard in Grafana",
			err,
			logger,
		)
		return ctrl.Result{}, err
	}

	r.setSuccessStatus(ctx, dashboard, dashboardResult, logger)

	logger.V(1).Info("Applied dashboard", "grafana_id", dashboardResult.ID)

	return ctrl.Result{}, nil
}

func (r *DashboardReconciler) deleteDashboard(ctx context.Context, dashboard *dawgv1.Dashboard, logger logr.Logger) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(dashboard, finalizer) {
		return ctrl.Result{}, nil
	}

	logger.V(1).Info("Deleting Dashboard")

	_, err := r.grafana.DeleteDashboard(ctx, &grafana.DeleteDashboardRequest{UID: dashboard.Status.Grafana.UID})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(dashboard, finalizer)
	if err := r.k8sClient.Update(ctx, dashboard); err != nil {
		return ctrl.Result{}, err
	}

	logger.V(1).Info("Deleted dashboard")

	return ctrl.Result{}, nil
}

func (r *DashboardReconciler) setSuccessStatus(ctx context.Context, dashboard *dawgv1.Dashboard, grafanaResponse *grafana.CreateDashboardResponse, logger logr.Logger) {
	dashboard.Status.SyncStatus = string(dawgv1.DashboardStatusOK)
	dashboard.Status.Grafana.ID = grafanaResponse.ID
	dashboard.Status.Grafana.UID = grafanaResponse.UID
	dashboard.Status.Grafana.URL = grafanaResponse.URL
	dashboard.Status.Grafana.Version = grafanaResponse.Version
	dashboard.Status.Grafana.Slug = grafanaResponse.Slug
	dashboard.Status.Error = ""

	if err := r.k8sClient.Status().Update(ctx, dashboard); err != nil {
		logger.Error(err, "Could not update dasboard status")
	}
}

func (r *DashboardReconciler) setFailureStatus(ctx context.Context, dashboard *dawgv1.Dashboard, message string, err error, logger logr.Logger) {
	logger.Error(err, message)

	dashboard.Status.SyncStatus = string(dawgv1.DashboardStatusError)
	dashboard.Status.Grafana = dawgv1.GrafanaInfo{}
	dashboard.Status.Error = err.Error()

	if err := r.k8sClient.Status().Update(ctx, dashboard); err != nil {
		logger.Error(err, "Could not update dasboard status")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.k8sClient = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		For(&dawgv1.Dashboard{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
