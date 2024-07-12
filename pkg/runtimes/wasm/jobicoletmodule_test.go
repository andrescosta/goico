package wasm_test

import (
	"context"
	_ "embed"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/runtimes/wasm"
	"github.com/andrescosta/goico/pkg/test"
)

//go:embed testdata/log.wasm
var logw []byte

//go:embed testdata/echo.wasm
var echo []byte

//go:embed testdata/error.wasm
var doerror []byte

//go:embed testdata/panic.wasm
var panicw []byte

//go:embed testdata/sleeper.wasm
var sleeper []byte

//go:embed testdata/greetrust.wasm
var greetrust []byte

// Test sunny case: input-output
// Test return error
// Test log
type (
	scenario interface {
		name() string
		input() string
		wasm() []byte
		logFn() wasm.LogFn
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
		logs         *collection.SyncMap[string, logvalue]
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
	}
)

func Test(t *testing.T) {
	scenarios := []scenario{
		&scenarioresult{
			config:    config{"test_ok", echo},
			inputdata: inputdata{"test_ok", 0},
		},
		&scenarioresult{
			config:    config{"Hello, from a rusty script!", greetrust},
			inputdata: inputdata{"Hello, from a rusty script!", 0},
		},
		&scenarioresult{
			config:    config{"test_error", doerror},
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
			inputdata: inputdata{"log_ok_", 0},
			logsexpected: []logvalue{
				{message: "_nolevel", lvl: 6},
				{message: "_info", lvl: 1},
				{message: "_debug", lvl: 0},
				{message: "_warn", lvl: 2},
				{message: "_error", lvl: 3},
				{message: "_fatal", lvl: 4},
				{message: "_panic", lvl: 5},
			},
			logs: collection.NewSyncMap[string, logvalue](),
		},
		&scenariolog{
			config:       config{"log_nook", logw},
			inputdata:    inputdata{"log_ok_", 0},
			logsexpected: nil,
		},
	}
	dir := t.TempDir()
	ctx := context.Background()
	wg := sync.WaitGroup{}
	for _, s := range scenarios {
		t.Run(s.name(), func(t *testing.T) {
			m, err := wasm.NewJobicoletModule(ctx, dir, s.wasm(), "event", s.logFn())
			test.Nil(t, err)
			t.Cleanup(func() {
				if m != nil {
					m.Close(ctx)
				}
			})
			n := 1
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(i int, s scenario) {
					defer wg.Done()
					msg := s.input()
					ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
					defer cancel()
					code, str, err := m.Run(ctx, msg)
					s.validateError(t, err)
					s.validate(t, code, str)
				}(i, s)
			}
			wg.Wait()
		})
	}
}

func (s *scenariolog) log(_ context.Context, lvl uint32, message string) error {
	if s.logsexpected == nil {
		return errors.New("error")
	}
	s.logs.Store(message, logvalue{message, lvl})
	return nil
}

func (s *scenariowitherror) validate(*testing.T, uint64, string) {
}

func (s *scenariolog) validate(t *testing.T, code uint64, message string) {
	for _, l := range s.logsexpected {
		line := s.message + l.message
		le, ok := s.logs.Load(line)
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
	t.Helper()
	test.NotNil(t, err)
}

func (*scenarioresult) validateError(t *testing.T, err error) {
	t.Helper()
	test.Nil(t, err)
}

func (i *inputdata) input() string {
	return i.message
}

func (s *scenariolog) logFn() wasm.LogFn {
	return s.log
}

func (c *config) wasm() []byte {
	return c.wasmm
}

func (c *config) name() string {
	return c.names
}

func (*defaultlog) logFn() wasm.LogFn {
	return log
}

func log(context.Context, uint32, string) error {
	return nil
}
