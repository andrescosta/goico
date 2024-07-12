package wasm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type (
	ModuleType uint32
	LogFn      func(context.Context, uint32, string) error
)

type JobicoletModule struct {
	mainFuncName string
	mod          wazero.CompiledModule
	rt           wazero.Runtime
	cacheDir     string
	cache        wazero.CompilationCache
	logFn        LogFn
}

type activeModule struct {
	module   api.Module
	ver      ModuleType
	mainFn   api.Function
	initFn   api.Function
	mallocFn api.Function
	freeFn   api.Function
}

type EventFuncResult struct {
	Errno         uint64
	StrPtrEncoded uint64
}

type importMode uint

const (
	modeDefault importMode = iota
	modeWasi
	modeWasiUnstable
)

const (
	TypeDefault ModuleType = iota
	TypeRust
)

type EnvVar struct {
	Key   string
	Value string
}

func NewJobicoletModule(ctx context.Context, tempDir string, wasmModule []byte, mainFuncName string, logExt LogFn) (*JobicoletModule, error) {
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
	module, err := rt.CompileModule(ctx, wasmModule)
	if err != nil {
		return nil, err
	}

	mod := &JobicoletModule{
		rt:           rt,
		mainFuncName: mainFuncName,
		logFn:        logExt,
		cacheDir:     cacheDir,
		cache:        cache,
		mod:          module,
	}

	switch detectImports(module.ImportedFunctions()) {
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

	_, err = rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(mod.log).Export("log").
		Instantiate(ctx)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func (f *JobicoletModule) Run(ctx context.Context, data string) (uint64, string, error) {
	logger := zerolog.Ctx(ctx)

	// active module preparation
	activeModule := activeModule{}
	conf := wazero.NewModuleConfig().
		WithRandSource(rand.Reader).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()

	module, err := f.rt.InstantiateModule(ctx, f.mod, conf)
	if err != nil {
		return 0, "", err
	}
	defer module.Close(ctx)
	activeModule.module = module
	addExportedFunctions(ctx, f.mainFuncName, &activeModule)

	// Starts the execution
	// Call the init function to initialize the module
	_, err = activeModule.initFn.Call(ctx)
	if err != nil {
		return 0, "", nil
	}

	// passes parameters
	strParamOffset, strParamSize, err := activeModule.writeToMemory(ctx, data)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_, err := activeModule.free(ctx, strParamOffset, strParamSize)
		if err != nil {
			logger.Warn().AnErr("err", err)
		}
	}()

	// prepares memory for results
	resultFuncPtr, resultFuncSize, err := activeModule.reserveMemoryForResult(ctx)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_, err := activeModule.free(ctx, resultFuncPtr, resultFuncSize)
		if err != nil {
			logger.Warn().AnErr("err", err)
		}
	}()
	logger.Debug().Msg("calling main method")

	// Calls the main Jobicolet func
	_, err = activeModule.mainFn.Call(ctx, resultFuncPtr, strParamOffset, strParamSize)
	if err != nil {
		return 0, "", err
	}

	// reads the results
	errno, res, err := activeModule.getResult(ctx, resultFuncPtr, resultFuncSize)
	if err != nil {
		return 0, "", err
	}
	return errno, res, nil
}

func (f *activeModule) free(ctx context.Context, offset, size uint64) ([]uint64, error) {
	if f.ver == TypeDefault {
		return f.freeFn.Call(ctx, offset)
	}
	return f.freeFn.Call(ctx, offset, size)
}

func addExportedFunctions(ctx context.Context, mainFuncName string, a *activeModule) {
	a.ver = TypeDefault
	verFunc := a.module.ExportedFunction("ver")
	if verFunc != nil {
		v, err := verFunc.Call(ctx)
		if err == nil {
			a.ver = ModuleType(v[0])
		}
	}
	a.mainFn = a.module.ExportedFunction(mainFuncName)
	a.initFn = a.module.ExportedFunction("init")
	a.mallocFn = a.module.ExportedFunction("malloc")
	a.freeFn = a.module.ExportedFunction("free")
}

func (f *activeModule) reserveMemoryForResult(ctx context.Context) (uint64, uint64, error) {
	eventDataSize := uint64(unsafe.Sizeof(EventFuncResult{}))
	results, err := f.mallocFn.Call(ctx, eventDataSize)
	if err != nil {
		return 0, 0, err
	}
	eventDataPtr := results[0]
	return eventDataPtr, eventDataSize, nil
}

func (f *activeModule) writeToMemory(ctx context.Context, data string) (uint64, uint64, error) {
	size := uint64(len(data))
	results, err := f.mallocFn.Call(ctx, size)
	if err != nil {
		return 0, 0, err
	}
	offset := results[0]
	if !f.module.Memory().Write(uint32(offset), []byte(data)) {
		return 0, 0, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			offset, size, f.module.Memory().Size())
	}
	return offset, size, nil
}

func (f *activeModule) getResult(ctx context.Context, offset uint64, size uint64) (uint64, string, error) {
	if data, ok := f.module.Memory().Read(uint32(offset), uint32(size)); ok {
		var result EventFuncResult
		err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &result)
		if err != nil {
			return 0, "", err
		}
		resultStr, err := f.getResultStr(ctx, result.StrPtrEncoded)
		if err != nil {
			return 0, "", err
		}
		return result.Errno, resultStr, nil
	}
	return 0, "", fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
		offset, size, f.module.Memory().Size())
}

func (f *activeModule) getResultStr(ctx context.Context, encodedPtr uint64) (string, error) {
	logger := zerolog.Ctx(ctx)
	offset := uint32(encodedPtr >> 32)
	size := uint32(encodedPtr)
	if offset != 0 {
		defer func() {
			_, err := f.free(ctx, uint64(offset), uint64(size))
			if err != nil {
				logger.Err(err).Msg("error freeing memory")
			}
		}()
	}
	bytes, ok := f.module.Memory().Read(offset, size)
	if !ok {
		return "", fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			offset, size, f.module.Memory().Size())
	}
	return string(bytes), nil
}

func (f *JobicoletModule) log(ctx context.Context, module api.Module, level, offset, byteCount uint32) {
	logger := zerolog.Ctx(ctx)
	buf, ok := module.Memory().Read(offset, byteCount)
	if !ok {
		logger.Error().Msgf("Memory.Read(%d, %d) out of range", offset, byteCount)
	}
	msg := string(buf)
	logger.WithLevel(zerolog.Level(level)).Msg(msg)
	if f.logFn != nil {
		if err := f.logFn(ctx, level, msg); err != nil {
			logger.Err(err).Msg("error executing log function.")
		}
	}
}

func (r *JobicoletModule) Close(ctx context.Context) error {
	var errs error
	errs = errors.Join(errs, r.cache.Close(ctx))
	errs = errors.Join(errs, os.RemoveAll(r.cacheDir))
	return errs
}
