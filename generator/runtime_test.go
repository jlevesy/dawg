package generator_test

import (
	"context"
	_ "embed"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlevesy/dawg/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate tinygo build -o ./testdata/runtime/goodtinygo.wasm -scheduler=none --no-debug -target wasi ./testdata/runtime/good
//go:embed testdata/runtime/goodtinygo.wasm
var goodTinygo []byte

func TestRuntime_Good(t *testing.T) {
	var payload = []byte(`{"some":"config"}`)

	result, err := runWasm(t, goodTinygo, payload)
	require.NoError(t, err)

	assert.Equal(t, payload, result.Payload)
}

//go:generate env GOOS=wasip1 GOARCH=wasm go build -o ./testdata/runtime/goodgo121.wasm ./testdata/runtime/good
//go:embed testdata/runtime/goodgo121.wasm
var goodGo121Bin []byte

func TestRuntime_GoodWithGo121(t *testing.T) {
	// See https://github.com/golang/go/issues/42372#issuecomment-1837330447
	t.Skip("go 121 does not allow module exports, this doesn't work")
	var payload = []byte(`{"some":"config"}`)

	result, err := runWasm(t, goodGo121Bin, payload)
	require.NoError(t, err)

	assert.Equal(t, payload, result.Payload)
}

//go:generate tinygo build -o ./testdata/runtime/panic.wasm -scheduler=none --no-debug -target wasi ./testdata/runtime/panic
//go:embed testdata/runtime/panic.wasm
var panicBin []byte

func TestRuntime_ReportPanic(t *testing.T) {
	_, err := runWasm(t, panicBin, nil)
	assert.Error(t, err)
}

//go:generate tinygo build -o ./testdata/runtime/nofsread.wasm -scheduler=none --no-debug -target wasi ./testdata/runtime/nofsread
//go:embed testdata/runtime/nofsread.wasm
var noFSReadBin []byte

func TestRuntime_NoFSRead(t *testing.T) {
	var (
		filePath = filepath.Join(t.TempDir(), "secret")
	)

	err := os.WriteFile(filePath, []byte("sensitive info"), 0777)
	require.NoError(t, err)

	res, err := runWasm(t, noFSReadBin, []byte(filePath))
	assert.Error(t, err)
	assert.Empty(t, res, "Returned result should have been empty")
}

//go:generate tinygo build -o ./testdata/runtime/nofswrite.wasm -scheduler=none --no-debug -target wasi ./testdata/runtime/nofswrite
//go:embed testdata/runtime/nofswrite.wasm
var noFSWriteBin []byte

func TestRuntime_NoFSWrite(t *testing.T) {
	_, err := runWasm(t, noFSWriteBin, nil)
	assert.Error(t, err)

	_, err = os.Open("./hacked")
	assert.Error(t, err, "File has been written to disk by WASM binary")
}

//go:generate tinygo build -o ./testdata/runtime/noenvread.wasm -scheduler=none --no-debug -target wasi ./testdata/runtime/noenvread
//go:embed testdata/runtime/noenvread.wasm
var noEnvReadBin []byte

func TestRuntime_NoEnvRead(t *testing.T) {
	err := os.Setenv("DAWG_TEST_SUITE_SECRET", "supersecret")
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Unsetenv("DAWG_TEST_SUITE_SECRET")
		require.NoError(t, err)
	})

	result, err := runWasm(t, noEnvReadBin, nil)
	assert.Error(t, err)
	assert.Empty(t, result)
}

//go:generate env GOOS=wasip1 GOARCH=wasm go build -o ./testdata/runtime/nonet.wasm ./testdata/runtime/nonet
//go:embed testdata/runtime/nonet.wasm
var noNetBin []byte

func TestRuntime_NoNetwork(t *testing.T) {
	var (
		called       int
		listenerDone = make(chan struct{})
	)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				close(listenerDone)
				return
			}

			called++

			_, err = conn.Write([]byte("pwned"))
			require.NoError(t, err)

			err = conn.Close()
			require.NoError(t, err)
		}
	}()

	t.Cleanup(func() {
		err := listener.Close()
		require.NoError(t, err)
		<-listenerDone
	})

	result, err := runWasm(t, noNetBin, []byte(listener.Addr().String()))
	t.Log("error is", err)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, called)
}

//go:generate env GOOS=wasip1 GOARCH=wasm go build -o ./testdata/runtime/hang.wasm ./testdata/runtime/hang
//go:embed testdata/runtime/hang.wasm
var hangBin []byte

func TestRuntime_Hang(t *testing.T) {
	_, err := runWasm(t, hangBin, nil)
	assert.Error(t, err)
}

func runWasm(t *testing.T, bin, args []byte) (*generator.ExecutionResult, error) {
	t.Helper()

	ctx := context.Background()
	runtime, shutdown, err := generator.DefaultRuntime(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := shutdown(ctx)
		require.NoError(t, err)
	})

	return runtime.Execute(ctx, &generator.Generator{Bin: bin}, args)
}
