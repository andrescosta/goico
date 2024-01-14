package wasm

import (
	"context"
	"errors"
	"os"

	"github.com/tetratelabs/wazero"
)

type Runtime struct {
	cacheDir      *string
	cache         wazero.CompilationCache
	runtimeConfig wazero.RuntimeConfig
}

func NewRuntime(tempDir string) (*Runtime, error) {
	wruntime := &Runtime{}
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
		err := os.RemoveAll(cacheDir)
		return nil, err
	}
	wruntime.runtimeConfig = wazero.NewRuntimeConfig().
		WithCompilationCache(cache).
		WithCloseOnContextDone(true)
	wruntime.cache = cache
	return wruntime, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	var errs error
	if err := r.cache.Close(ctx); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := os.RemoveAll(*r.cacheDir); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}
