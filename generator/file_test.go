package generator_test

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/jlevesy/dawg/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Filesystem(t *testing.T) {
	var (
		ctx     = context.Background()
		workDir = t.TempDir()
		gen     = generator.Generator{
			Bin: []byte("coucou"),
		}
	)

	genStore, err := generator.DefaultStore()
	require.NoError(t, err)

	genUrl, err := url.Parse("file://" + filepath.Join(workDir, "test.wasm"))
	require.NoError(t, err)

	err = genStore.Store(ctx, genUrl, &gen)
	require.NoError(t, err)

	gotGen, err := genStore.Load(ctx, genUrl)
	require.NoError(t, err)

	assert.Equal(t, &gen, gotGen)
}
