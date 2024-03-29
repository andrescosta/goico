package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/andrescosta/goico/pkg/broadcaster"
	"github.com/andrescosta/goico/pkg/service/grpc/cache/event"
)

type Cache[K comparable, V any] struct {
	name        string
	defs        *sync.Map
	broadcaster *broadcaster.Broadcaster[*event.Event]
}

func New[K comparable, V any](ctx context.Context, name string, publish bool) *Cache[K, V] {
	var b *broadcaster.Broadcaster[*event.Event]
	if publish {
		b = broadcaster.NewAndStart[*event.Event](ctx)
	}
	return &Cache[K, V]{
		name:        name,
		defs:        &sync.Map{},
		broadcaster: b,
	}
}

func (c *Cache[K, V]) Close() error {
	if c.broadcaster != nil {
		return c.broadcaster.Stop()
	}
	return nil
}

func (c *Cache[K, V]) Name() string {
	return c.name
}

func (c *Cache[K, V]) Subscribe() (*broadcaster.Listener[*event.Event], error) {
	if c.broadcaster == nil {
		return nil, errors.New("broadcasting disabled")
	}
	l, err := c.broadcaster.Subscribe()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (c *Cache[K, V]) Unsubscribe(l *broadcaster.Listener[*event.Event]) error {
	if c.broadcaster == nil {
		return errors.New("broadcasting disabled")
	}
	return c.broadcaster.Unsubscribe(l)
}

func (c *Cache[K, V]) AddOrUpdate(k K, v V) error {
	c.defs.Store(k, v)
	e := event.Event{
		Type: event.Event_Add,
		Name: c.name,
	}
	if c.broadcaster != nil {
		return c.broadcaster.Write(&e)
	}
	return nil
}

func (c *Cache[K, V]) Delete(k K) error {
	c.defs.Delete(k)
	e := event.Event{
		Type: event.Event_Delete,
		Name: c.name,
	}
	if c.broadcaster != nil {
		return c.broadcaster.Write(&e)
	}
	return nil
}

func (c *Cache[K, V]) Update(k K, v V) error {
	c.defs.Swap(k, v)
	e := event.Event{
		Type: event.Event_Update,
		Name: c.name,
	}
	if c.broadcaster != nil {
		return c.broadcaster.Write(&e)
	}
	return nil
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.defs.Load(k)
	if !ok {
		var vv V
		return vv, false
	}
	return v.(V), true
}
