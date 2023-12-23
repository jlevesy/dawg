package generator

import (
	"context"
	"errors"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type ExecutionResult struct {
	Payload []byte
}

// Runtime represents any implementation that could execute a given generator with a given payload and retrieve results.
type Runtime interface {
	Execute(ctx context.Context, gen *Generator, payload []byte) (*ExecutionResult, error)
}

func DefaultRuntime(ctx context.Context) (Runtime, func(context.Context) error, error) {
	// TODO(jly): compilation cache?
	wasmRuntime := wazero.NewRuntime(ctx)

	// TODO(jly): this is exposing way too many capabilities to the binary
	// We should find a way to only instanciate what we need, or unexport stuff.
	_, err := wasi_snapshot_preview1.Instantiate(ctx, wasmRuntime)
	if err != nil {
		return nil, nil, err
	}

	return &runtime{
		wasm: wasmRuntime,
		// TODO(jly): support other ways to call wasm based on Generator metadata?
		callWASM: tinygoCall,
	}, wasmRuntime.Close, nil
}

// callProtocol represents a way of calling a wasm module function with arbitrary data.
type callProtocol func(ctx context.Context, wasmModule api.Module, funcName string, payload []byte) ([]byte, error)

// runtime executes WASM binaries built with tinygo (the only one we support right now :shrug:
type runtime struct {
	wasm     wazero.Runtime
	callWASM callProtocol
}

func (r *runtime) Execute(ctx context.Context, gen *Generator, payload []byte) (*ExecutionResult, error) {
	mod, err := r.wasm.Instantiate(ctx, gen.Bin)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate generator module: %w", err)
	}

	defer mod.Close(ctx)

	resultBytes, err := r.callWASM(ctx, mod, "generate", payload)
	if err != nil {
		return nil, fmt.Errorf("could not execute generate: %w", err)
	}

	return &ExecutionResult{
		Payload: resultBytes,
	}, nil
}

var (
	errFailedToWriteToMemory = errors.New("could not write to the module memory")
	errFailedToReadMemory    = errors.New("could not read to the module memory")
)

// tinygoCall allows to call a generator function on a wasm module built with tinygo.
func tinygoCall(ctx context.Context, mod api.Module, funcName string, arg []byte) (resultBuf []byte, err error) {
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

	defer func() {
		_, err = free.Call(ctx, bufPtr)
	}()

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

type unexportedSymbolError string

func (u unexportedSymbolError) Error() string {
	return fmt.Sprintf("module does not export the function %q", string(u))
}
