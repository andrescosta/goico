package broadcaster_test

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/andrescosta/goico/pkg/broadcaster"
)

type data struct {
	id   int
	name string
}

func TestBroadcasters(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	for i := 0; i < 1000; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer w.Done()
			<-waiter
			timer := time.NewTimer(5 * time.Second)
			select {
			case d := <-listener.C:
				if d.name != newdata.name {
					t.Errorf("error name expected %s got %s", newdata.name, d.name)
				}
				if d.id != newdata.id {
					t.Errorf("error id expected %d got %d", newdata.id, d.id)
				}
			case <-timer.C:
				t.Error("timeout")
			}
		}()
	}
	close(waiter)
	if err := b.WriteSync(newdata); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	w.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}
}

func TestUnsubscribe(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	var w sync.WaitGroup
	start := make(chan int)
	listeners := make([]*Listener[data], 0)
	for i := 0; i < 200; i++ {
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		tounsusbcribe := i%2 != 0
		if tounsusbcribe {
			listeners = append(listeners, listener)
		}
		w.Add(1)
		go func(l *Listener[data], w *sync.WaitGroup, subcribed bool) {
			defer w.Done()
			<-start
			timer := time.NewTimer(4 * time.Second)
			select {
			case d, ok := <-l.C:
				if subcribed && ok {
					if d.name != newdata.name {
						t.Errorf("error name expected %s got %s", newdata.name, d.name)
					}
					if d.id != newdata.id {
						t.Errorf("error id expected %d got %d", newdata.id, d.id)
					}
				}
				if subcribed && !ok {
					t.Error("the channel was closed")
				}
			case <-timer.C:
				t.Error("timeout")
			}
		}(listener, &w, !tounsusbcribe)
	}
	close(start)
	for _, l := range listeners {
		if err := b.Unsubscribe(l); err != nil {
			t.Errorf("Error not expected:%s", err)
			return
		}
	}
	if err := b.Write(newdata); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	w.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}
}

func TestStopWrite(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	var w sync.WaitGroup
	start := make(chan int)
	for i := 0; i < 10; i++ {
		l, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		w.Add(1)
		go func(l *Listener[data], w *sync.WaitGroup) {
			defer w.Done()
			<-start
			timer := time.NewTimer(1 * time.Millisecond)
			for {
				select {
				case d, ok := <-l.C:
					if ok {
						if d.name != newdata.name {
							t.Errorf("error name expected %s got %s", newdata.name, d.name)
						}
						if d.id != newdata.id {
							t.Errorf("error id expected %d got %d", newdata.id, d.id)
						}
					}
					return
				case <-timer.C:
					t.Error("timeout")
					return
				}
			}
		}(l, &w)
	}
	close(start)
	if err := b.Write(newdata); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	w.Wait()
}
func TestStopWriteSync(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	var w sync.WaitGroup
	start := make(chan int)
	for i := 0; i < 10; i++ {
		l, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		w.Add(1)
		go func(l *Listener[data], w *sync.WaitGroup) {
			defer w.Done()
			<-start
			timer := time.NewTimer(1 * time.Millisecond)
			for {
				select {
				case d, ok := <-l.C:
					if ok {
						if d.name != newdata.name {
							t.Errorf("error name expected %s got %s", newdata.name, d.name)
						}
						if d.id != newdata.id {
							t.Errorf("error id expected %d got %d", newdata.id, d.id)
						}
					}
					return
				case <-timer.C:
					t.Error("timeout")
					return
				}
			}
		}(l, &w)
	}
	close(start)
	if err := b.WriteSync(newdata); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	w.Wait()
}
func TestStoppedError(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	l, err := b.Subscribe()
	if err != nil {
		t.Errorf("Broadcaster.Subscribe: %s", err)
		return
	}
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Stop(); !errors.Is(err, ErrStopped) {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Write(newdata); !errors.Is(err, ErrStopped) {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.WriteSync(newdata); !errors.Is(err, ErrStopped) {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Unsubscribe(l); !errors.Is(err, ErrStopped) {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if _, err := b.Subscribe(); !errors.Is(err, ErrStopped) {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if _, err := b.IsSubscribed(l); !errors.Is(err, ErrStopped) {
		t.Errorf("Error not expected:%s", err)
		return
	}

}
func TestUnsubscribeUnsubscribe(t *testing.T) {
	b := Start[data](context.Background())
	l, err := b.Subscribe()
	if err != nil {
		t.Errorf("Broadcaster.Subscribe: %s", err)
		return
	}
	if err := b.Unsubscribe(l); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Unsubscribe(l); err != nil {
		t.Errorf("Error not expected:%s", err)
		return
	}
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}

}

func TestMultiWriters(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 20000
	maxListeners := 10
	finished := atomic.Bool{}
	finished.Store(false)
	timeoutTime := 4 * time.Second
	for i := 0; i < maxListeners; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer w.Done()
			<-waiter
			recved := 0
			timer := time.NewTimer(timeoutTime)
			for {
				select {
				case _, ok := <-listener.C:
					if !ok {
						t.Errorf("channel closed")
						return
					}
					recved++
					if recved >= maxProducers {
						return
					}
				case <-timer.C:
					if !finished.Load() {
						timer.Reset(timeoutTime)
						continue
					}
					t.Error("timeout")
					return
				}
			}
		}()
	}
	close(waiter)
	for i := 0; i < maxProducers; i++ {
		go func(id int) {
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.Write(newdata); err != nil {
				t.Errorf("Error not expected:%s", err)
				return
			}
		}(i)
	}
	finished.Store(true)
	w.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}

}

func TestMultiWritersSync(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxd := 20
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < 1000; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer w.Done()
			<-waiter
			n := 0
			timeoutTime := 4 * time.Second
			timer := time.NewTimer(timeoutTime)
			for {
				select {
				case _, ok := <-listener.C:
					if !ok {
						t.Errorf("channel closed")
						return
					}
					n++
					if n >= maxd {
						return
					}
				case <-timer.C:
					if !finished.Load() {
						timer.Reset(timeoutTime)
						continue
					}
					t.Error("timeout")
					return
				}
			}
		}()
	}
	close(waiter)
	for i := 0; i < maxd; i++ {
		go func(id int) {
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.WriteSync(newdata); err != nil {
				t.Errorf("Error not expected:%s", err)
				return
			}
		}(i)
	}
	finished.Store(true)
	w.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}

}

