package wasm

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sysfs"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Part of this code was copied from https://github.com/tetratelabs/wazero/blob/c6a907bb9d6c4605355562859e7cd2fae67fd246/cmd/wazero/wazero.go

type GenericModule struct {
	runtime  wazero.Runtime
	module   wazero.CompiledModule
	conf     wazero.ModuleConfig
	cacheDir *string
	cache    wazero.CompilationCache
}

func NewGenericModule(ctx context.Context, tempDir string, wasmModule []byte, logExt LogFn) (*GenericModule, error) {
	if tempDir == "" {
		return nil, errors.New("directory cannot be empty")
	}
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
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

	rt := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	guest, err := rt.CompileModule(ctx, wasmModule)
	if err != nil {
		return nil, err
	}
	fmt.Println("module compiled")
	switch detectImports(guest.ImportedFunctions()) {
	case modeWasi:
		wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	case modeWasiUnstable:
		wasiBuilder := rt.NewHostModuleBuilder("wasi_unstable")
		wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)
		_, err = wasiBuilder.Instantiate(ctx)
		if err != nil {
			return nil, err
		}
	case modeDefault:
		//
	}
	return &GenericModule{
		cacheDir: &tempDir,
		cache:    cache,
		runtime:  rt,
		module:   guest,
	}, nil
}

func (i *GenericModule) Run(ctx context.Context, mounts []string, args []string, env []EnvVar, in io.Reader, out io.Writer, outErr io.Writer) error {
	_, fsConfig, err := validateMounts(mounts)
	if err != nil {
		return err
	}

	conf := wazero.NewModuleConfig().
		WithRandSource(rand.Reader).
		WithFSConfig(fsConfig).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithArgs(args...).
		WithStdout(out).
		WithStderr(outErr).
		WithStdin(in)
	for _, e := range env {
		conf = conf.WithEnv(e.Key, e.Value)
	}

	mod, err := i.runtime.InstantiateModule(ctx, i.module, conf)
	if err != nil {
		return err
	}
	defer mod.Close(ctx)
	return nil
}

func validateMounts(mounts []string) (rootPath string, config wazero.FSConfig, err error) {
	config = wazero.NewFSConfig()
	for _, mount := range mounts {
		readOnly := false
		if trimmed := strings.TrimSuffix(mount, ":ro"); trimmed != mount {
			mount = trimmed
			readOnly = true
		}
		var dir, guestPath string
		if clnIdx := strings.LastIndexByte(mount, ':'); clnIdx != -1 {
			dir, guestPath = mount[:clnIdx], mount[clnIdx+1:]
		} else {
			dir = mount
			guestPath = dir
		}
		if abs, erra := filepath.Abs(dir); erra != nil {
			err = fmt.Errorf("invalid mount: path %q invalid: %v", dir, erra)
			return
		} else {
			dir = abs
		}

		if stat, errst := os.Stat(dir); errst != nil {
			err = fmt.Errorf("invalid mount: path %q error: %v", dir, errst)
			return
		} else if !stat.IsDir() {
			err = fmt.Errorf("invalid mount: path %q is not a directory", dir)
			return
		}

		root := sysfs.DirFS(dir)
		if readOnly {
			root = &sysfs.ReadFS{FS: root}
		}

		config = config.(sysfs.FSConfig).WithSysFSMount(root, guestPath)

		if stripPrefixesAndTrailingSlash(guestPath) == "" {
			rootPath = dir
		}
	}
	return rootPath, config, nil
}

func stripPrefixesAndTrailingSlash(path string) string {
	pathLen := len(path)
	for ; pathLen > 0 && path[pathLen-1] == '/'; pathLen-- {
	}

	pathI := 0
loop:
	for pathI < pathLen {
		switch path[pathI] {
		case '/':
			pathI++
		case '.':
			nextI := pathI + 1
			if nextI < pathLen && path[nextI] == '/' {
				pathI = nextI + 1
			} else if nextI == pathLen {
				pathI = nextI
			} else {
				break loop
			}
		default:
			break loop
		}
	}
	return path[pathI:pathLen]
}

func detectImports(imports []api.FunctionDefinition) importMode {
	for _, f := range imports {
		moduleName, _, _ := f.Import()
		switch moduleName {
		case wasi_snapshot_preview1.ModuleName:
			return modeWasi
		case "wasi_unstable":
			return modeWasiUnstable
		}
	}
	return modeDefault
}

func (r *GenericModule) Close(ctx context.Context) error {
	var errs error
	if r.module != nil {
		errs = errors.Join(errs, r.module.Close(ctx))
	}
	if r.cache != nil {
		errs = errors.Join(errs, r.cache.Close(ctx))
	}
	if r.cacheDir != nil {
		errs = errors.Join(errs, os.RemoveAll(*r.cacheDir))
	}
	return errs
}
