package testutil

import (
	"context"
	"path/filepath"
	"testing"

	dawgv1 "github.com/jlevesy/dawg/api/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

type SetupableWithManager interface {
	SetupWithManager(m manager.Manager) error
}

func NewTestingManager(t *testing.T, restConfig *rest.Config, reconcilers ...SetupableWithManager) manager.Manager {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, dawgv1.AddToScheme(scheme))

	_, err := envtest.InstallCRDs(restConfig, envtest.CRDInstallOptions{
		Paths:              []string{filepath.Join("..", "..", "k8s", "crd")},
		ErrorIfPathMissing: true,
		Scheme:             scheme,
	})
	require.NoError(t, err)

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			// This disables the metric server.
			BindAddress: "0",
		},
	})
	require.NoError(t, err)

	for _, reconciler := range reconcilers {
		err := reconciler.SetupWithManager(mgr)
		require.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		err := mgr.Start(ctx)
		require.NoError(t, err)
	}()

	// Subtle hack that allows us to start testing when the manager is ready!
	<-mgr.Elected()

	return mgr
}
