package syncutil

import (
	"context"
	"sync"
)

type OnceDisposable struct {
	done    *GValue[bool]
	diposed *GValue[bool]
	err     *GValue[error]
	m       sync.Mutex
}

func NewOnceDisposable() *OnceDisposable {
	return &OnceDisposable{
		done:    NewGValue(false),
		diposed: NewGValue(false),
		err:     &GValue[error]{},
	}
}

func (o *OnceDisposable) Do(ctx context.Context, f func(ctx context.Context) error) error {
	if !o.done.Load() && !o.diposed.Load() {
		o.m.Lock()
		defer o.m.Unlock()
		if !o.done.Load() && !o.diposed.Load() {
			defer o.done.Store(true)
			err := f(ctx)
			if err != nil {
				o.err.Store(err)
			}
		}
	}
	return o.err.Load()
}

func (o *OnceDisposable) Dispose(ctx context.Context, f func(ctx context.Context) error) {
	if o.done.Load() && !o.diposed.Load() {
		o.m.Lock()
		defer o.m.Unlock()
		if o.done.Load() && !o.diposed.Load() {
			defer o.diposed.Store(true)
			err := f(ctx)
			if err != nil {
				o.err.Store(err)
			}
		}
	}
}

func (o *OnceDisposable) Err() error {
	return o.err.Load()
}
