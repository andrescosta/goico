package wasm

import (
	"context"
	"crypto/rand"
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

type IntModule struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
	conf    wazero.ModuleConfig
}

func (i *IntModule) Run(ctx context.Context) error {
	_, err := i.runtime.InstantiateModule(ctx, i.module, i.conf)
	if err != nil {
		return err
	}
	return nil
}

func (i *IntModule) Close(ctx context.Context) error {
	return i.module.Close(ctx)
}

func NewIntModule(ctx context.Context, runtime *Runtime, wasmModule []byte, logExt LogFn, mounts []string, args []string, env []EnvVar, in io.Reader, out io.Writer, outErr io.Writer) (*IntModule, error) {
	_, fsConfig, err := validateMounts(mounts)
	if err != nil {
		return nil, err
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
	rt := wazero.NewRuntimeWithConfig(ctx, runtime.runtimeConfig)
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
		_, err = rt.InstantiateModule(ctx, guest, conf)
		if err != nil {
			return nil, err
		}
	}
	return &IntModule{
		conf:    conf,
		runtime: rt,
		module:  guest,
	}, nil
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

		if StripPrefixesAndTrailingSlash(guestPath) == "" {
			rootPath = dir
		}
	}
	return rootPath, config, nil
}

func StripPrefixesAndTrailingSlash(path string) string {
	// strip trailing slashes
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
