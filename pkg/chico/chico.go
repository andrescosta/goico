package chico

import (
	"context"
	"sync"
)

type Broadcaster[T any] struct {
	listeners sync.Map
	c         chan T
}

type Listener[T any] struct {
	C <-chan T
}

func Start[T any](ctx context.Context) *Broadcaster[T] {
	b := &Broadcaster[T]{
		listeners: sync.Map{},
		c:         make(chan T, 1),
	}

	go func(c *Broadcaster[T]) {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-c.c:
				if ok {
					c.listeners.Range(func(_ any, v any) bool {
						select {
						case v.(chan T) <- d:
						default:
						}
						return true
					})
				}
			}
		}

	}(b)
	return b
}

func (c *Broadcaster[T]) Stop() {
	c.listeners.Range(func(k any, _ any) bool {
		c.Unsubscribe(k.(*Listener[T]))
		return true
	})

}

func (c *Broadcaster[T]) Subscribe() *Listener[T] {
	m := make(chan T, 1)
	l := &Listener[T]{C: m}
	c.listeners.Store(l, m)
	return l
}

func (b *Broadcaster[T]) Unsubscribe(l *Listener[T]) {
	c, ok := b.listeners.LoadAndDelete(l)
	if ok {
		close(c.(chan T))
	}
}

func (c *Broadcaster[T]) Write(t T) {
	go func() {
		select {
		case c.c <- t:
		default:
		}
	}()
}
