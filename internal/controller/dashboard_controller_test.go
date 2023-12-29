package controller_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/jlevesy/dawg/generator"
	"github.com/jlevesy/dawg/internal/controller"
	"github.com/jlevesy/dawg/pkg/testutil"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

var (
	//go:generate tinygo build -o ./testdata/v1.wasm -scheduler=none --no-debug -target wasi ./testdata/v1
	//go:embed testdata/v1.wasm
	v1Bin []byte

	//go:generate tinygo build -o ./testdata/v2.wasm -scheduler=none --no-debug -target wasi ./testdata/v2
	//go:embed testdata/v2.wasm
	v2Bin []byte
)

func TestDashboardController(t *testing.T) {
	ctx := context.Background()

	registry := testutil.RunContainer(t, testutil.RegistryContainerConfig)
	t.Cleanup(func() {
		require.NoError(t, registry.Shutdown(context.Background()))
	})

	genStore, err := generator.DefaultStore()
	require.NoError(t, err)

	v1URL, v2URL := pushGenerators(t, genStore, registry.Port)

	k8sCluster := testutil.RunContainer(t, testutil.KWOKContainerConfig)
	t.Cleanup(func() {
		require.NoError(t, k8sCluster.Shutdown(ctx))
	})

	mgr := testutil.NewTestingManager(
		t,
		&rest.Config{Host: "http://localhost:" + k8sCluster.Port},
		controller.NewDashboardReconciller(),
	)

	client := mgr.GetClient()

}

func pushGenerators(t *testing.T, store generator.Store, registryPort string) (string, string) {
	t.Helper()

	ctx := context.Background()

	v1URL, err := url.Parse("registry://localhost:" + registryPort + "/testgenerators/test:v1")
	require.NoError(t, err)

	v2URL, err := url.Parse("registry://localhost:" + registryPort + "/testgenerators/test:v2")
	require.NoError(t, err)

	err = store.Store(ctx, v1URL, &generator.Generator{Bin: v1Bin})
	require.NoError(t, err)

	err = store.Store(ctx, v2URL, &generator.Generator{Bin: v2Bin})
	require.NoError(t, err)

	return v1URL.String(), v2URL.String()
}
