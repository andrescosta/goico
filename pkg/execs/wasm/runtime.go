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

func NewRuntimeWithCompilationCache(tempDir string) (*Runtime, error) {
	if tempDir == "" {
		return nil, errors.New("directory cannot be empty")
	}
	if err := os.MkdirAll(tempDir, os.ModeExclusive); err != nil {
		return nil, err
	}
	cacheDir, err := os.MkdirTemp(tempDir, "cache")
	if err != nil {
		return nil, err
	}
	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		err := os.RemoveAll(cacheDir)
		return nil, err
	}
	runtimeConfig := wazero.NewRuntimeConfig().
		WithCompilationCache(cache).
		WithCloseOnContextDone(true)
	return &Runtime{
		cacheDir:      &cacheDir,
		cache:         cache,
		runtimeConfig: runtimeConfig,
	}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	var errs error
	if r.cache != nil {
		if err := r.cache.Close(ctx); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if r.cacheDir != nil {
		if err := os.RemoveAll(*r.cacheDir); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
