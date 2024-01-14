package wazero_test

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/wasm/wazero"
)

//go:embed testdata/log.wasm
var logw []byte

//go:embed testdata/echo.wasm
var echo []byte

//go:embed testdata/panic.wasm
var panicw []byte

//go:embed testdata/sleeper.wasm
var sleeper []byte

// Test sunny case: input-output
// Test return error
// Test log
type (
	scenario interface {
		name() string
		input() (uint32, string)
		wasm() []byte
		logFn() wazero.LogExt
		validate(*testing.T, uint64, string)
		validateError(*testing.T, error)
	}

	scenarioresult struct {
		config
		defaultlog
		inputdata
	}
	scenariowitherror struct {
		config
		defaultlog
		inputdata
	}

	scenariolog struct {
		config
		inputdata
		logs         map[string]logvalue
		logsexpected []logvalue
	}
)

type (
	config struct {
		names string
		wasmm []byte
	}
	defaultlog struct{}

	inputdata struct {
		message string
		code    uint32
	}
	logvalue struct {
		message string
		lvl     uint32
		id      uint32
	}
)

func Test(t *testing.T) {
	scenarios := []scenario{
		&scenarioresult{
			config:    config{"test_ok", echo},
			inputdata: inputdata{"test_ok", 0},
		},
		&scenarioresult{
			config:    config{"test_error", echo},
			inputdata: inputdata{"test_error", 500},
		},
		&scenariowitherror{
			config:    config{"test_error", panicw},
			inputdata: inputdata{"test_error", 500},
		},
		&scenariowitherror{
			config:    config{"test_infine_loop", sleeper},
			inputdata: inputdata{"test_infine_loop", 500},
		},
		&scenariolog{
			config:    config{"log", logw},
			inputdata: inputdata{"log_ok_", 10},
			logsexpected: []logvalue{
				{message: "_nolevel", lvl: 6},
				{message: "_info", lvl: 1},
				{message: "_debug", lvl: 0},
				{message: "_warn", lvl: 2},
				{message: "_error", lvl: 3},
				{message: "_fatal", lvl: 4},
				{message: "_panic", lvl: 5},
			},
			logs: make(map[string]logvalue),
		},
		&scenariolog{
			config:       config{"log_nook", logw},
			inputdata:    inputdata{"log_ok_", 10},
			logsexpected: nil,
		},
	}
	dir := t.TempDir()
	ctx := context.Background()
	runtime, err := wazero.NewWasmRuntime(dir)
	if err != nil {
		t.Fatalf("not expected error: %v", err)
	}
	defer func() {
		runtime.Close(ctx)
	}()
	for _, s := range scenarios {
		t.Run(s.name(), func(t *testing.T) {
			t.Setenv("wasm.timeout", (2 * time.Second).String())
			m, err := wazero.NewWasmModuleString(ctx, runtime, s.wasm(), "event", s.logFn())
			if err != nil {
				t.Fatalf("not expected error: %v", err)
			}
			t.Cleanup(func() {
				m.Close(ctx)
			})
			id, msg := s.input()
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			code, str, err := m.ExecuteMainFunc(ctx, id, msg)
			s.validateError(t, err)
			s.validate(t, code, str)
		})
	}
}

func TestParalel(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	runtime, err := wazero.NewWasmRuntime(dir)
	if err != nil {
		t.Fatalf("not expected error: %v", err)
	}
	defer func() {
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("not expecting error: %v", err)
		}
	}()

	wgready := sync.WaitGroup{}
	wg := sync.WaitGroup{}

	scenarios := []scenario{
		&scenarioresult{
			config:    config{"test_ok", echo},
			inputdata: inputdata{"test_ok", 0},
		},
		&scenarioresult{
			config:    config{"test_error", echo},
			inputdata: inputdata{"test_error", 500},
		},
		&scenariowitherror{
			config:    config{"test_error", panicw},
			inputdata: inputdata{"test_error", 500},
		},
		&scenariowitherror{
			config:    config{"test_infine_loop", sleeper},
			inputdata: inputdata{"test_infine_loop", 500},
		},
		&scenariolog{
			config:    config{"log", logw},
			inputdata: inputdata{"log_ok_", 10},
			logsexpected: []logvalue{
				{message: "_nolevel", lvl: 6},
				{message: "_info", lvl: 1},
				{message: "_debug", lvl: 0},
				{message: "_warn", lvl: 2},
				{message: "_error", lvl: 3},
				{message: "_fatal", lvl: 4},
				{message: "_panic", lvl: 5},
			},
			logs: make(map[string]logvalue),
		},
		&scenariolog{
			config:       config{"log_nook", logw},
			inputdata:    inputdata{"log_ok_", 10},
			logsexpected: nil,
		},
	}

	for _, s := range scenarios {
		wg.Add(1)
		wgready.Add(1)
		go func(s scenario) {
			defer wg.Done()
			t.Run(s.name()+"_parallel", func(t *testing.T) {
				_, ok := os.LookupEnv("wasm.timeout")
				print(ok)
				m, err := wazero.NewWasmModuleString(ctx, runtime, s.wasm(), "event", s.logFn())
				if err != nil {
					t.Errorf("not expected error: %v", err)
					return
				}
				t.Cleanup(func() {
					if err := m.Close(ctx); err != nil {
						t.Errorf("not expecting error %v", err)
					}
				})
				id, msg := s.input()
				wgready.Done()
				wgready.Wait()
				ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
				code, str, err := m.ExecuteMainFunc(ctx, id, msg)
				cancel()
				s.validateError(t, err)
				s.validate(t, code, str)
			})
		}(s)
	}
	wg.Wait()
}

func (s *scenariolog) log(_ context.Context, id, lvl uint32, message string) error {
	if s.logsexpected == nil {
		return errors.New("error")
	}
	s.logs[message] = logvalue{message, lvl, id}
	return nil
}

func (s *scenariowitherror) validate(*testing.T, uint64, string) {
}

func (s *scenariolog) validate(t *testing.T, code uint64, message string) {
	for _, l := range s.logsexpected {
		line := s.message + l.message
		le, ok := s.logs[line]
		if !ok {
			t.Errorf(line + " not sent")
			continue
		}
		if code != 0 {
			t.Errorf("expected 0 got %d", code)
		}
		if message != "ok" {
			t.Errorf("expected ok got %s", message)
		}
		if s.code != le.id {
			t.Errorf("expected %d got %d", l.id, le.id)
		}
		if le.lvl != l.lvl {
			t.Errorf("expected %d got %d", l.lvl, le.lvl)
		}
	}
}

func (s *scenarioresult) validate(t *testing.T, code uint64, message string) {
	if code != uint64(s.code) {
		t.Errorf("expected %d got %d", s.code, code)
	}
	if message != s.message {
		t.Errorf("expected %s got %s", s.message, message)
	}
}

func (s *scenariolog) validateError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("not expected error: %v", err)
		return
	}
}

func (*scenariowitherror) validateError(t *testing.T, err error) {
	if err == nil {
		t.Error("expected error got <nil>")
	}
}

func (*scenarioresult) validateError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("not expected error: %v", err)
	}
}

func (i *inputdata) input() (uint32, string) {
	return i.code, i.message
}

func (s *scenariolog) logFn() wazero.LogExt {
	return s.log
}

func (c *config) wasm() []byte {
	return c.wasmm
}

func (c *config) name() string {
	return c.names
}

func (*defaultlog) logFn() wazero.LogExt {
	return log
}

func log(context.Context, uint32, uint32, string) error {
	return nil
}
