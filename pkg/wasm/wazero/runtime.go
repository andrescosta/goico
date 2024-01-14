package wazero

import (
	"context"
	"errors"
	"os"

	"github.com/tetratelabs/wazero"
)

type WasmRuntime struct {
	cacheDir      *string
	cache         wazero.CompilationCache
	runtimeConfig wazero.RuntimeConfig
}

// Documentation: https://github.com/tetratelabs/wazero/blob/main/examples/multiple-runtimes/counter.go
func NewWasmRuntime(ctx context.Context, tempDir string) (*WasmRuntime, error) {
	wruntime := &WasmRuntime{}
	// Creates a directory to store wazero cache
	if err := os.MkdirAll(tempDir, os.ModeExclusive); err != nil {
		return nil, err
	}
	cacheDir, err := os.MkdirTemp(tempDir, "cache")
	if err != nil {
		return nil, err
	}
	wruntime.cacheDir = &cacheDir
	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		wruntime.Close(ctx)
		return nil, err
	}
	wruntime.runtimeConfig =
		wazero.NewRuntimeConfig().
			WithCompilationCache(cache).
			WithCloseOnContextDone(true)
	wruntime.cache = cache
	return wruntime, nil
}

func (r *WasmRuntime) Close(ctx context.Context) error {
	var errs error
	if err := r.cache.Close(ctx); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := os.RemoveAll(*r.cacheDir); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}
