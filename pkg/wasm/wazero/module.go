package wazero

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
	LogExt     func(context.Context, uint32, uint32, string) error
)

type WasmModuleString struct {
	mainFunc   api.Function
	initFunc   api.Function
	mallocFunc api.Function
	freeFunc   api.Function
	logExt     LogExt
	FreeAdpter func(context.Context, uint64, uint64) ([]uint64, error)
	module     api.Module
	ver        ModuleType
}

type EventFuncResult struct {
	ResponseCode  uint64
	ResultPtrSize uint64
}

const (
	TypeDefault ModuleType = iota
	TypeRust
)

func NewWasmModuleString(ctx context.Context, runtime *WasmRuntime, wasmModule []byte, mainFuncName string, logExt LogExt) (*WasmModuleString, error) {
	wm := &WasmModuleString{
		logExt: logExt,
	}

	wazeroRuntime := wazero.NewRuntimeWithConfig(ctx, runtime.runtimeConfig)

	// env is used by the module name used by the SDKs.
	// TODO: change it to something more appropriate like sdk
	// This call instantiate an ENV. DON'T MOVE IT.
	_, err := wazeroRuntime.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(wm.logForExport).Export("log").
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
		v, err := verFunc.Call(ctx)
		if err == nil {
			ver = ModuleType(v[0])
		}
	}
	// TODO: Replace init with _start or _init
	initf := module.ExportedFunction("init")
	wm.mainFunc = module.ExportedFunction(mainFuncName)
	wm.initFunc = initf
	// These are undocumented, but exported. See tinygo-org/tinygo#2788
	wm.mallocFunc = module.ExportedFunction("malloc")
	wm.freeFunc = module.ExportedFunction("free")
	wm.module = module
	wm.ver = ver

	wm.FreeAdpter = wm.free
	// Call the init function to initialize the module
	_, err = initf.Call(ctx)
	if err != nil {
		return nil, err
	}
	return wm, nil
}

func (f *WasmModuleString) free(ctx context.Context, ptr, size uint64) ([]uint64, error) {
	if f.ver == TypeDefault {
		return f.freeFunc.Call(ctx, ptr)
	}
	return f.freeFunc.Call(ctx, ptr, size)
}

func (f *WasmModuleString) ExecuteMainFunc(ctx context.Context, id uint32, data string) (uint64, string, error) {
	logger := zerolog.Ctx(ctx)
	// reserve memory for the string parameter
	funcParameterPtr, funcParameterSize, err := f.writeParameterToMemory(ctx, data)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_, err := f.FreeAdpter(ctx, funcParameterPtr, funcParameterSize)
		if err != nil {
			logger.Warn().AnErr("err", err)
		}
	}()
	resultFuncPtr, resultFuncSize, err := f.reserveMemoryForResult(ctx)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_, err := f.FreeAdpter(ctx, resultFuncPtr, resultFuncSize)
		if err != nil {
			logger.Warn().AnErr("err", err)
		}
	}()
	logger.Debug().Msg("calling main method")
	// The result of the call will be stored in struct pointed by resultFuncPtr
	_, err = f.mainFunc.Call(ctx, resultFuncPtr, api.EncodeU32(id), funcParameterPtr, funcParameterSize)
	if err != nil {
		return 0, "", err
	}
	code, res, err := f.readResultFromMemory(ctx, resultFuncPtr, resultFuncSize)
	if err != nil {
		return 0, "", err
	}
	return code, res, nil
}

func (f *WasmModuleString) reserveMemoryForResult(ctx context.Context) (uint64, uint64, error) {
	eventDataSize := uint64(unsafe.Sizeof(EventFuncResult{}))
	results, err := f.mallocFunc.Call(ctx, eventDataSize)
	if err != nil {
		return 0, 0, err
	}
	eventDataPtr := results[0]
	return eventDataPtr, eventDataSize, nil
}

func (f *WasmModuleString) writeParameterToMemory(ctx context.Context, eventData string) (uint64, uint64, error) {
	eventDataSize := uint64(len(eventData))
	results, err := f.mallocFunc.Call(ctx, eventDataSize)
	if err != nil {
		return 0, 0, err
	}
	eventDataPtr := results[0]
	if !f.module.Memory().Write(uint32(eventDataPtr), []byte(eventData)) {
		return 0, 0, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			eventDataPtr, eventDataSize, f.module.Memory().Size())
	}
	return eventDataPtr, eventDataSize, nil
}

func (f *WasmModuleString) readResultFromMemory(ctx context.Context, eventResultPtr uint64, eventResultSize uint64) (uint64, string, error) {
	if data, ok := f.module.Memory().Read(uint32(eventResultPtr), uint32(eventResultSize)); ok {
		var result EventFuncResult
		err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &result)
		if err != nil {
			return 0, "", err
		}
		responseString, err := f.readDataFromMemory(ctx, result.ResultPtrSize)
		if err != nil {
			return 0, "", err
		}
		return result.ResponseCode, responseString, nil
	}
	return 0, "", fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
		eventResultPtr, eventResultSize, f.module.Memory().Size())
}

func (f *WasmModuleString) readDataFromMemory(ctx context.Context, eventResultPtrSize uint64) (string, error) {
	logger := zerolog.Ctx(ctx)
	eventResultPtr := uint32(eventResultPtrSize >> 32)
	eventResultSize := uint32(eventResultPtrSize)
	if eventResultPtr != 0 {
		defer func() {
			_, err := f.FreeAdpter(ctx, uint64(eventResultPtr), uint64(eventResultSize))
			if err != nil {
				logger.Err(err).Msg("error freeing memory")
			}
		}()
	}
	bytes, ok := f.module.Memory().Read(eventResultPtr, eventResultSize)
	if !ok {
		return "", fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			eventResultPtr, eventResultSize, f.module.Memory().Size())
	}
	return string(bytes), nil
}

func (f *WasmModuleString) logForExport(ctx context.Context, m api.Module, id, level, offset, byteCount uint32) {
	logger := zerolog.Ctx(ctx)
	buf, ok := m.Memory().Read(offset, byteCount)
	if !ok {
		logger.Error().Msgf("Memory.Read(%d, %d) out of range", offset, byteCount)
	}
	msg := string(buf)
	logger.WithLevel(zerolog.Level(level)).Msg(msg)
	if f.logExt != nil {
		if err := f.logExt(ctx, id, level, msg); err != nil {
			logger.Err(err).Msg("error executing log extension function.")
		}
	}
}

func (f *WasmModuleString) Close(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	if err := f.module.Close(ctx); err != nil {
		logger.Err(err).Msg("error closing wasm runtime.")
	}
}
