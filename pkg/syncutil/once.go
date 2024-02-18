package syncutil

import (
	"context"
	"errors"
	"sync"
)

type OnceDisposable struct {
	done       *GValue[bool]
	disposed   *GValue[bool]
	errDone    *GValue[error]
	errDispose *GValue[error]
	m          *sync.Mutex
}

func NewOnceDisposable() *OnceDisposable {
	o := &OnceDisposable{
		done:       NewGValue(false),
		disposed:   NewGValue(false),
		errDone:    &GValue[error]{},
		errDispose: &GValue[error]{},
		m:          &sync.Mutex{},
	}
	return o
}

func (o *OnceDisposable) Do(ctx context.Context, f func(ctx context.Context) error) error {
	if o.disposed.Load() {
		return errors.New("disposable object already disposed")
	}
	if !o.done.Load() && !o.disposed.Load() {
		o.m.Lock()
		defer o.m.Unlock()
		if !o.done.Load() && !o.disposed.Load() {
			defer o.done.Store(true)
			err := f(ctx)
			if err != nil {
				o.errDone.Store(err)
			}
		}
	}
	return o.errDone.Load()
}

func (o *OnceDisposable) Dispose(ctx context.Context, f func(ctx context.Context) error) error {
	if !o.wasDisposed() {
		o.m.Lock()
		defer o.m.Unlock()
		if !o.wasDisposed() {
			defer o.setDisposed()
			err := f(ctx)
			if err != nil {
				o.errDispose.Store(err)
			}
		}
	}
	return o.errDispose.Load()
}

func (o *OnceDisposable) wasDisposed() bool {
	return o.disposed.Load()
}

func (o *OnceDisposable) setDisposed() {
	o.disposed.Store(true)
}
