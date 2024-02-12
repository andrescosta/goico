package syncutil

import "sync/atomic"

type GValue[T any] struct {
	_     [0]func()
	value atomic.Value
}

func NewGValue[T any](t T) *GValue[T] {
	gv := &GValue[T]{}
	gv.Store(t)
	return gv
}

func (v *GValue[T]) Load() T {
	val, ok := v.value.Load().(T)
	if !ok {
		var t T
		return t
	}
	return val
}

func (v *GValue[T]) Store(val T) {
	v.value.Store(val)
}

func (v *GValue[T]) Swap(newval T) T {
	val, ok := v.value.Swap(newval).(T)
	if !ok {
		var t T
		return t
	}
	return val
}

func (v *GValue[T]) CompareAndSwap(oldval T, newval T) bool {
	return v.value.CompareAndSwap(oldval, newval)
}

func (v *GValue[T]) Init() {
	var t T
	v.Store(t)
}