func TestMultiWritersSyncStop(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxd := 20
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < 1000; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer w.Done()
			<-waiter
			n := 0
			timer := time.NewTimer(4 * time.Second)
			for {
				select {
				case _, ok := <-listener.C:
					if !ok {
						return
					}
					n++
					if n >= maxd {
						return
					}
				case <-timer.C:
					if !finished.Load() {
						continue
					}
					t.Error("timeout")
					return
				}
			}
		}()
	}
	close(waiter)
	stopId := randomInt(t, maxd)
	ww := sync.WaitGroup{}
	for i := 0; i < maxd; i++ {
		ww.Add(1)
		go func(id int) {
			defer ww.Done()
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.WriteSync(newdata); err != nil && !errors.Is(err, ErrStopped) {
				t.Errorf("Error not expected:%s", err)
				return
			}
			if id == stopId {
				if err := b.Stop(); err != nil && !errors.Is(err, ErrStopped) {
					t.Errorf("Error not expected:%s", err)
					return
				}
			}
		}(i)
	}
	finished.Store(true)
	ww.Wait()
	w.Wait()
}
func TestMultiWritersStop(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxd := 20
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < 1000; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer w.Done()
			<-waiter
			n := 0
			timer := time.NewTimer(4 * time.Second)
			for {
				select {
				case _, ok := <-listener.C:
					if !ok {
						return
					}
					n++
					if n >= maxd {
						return
					}
				case <-timer.C:
					if !finished.Load() {
						continue
					}
					t.Error("timeout")
					return
				}
			}
		}()
	}
	close(waiter)
	stopId := randomInt(t, maxd)
	ww := sync.WaitGroup{}
	for i := 0; i < maxd; i++ {
		ww.Add(1)
		go func(id int) {
			defer ww.Done()
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.Write(newdata); err != nil && !errors.Is(err, ErrStopped) {
				t.Errorf("Error not expected:%s", err)
				return
			}
			if id == stopId {
				if err := b.Stop(); err != nil && !errors.Is(err, ErrStopped) {
					t.Errorf("Error not expected:%s", err)
					return
				}
			}
		}(i)
	}
	finished.Store(true)
	ww.Wait()
	w.Wait()
}

