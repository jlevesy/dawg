package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jlevesy/dawg/gdk"
	"github.com/liamg/memoryfs"
	"github.com/tetratelabs/wazero"
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

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, wasmRuntime); err != nil {
		return nil, nil, err
	}

	return &runtime{
		wasm:           wasmRuntime,
		executeTimeout: time.Second,
	}, wasmRuntime.Close, nil
}

type runtime struct {
	wasm           wazero.Runtime
	executeTimeout time.Duration
}

func (r *runtime) Execute(ctx context.Context, gen *Generator, payload []byte) (*ExecutionResult, error) {
	fs := memoryfs.New()

	if err := fs.WriteFile(filepath.Base(gdk.InputPath), payload, 0o600); err != nil {
		return nil, fmt.Errorf("could not write config file %w", err)
	}

	mod, err := r.wasm.InstantiateWithConfig(
		ctx,
		gen.Bin,
		wazero.NewModuleConfig().WithFSConfig(
			wazero.NewFSConfig().WithFSMount(fs, "/dawg"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate generator module: %w", err)
	}

	defer func() {
		// TODO(jly): Log?
		_ = mod.Close(ctx)
	}()

	fn := mod.ExportedFunction("generate")
	if fn == nil {
		return nil, unexportedSymbolError("generate")
	}

	callCtx, cancel := context.WithTimeout(ctx, r.executeTimeout)
	defer cancel()

	fnResult, err := fn.Call(callCtx)
	if err != nil {
		return nil, fmt.Errorf("call to function generate reported  an error: %w", err)
	}

	// This limits the payload size to 4Gb.
	resultBufPtr, resultBufSize := uint32(fnResult[0]>>32), uint32(fnResult[0])

	resultBuf, ok := mod.Memory().Read(resultBufPtr, resultBufSize)
	if !ok {
		return nil, errFailedToReadMemory
	}

	// This is a very dumb and naive way of reporing an error.
	// But at least comporate the first byte efficiently, do not try to json unmarshal every result.
	// TODO(jly): do better.
	if len(resultBuf) > 0 && resultBuf[0] == 'e' {
		var gdkErr gdk.RuntimeError

		if err := json.Unmarshal(resultBuf[1:], &gdkErr); err != nil {
			return nil, fmt.Errorf("generator reported an error but we could not deserialize it: %w", err)
		}

		return nil, errors.New(gdkErr.Err)
	}

	return &ExecutionResult{
		Payload: resultBuf,
	}, nil
}

var (
	errFailedToReadMemory = errors.New("could not read to the module memory")
)

type unexportedSymbolError string

func (u unexportedSymbolError) Error() string {
	return fmt.Sprintf("module does not export the function %q", string(u))
}
