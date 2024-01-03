package broadcaster

import (
	"context"
	"sync"
)

type Broadcaster[T any] struct {
	listeners sync.Map
	c         chan T
	cancel    context.CancelFunc
	ctx       context.Context
}

type Listener[T any] struct {
	C <-chan T
}

func Start[T any](ctx context.Context) *Broadcaster[T] {
	ctxB, cancel := context.WithCancel(ctx)
	b := &Broadcaster[T]{
		listeners: sync.Map{},
		c:         make(chan T, 1),
		cancel:    cancel,
		ctx:       ctxB,
	}
	go func(c *Broadcaster[T]) {
		for {
			select {
			case <-ctxB.Done():
				return
			case d, ok := <-c.c:
				if ok {
					c.listeners.Range(func(_ any, v any) bool {
						select {
						case <-ctxB.Done():
							return false
						case v.(chan T) <- d:
							return true
						}
					})
				}
			}
		}
	}(b)
	return b
}

func (b *Broadcaster[T]) Stop() {
	b.cancel()
	b.listeners.Range(func(k any, _ any) bool {
		b.Unsubscribe(k.(*Listener[T]))
		return true
	})
}

func (b *Broadcaster[T]) Subscribe() *Listener[T] {
	m := make(chan T, 1)
	l := &Listener[T]{C: m}
	b.listeners.Store(l, m)
	return l
}

func (b *Broadcaster[T]) Unsubscribe(l *Listener[T]) {
	c, ok := b.listeners.LoadAndDelete(l)
	if ok {
		close(c.(chan T))
	}
}

func (b *Broadcaster[T]) Write(t T) {
	go func() {
		select {
		case <-b.ctx.Done():
			return
		case b.c <- t:
			return
		}
	}()
}