func TestMultiWritersUnsubscribe(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 200
	maxListeners := 1000
	finished := atomic.Bool{}
	finished.Store(false)
	listeners := make([]*listenerSync, maxListeners)
	group := listenersGroup{
		ls: listeners,
	}
	for i := 0; i < maxListeners; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		listeners[i] = &listenerSync{
			l: listener,
			b: b,
		}
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func(listener *Listener[data]) {
			defer w.Done()
			<-waiter
			n := 0
			timer := time.NewTimer(4 * time.Second)
			for {
				select {
				case _, ok := <-listener.C:
					if !ok {
						return
					}
					n++
					if n >= maxProducers {
						return
					}
				case <-timer.C:
					if !finished.Load() {
						continue
					}
					t.Error("timeout")
					return
				}
			}
		}(listener)
	}
	close(waiter)
	wp := sync.WaitGroup{}
	for i := 0; i < maxProducers; i++ {
		wp.Add(1)
		go func(id int) {
			defer wp.Done()
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.WriteSync(newdata); err != nil && !errors.Is(err, ErrStopped) {
				t.Errorf("Error not expected:%s", err)
				return
			}
		}(i)
	}
	for i := 0; i < maxListeners; i++ {
		go func() {
			i := randomInt(t, 600)
			group.stopit(t, i)
		}()
	}
	finished.Store(true)
	w.Wait()
	wp.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}

}

func TestMultiWritersMultiStop(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxListeners := 1000
	maxProducers := 200
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < maxListeners; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer w.Done()
			<-waiter
			n := 0
			timer := time.NewTimer(4 * time.Second)
			for {
				select {
				case _, ok := <-listener.C:
					if !ok {
						return
					}
					n++
					if n >= maxProducers {
						return
					}
				case <-timer.C:
					if !finished.Load() {
						continue
					}
					t.Error("timeout")
					return
				}
			}
		}()
	}
	close(waiter)
	for i := 0; i < maxProducers; i++ {
		go func(id int) {
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.Write(newdata); err != nil && !errors.Is(err, ErrStopped) {
				t.Errorf("Error not expected:%s", err)
				return
			}
			if id == randomInt(t, maxProducers) {
				if err := b.Stop(); err != nil && !errors.Is(err, ErrStopped) {
					t.Errorf("Error not expected:%s", err)
					return
				}
			}
		}(i)
	}
	finished.Store(true)
	w.Wait()
}

func TestWithTimeoutContext(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	for i := 0; i < 1000; i++ {
		w.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			cancel()
			return
		}
		go func() {
			defer w.Done()
			<-waiter
		loop:
			for {
				timer := time.NewTimer(1 * time.Millisecond)
				select {
				case d, ok := <-listener.C:
					if !ok {
						break loop
					}
					if d.name != newdata.name {
						t.Errorf("error name expected %s got %s", newdata.name, d.name)
					}
					if d.id != newdata.id {
						t.Errorf("error id expected %d got %d", newdata.id, d.id)
					}

				case <-timer.C:
					break loop
				}
			}
		}()
	}
	close(waiter)
	ww := sync.WaitGroup{}
	ww.Add(1)
	go func() {
		defer ww.Done()
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			default:
				if err := b.Write(newdata); err != nil {
					if errors.Is(err, ErrStopped) {
						return
					}
					t.Errorf("Error not expected:%s", err)
					return
				}
			}
		}
	}()
	<-ctx.Done()
	b.Stop()
	w.Wait()
	ww.Wait()
	cancel()
}

type listenersGroup struct {
	ls []*listenerSync
}

func (l *listenersGroup) stopit(t *testing.T, i int) {
	l.ls[i].stopit(t)
}

type listenerSync struct {
	l *Listener[data]
	b *Broadcaster[data]
	m sync.Mutex
}

func (l *listenerSync) stopit(t *testing.T) {
	l.m.Lock()
	defer l.m.Unlock()
	b, err := l.b.IsSubscribed(l.l)
	if err != nil {
		t.Errorf("IsSubscribed: %s", err)
	}
	if b {
		l.b.Unsubscribe(l.l)
	}
}

func randomInt(t *testing.T, max int) int {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		t.Errorf("rand.Int:%s", err)
	}
	return int(i.Uint64())
}
