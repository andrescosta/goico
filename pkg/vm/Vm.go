package vm

import "github.com/andrescosta/goico/pkg/vm/wasm"

type Vm struct {
	runtime *wasm.Runtime
	hook    *Hook
}
type Hook struct {
	log wasm.LogFn
}

func New(dir string) (*Vm, error) {
	return NewWithHook(dir, nil)
}

func NewWithHook(dir string, hook *Hook) (*Vm, error) {
	r, err := wasm.NewRuntime(dir)
	if err != nil {
		return nil, err
	}
	return &Vm{
		runtime: r,
		hook:    hook,
	}, nil
}

func (v Vm) Execute() error {
	//wasm.NewModule()
}
