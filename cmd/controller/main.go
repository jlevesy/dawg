package main

import (
	"context"
	"flag"
	"os"

	dawgv1 "github.com/jlevesy/dawg/api/v1"
	"github.com/jlevesy/dawg/generator"
	"github.com/jlevesy/dawg/internal/controller"
	"github.com/jlevesy/dawg/pkg/grafana"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dawgv1.AddToScheme(scheme))
}

func main() {
	os.Exit(run())
}

func run() int {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		grafanaURL           string
		grafanaToken         string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&grafanaURL, "grafana-url", "", "URL of the managed grafana server")
	flag.StringVar(&grafanaToken, "grafana-token", "", "Auth token for the grafana server")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	logOpts := zap.Options{Development: true}
	logOpts.BindFlags(flag.CommandLine)

	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&logOpts))

	if grafanaURL == "" {
		grafanaURL = os.Getenv("GRAFANA_URL")
	}

	if grafanaToken == "" {
		grafanaToken = os.Getenv("GRAFANA_TOKEN")
	}

	if grafanaURL == "" {
		logger.Info("Please provide a Grafana URL. Exiting.")
		return 1
	}

	if grafanaToken == "" {
		logger.Info("Please provide a Grafana Token. Exiting.")
		return 1
	}

	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		Metrics:                       metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "c2061b9e.dawg.urcloud.cc",
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		logger.Error(err, "unable to start manager")
		return 1
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		return 1
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		return 1
	}

	store, err := generator.DefaultStore()
	if err != nil {
		logger.Error(err, "could not build default generator stores")
		return 1
	}

	runtime, shutdownRuntime, err := generator.DefaultRuntime(context.TODO())
	if err != nil {
		logger.Error(err, "could not setup generator runtime")
		return 1
	}

	defer func() {
		if err := shutdownRuntime(context.Background()); err != nil {
			logger.Error(err, "could not shutdown runtime")
		}
	}()

	grafanaClient := grafana.NewClient(grafanaURL, grafana.WithAuthToken(grafanaToken))

	if err := controller.NewDashboardReconciller(
		store,
		runtime,
		grafanaClient,
	).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to set up the dashboard reconsiller")
		return 1
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		return 1
	}

	return 0
}
