package broadcaster

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrStopped = errors.New("broadcaster is stopped")

type Broadcaster[T any] struct {
	listeners     sync.Map
	c             chan T
	ctx           context.Context
	cancel        context.CancelFunc
	worker        *sync.WaitGroup
	statusBarrier *statusBarrier
}

type Listener[T any] struct {
	C             <-chan T
	c             chan T
	statusBarrier *statusBarrier
}

func Start[T any](ctx context.Context) *Broadcaster[T] {
	ctx1, cancel := context.WithCancel(ctx)
	broadcaster := &Broadcaster[T]{
		listeners:     sync.Map{},
		c:             make(chan T, 1),
		ctx:           ctx1,
		cancel:        cancel,
		worker:        &sync.WaitGroup{},
		statusBarrier: NewStatusBarrier(),
	}
	broadcaster.statusBarrier.MarkStarted()
	broadcaster.worker.Add(1)
	go func(b *Broadcaster[T]) {
		defer func() {
			close(b.c)
			b.worker.Done()
		}()
		for {
			select {
			case <-b.ctx.Done():
				return
			case d := <-b.c:
				b.listeners.Range(func(k any, _ any) bool {
					select {
					case <-b.ctx.Done():
						return false
					default:
						_ = k.(*Listener[T]).write(d)
						return true
					}
				})
			}
		}
	}(broadcaster)
	return broadcaster
}

func (b *Broadcaster[T]) Stop() error {
	b.statusBarrier.EnterForStopping()
	defer b.statusBarrier.EndStop()
	if b.statusBarrier.WasStopped() {
		return ErrStopped
	}
	b.cancel()
	b.worker.Wait()
	b.listeners.Range(func(k any, c any) bool {
		k.(*Listener[T]).stop()
		b.listeners.Delete(k)
		return true
	})
	b.statusBarrier.MarkStopped()
	return nil
}

func (b *Broadcaster[T]) Subscribe() (*Listener[T], error) {
	if b.statusBarrier.WasStopped() {
		return nil, ErrStopped
	}
	m := make(chan T, 1)
	l := &Listener[T]{
		C:             m,
		c:             m,
		statusBarrier: NewStatusBarrier(),
	}
	l.statusBarrier.MarkStarted()
	b.listeners.Store(l, m)
	return l, nil
}

func (b *Broadcaster[T]) Unsubscribe(l *Listener[T]) error {
	b.statusBarrier.Enter()
	defer b.statusBarrier.Out()
	if b.statusBarrier.WasStopped() {
		return ErrStopped
	}
	_, ok := b.listeners.LoadAndDelete(l)
	if ok {
		_ = l.stop()
	}
	return nil
}
func (b *Broadcaster[T]) IsSubscribed(l *Listener[T]) (bool, error) {
	if b.statusBarrier.WasStopped() {
		return false, ErrStopped
	}
	_, ok := b.listeners.Load(l)
	return ok, nil
}

func (b *Broadcaster[T]) Write(t T) error {
	b.statusBarrier.Enter()
	if b.statusBarrier.WasStopped() {
		return ErrStopped
	}
	go func() {
		defer b.statusBarrier.Out()
		ti := time.NewTimer(1 * time.Second)
		select {
		case b.c <- t:
		case <-ti.C:
		}
	}()
	return nil
}

func (b *Broadcaster[T]) WriteSync(t T) error {
	b.statusBarrier.Enter()
	defer b.statusBarrier.Out()
	if b.statusBarrier.WasStopped() {
		return ErrStopped
	}
	ti := time.NewTimer(1 * time.Second)
	select {
	case b.c <- t:
	case <-ti.C:
	}
	return nil
}

func (b *Listener[T]) stop() error {
	b.statusBarrier.EnterForStopping()
	defer b.statusBarrier.EndStop()
	if b.statusBarrier.WasStopped() {
		return ErrStopped
	}
	close(b.c)
	b.statusBarrier.MarkStopped()
	return nil
}

func (b *Listener[T]) write(t T) error {
	b.statusBarrier.Enter()
	if b.statusBarrier.WasStopped() {
		return ErrStopped
	}
	go func() {
		defer b.statusBarrier.Out()
		ti := time.NewTimer(1 * time.Second)
		select {
		case b.c <- t:
		case <-ti.C:
			break
		}
	}()
	return nil
}
