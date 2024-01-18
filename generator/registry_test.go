package generator_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/jlevesy/dawg/generator"
	"github.com/jlevesy/dawg/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Registry(t *testing.T) {
	var (
		ctx = context.Background()
		gen = generator.Generator{
			Bin: []byte("coucou"),
		}
	)

	ts := testutil.RunContainer(t, testutil.RegistryContainerConfig)
	t.Cleanup(func() {
		require.NoError(t, ts.Shutdown(context.Background()))
	})

	genStore, err := generator.DefaultStore()
	require.NoError(t, err)

	genUrl, err := url.Parse("registry://localhost:" + ts.Port + "/testgenerators/test:v0.0.1")
	require.NoError(t, err)

	err = genStore.Store(ctx, genUrl, &gen)
	require.NoError(t, err)

	gotGen, err := genStore.Load(ctx, genUrl)
	require.NoError(t, err)

	assert.Equal(t, &gen, gotGen)
}
