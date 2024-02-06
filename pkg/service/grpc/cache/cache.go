package cache

import (
	"context"
	"sync"

	"github.com/andrescosta/goico/pkg/broadcaster"
	"github.com/andrescosta/goico/pkg/service/grpc/cache/event"
)

type Cache[K comparable, V any] struct {
	name      string
	defs      *sync.Map
	listeners *broadcaster.Broadcaster[*event.Event]
}

func New[K comparable, V any](ctx context.Context, name string) *Cache[K, V] {
	b := broadcaster.Start[*event.Event](ctx)
	return &Cache[K, V]{
		name:      name,
		defs:      &sync.Map{},
		listeners: b,
	}
}

func (c *Cache[K, V]) Close() error {
	return c.listeners.Stop()
}

func (c *Cache[K, V]) Name() string {
	return c.name
}

func (c *Cache[K, V]) Subscribe() (*broadcaster.Listener[*event.Event], error) {
	l, err := c.listeners.Subscribe()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (c *Cache[K, V]) Unsubscribe(l *broadcaster.Listener[*event.Event]) error {
	return c.listeners.Unsubscribe(l)
}

func (c *Cache[K, V]) AddOrUpdate(k K, v V) error {
	c.defs.Store(k, v)
	e := event.Event{
		Type: event.Event_Add,
		Name: c.name,
	}
	return c.listeners.Write(&e)
}

func (c *Cache[K, V]) Delete(k K) error {
	c.defs.Delete(k)
	e := event.Event{
		Type: event.Event_Delete,
		Name: c.name,
	}
	return c.listeners.Write(&e)
}

func (c *Cache[K, V]) Update(k K, v V) error {
	c.defs.Swap(k, v)
	e := event.Event{
		Type: event.Event_Update,
		Name: c.name,
	}
	return c.listeners.Write(&e)
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.defs.Load(k)
	if !ok {
		var vv V
		return vv, false
	}
	return v.(V), true
}
