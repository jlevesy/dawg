package generator_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/jlevesy/dawg/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate tinygo build -o ./testdata/runtime/good.wasm -scheduler=none --no-debug -target wasi ./testdata/runtime/good
//go:embed testdata/runtime/good.wasm
var goodBin []byte

func TestRuntimeGood(t *testing.T) {
	ctx := context.Background()
	runtime, shutdown, err := generator.DefaultRuntime(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := shutdown(ctx)
		require.NoError(t, err)
	})

	result, err := runtime.Execute(ctx, &generator.Generator{Bin: goodBin}, nil)
	require.NoError(t, err)

	assert.Equal(t, []byte(`{"some":"config"}`), result.Payload)
}
