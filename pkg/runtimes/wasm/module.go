package wasm

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type (
	ModuleType uint32
	LogFn      func(context.Context, uint32, uint32, string) error
)

type Module struct {
	mainFunc   api.Function
	initFunc   api.Function
	mallocFunc api.Function
	freeFunc   api.Function
	logFn      LogFn
	freeFn     func(context.Context, uint64, uint64) ([]uint64, error)
	module     api.Module
	ver        ModuleType
}

type EventFuncResult struct {
	Errno         uint64
	StrPtrEncoded uint64
}

const (
	TypeDefault ModuleType = iota
	TypeRust
)

func NewModule(ctx context.Context, runtime *Runtime, wasmModule []byte, mainFuncName string, logExt LogFn) (*Module, error) {
	wm := &Module{
		logFn: logExt,
	}

	wazeroRuntime := wazero.NewRuntimeWithConfig(ctx, runtime.runtimeConfig)

	// DON'T MOVE IT.
	_, err := wazeroRuntime.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(wm.log).Export("log").
		Instantiate(ctx)
	if err != nil {
		return nil, err
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, wazeroRuntime)
	module, err := wazeroRuntime.Instantiate(ctx, wasmModule)
	if err != nil {
		return nil, err
	}
	ver := TypeDefault
	verFunc := module.ExportedFunction("ver")
	if verFunc != nil {
		v, err := call(ctx, verFunc)
		if err == nil {
			ver = ModuleType(v[0])
		}
	}
	initf := module.ExportedFunction("init")
	wm.mainFunc = module.ExportedFunction(mainFuncName)
	wm.initFunc = initf
	// for tinygo: tinygo-org/tinygo#2788
	wm.mallocFunc = module.ExportedFunction("malloc")
	wm.freeFunc = module.ExportedFunction("free")
	wm.module = module
	wm.ver = ver

	wm.freeFn = wm.free
	// Call the init function to initialize the module
	_, err = call(ctx, initf)
	if err != nil {
		return nil, err
	}
	return wm, nil
}

func (f *Module) free(ctx context.Context, offset, size uint64) ([]uint64, error) {
	if f.ver == TypeDefault {
		return call(ctx, f.freeFunc, offset)
	}
	return call(ctx, f.freeFunc, offset, size)
}

func (f *Module) Run(ctx context.Context, id uint32, data string) (uint64, string, error) {
	logger := zerolog.Ctx(ctx)
	// reserve memory for the string parameter
	strParamOffset, strParamSize, err := f.writeToMemory(ctx, data)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_, err := f.freeFn(ctx, strParamOffset, strParamSize)
		if err != nil {
			logger.Warn().AnErr("err", err)
		}
	}()
	resultFuncPtr, resultFuncSize, err := f.reserveMemoryForResult(ctx)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_, err := f.freeFn(ctx, resultFuncPtr, resultFuncSize)
		if err != nil {
			logger.Warn().AnErr("err", err)
		}
	}()
	logger.Debug().Msg("calling main method")
	// The result of the call will be stored in struct pointed by resultFuncPtr
	_, err = call(ctx, f.mainFunc, resultFuncPtr, api.EncodeU32(id), strParamOffset, strParamSize)
	if err != nil {
		return 0, "", err
	}
	errno, res, err := f.getResult(ctx, resultFuncPtr, resultFuncSize)
	if err != nil {
		return 0, "", err
	}
	return errno, res, nil
}

func (f *Module) reserveMemoryForResult(ctx context.Context) (uint64, uint64, error) {
	eventDataSize := uint64(unsafe.Sizeof(EventFuncResult{}))
	results, err := call(ctx, f.mallocFunc, eventDataSize)
	if err != nil {
		return 0, 0, err
	}
	eventDataPtr := results[0]
	return eventDataPtr, eventDataSize, nil
}

func (f *Module) writeToMemory(ctx context.Context, data string) (uint64, uint64, error) {
	size := uint64(len(data))
	results, err := call(ctx, f.mallocFunc, size)
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

func (f *Module) getResult(ctx context.Context, offset uint64, size uint64) (uint64, string, error) {
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

func (f *Module) getResultStr(ctx context.Context, encodedPtr uint64) (string, error) {
	logger := zerolog.Ctx(ctx)
	offset := uint32(encodedPtr >> 32)
	size := uint32(encodedPtr)
	if offset != 0 {
		defer func() {
			_, err := f.freeFn(ctx, uint64(offset), uint64(size))
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

func (f *Module) log(ctx context.Context, m api.Module, id, level, offset, byteCount uint32) {
	logger := zerolog.Ctx(ctx)
	buf, ok := m.Memory().Read(offset, byteCount)
	if !ok {
		logger.Error().Msgf("Memory.Read(%d, %d) out of range", offset, byteCount)
	}
	msg := string(buf)
	logger.WithLevel(zerolog.Level(level)).Msg(msg)
	if f.logFn != nil {
		if err := f.logFn(ctx, id, level, msg); err != nil {
			logger.Err(err).Msg("error executing log function.")
		}
	}
}

func (f *Module) Close(ctx context.Context) error {
	if err := f.module.Close(ctx); err != nil {
		return err
	}
	return nil
}

func call(ctx context.Context, f api.Function, params ...uint64) ([]uint64, error) {
	return f.Call(ctx, params...)
}
