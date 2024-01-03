package wazero

import (
	"context"
	"os"

	"github.com/rs/zerolog"
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
	wruntime.cache = cache
	runtimeConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)
	wruntime.runtimeConfig = runtimeConfig
	wruntime.runtimeConfig.WithCloseOnContextDone(true)
	return wruntime, nil
}

func (r *WasmRuntime) Close(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	if r.cacheDir != nil {
		if err := os.RemoveAll(*r.cacheDir); err != nil {
			logger.Err(err).Msg("error deleting cache files.")
		}
	}
	if r.cache != nil {
		if err := r.cache.Close(ctx); err != nil {
			logger.Err(err).Msg("error closing wasm runtime.")
		}
	}
}
