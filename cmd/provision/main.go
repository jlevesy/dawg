package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jlevesy/dawg/pkg/grafana"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		generatorPath string
		configPath    string
		grafanaURL    string
		grafanaToken  string
	)

	flag.StringVar(&generatorPath, "generator", "", "Path to the WASM binary of the generator")
	flag.StringVar(&configPath, "config", "", "Path to the config of the generator")
	flag.StringVar(&grafanaURL, "grafana-url", "", "URL of the grafana instance to provision")
	flag.StringVar(&grafanaToken, "grafana-token", "", "API token to use with the grafana instance")
	flag.Parse()

	if generatorPath == "" || configPath == "" {
		fmt.Println("Must provide a generator path and the config path")
		return 1
	}

	if grafanaURL == "" {
		fmt.Println("Must provide a grafana URL")
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	genBin, err := os.ReadFile(generatorPath)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	wasmRuntime := wazero.NewRuntime(ctx)
	defer wasmRuntime.Close(context.Background())

	wasi_snapshot_preview1.MustInstantiate(ctx, wasmRuntime)

	mod, err := wasmRuntime.Instantiate(ctx, genBin)
	if err != nil {
		fmt.Println("could not instanciate binary", err)
		return 1
	}

	dashboardPayload, err := call(ctx, mod, "generate", configBytes)
	if err != nil {
		fmt.Println("could not call generate", err)
		return 1
	}

	var grafanaOpts []grafana.ClientOpt

	if grafanaToken != "" {
		grafanaOpts = append(grafanaOpts, grafana.WithAuthToken(grafanaToken))
	}

	grafanaClient := grafana.NewClient(grafanaURL, grafanaOpts...)

	dashboard, err := grafanaClient.CreateDashboard(
		ctx,
		&grafana.CreateDashboardRequest{
			Dashboard: dashboardPayload,
			Overwrite: true,
		},
	)
	if err != nil {
		fmt.Println("could not create dashboard", err)
		return 1
	}

	fmt.Println("Created dashboard", dashboard)

	return 0
}

var (
	errFailedToWriteToMemory = errors.New("could not write to the module memory")
	errFailedToReadMemory    = errors.New("could not read to the module memory")
)

type unexportedSymbolError string

func (u unexportedSymbolError) Error() string {
	return fmt.Sprintf("module does not export the function %q", string(u))
}

func call(ctx context.Context, mod api.Module, funcName string, arg []byte) ([]byte, error) {
	var (
		malloc = mod.ExportedFunction("malloc")
		free   = mod.ExportedFunction("free")
		fn     = mod.ExportedFunction(funcName)
	)

	switch {
	case malloc == nil:
		return nil, unexportedSymbolError("malloc")
	case free == nil:
		return nil, unexportedSymbolError("free")
	case fn == nil:
		return nil, unexportedSymbolError(funcName)
	}

	mallocResult, err := malloc.Call(ctx, uint64(len(arg)))
	if err != nil {
		return nil, fmt.Errorf("could not allocate memory: %w", err)
	}

	bufPtr := mallocResult[0]

	defer free.Call(ctx, bufPtr)

	if !mod.Memory().Write(uint32(bufPtr), arg) {
		return nil, errFailedToWriteToMemory
	}

	fnResult, err := fn.Call(ctx, bufPtr, uint64(len(arg)))
	if err != nil {
		return nil, fmt.Errorf("call to function %q reported  an error: %w", funcName, err)
	}

	// this limits the payload size to 4Gb.
	resultBufPtr, resultBufSize := uint32(fnResult[0]>>32), uint32(fnResult[0])

	resultBuf, ok := mod.Memory().Read(resultBufPtr, resultBufSize)
	if !ok {
		return nil, errFailedToReadMemory
	}

	return resultBuf, nil
}
