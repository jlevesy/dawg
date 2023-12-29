package controller

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dawgv1 "github.com/jlevesy/dawg/api/v1"
)

// DashboardReconciler reconciles a Dashboard object
type DashboardReconciler struct {
	k8sClient client.Client
}

func NewDashboardReconciller() *DashboardReconciler {
	return &DashboardReconciler{}
}

//+kubebuilder:rbac:groups=dawg.urcloud.cc,resources=dashboards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dawg.urcloud.cc,resources=dashboards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dawg.urcloud.cc,resources=dashboards/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *DashboardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.k8sClient = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		For(&dawgv1.Dashboard{}).
		Complete(r)
}
