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

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/broadcaster"
	"github.com/andrescosta/goico/pkg/test"
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
	maxListeners := 100
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	for i := 0; i < maxListeners; i++ {
		waitListeners.Add(1)
		listener, err := b.Subscribe()
		test.Nil(t, err)
		go func() {
			defer waitListeners.Done()
			<-waiter
			timer := time.NewTimer(5 * time.Second)
			select {
			case d := <-listener.C:
				if d.name != newdata.name {
					t.Errorf("	 %s got %s", newdata.name, d.name)
				}
				if d.id != newdata.id {
					t.Errorf("id expected %d got %d", newdata.id, d.id)
				}
			case <-timer.C:
				t.Error("timeout")
			}
		}()
	}
	close(waiter)
	err := b.WriteSync(newdata)
	test.Nil(t, err)
	waitListeners.Wait()
	err = b.Stop()
	test.Nil(t, err)
}

func TestUnsubscribe(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	maxListeners := 200
	waiter := make(chan struct{})
	listeners := make([]*Listener[data], 0)
	for i := 0; i < maxListeners; i++ {
		listener, err := b.Subscribe()
		test.Nil(t, err)
		tounsusbcribe := i%2 != 0
		if tounsusbcribe {
			listeners = append(listeners, listener)
		}
		waitListeners.Add(1)
		go func(l *Listener[data], w *sync.WaitGroup, subcribed bool) {
			defer w.Done()
			<-waiter
			timer := time.NewTimer(1 * time.Second)
			select {
			case d, ok := <-l.C:
				if subcribed && ok {
					if d.name != newdata.name {
						t.Errorf("name expected %s got %s", newdata.name, d.name)
					}
					if d.id != newdata.id {
						t.Errorf("id expected %d got %d", newdata.id, d.id)
					}
				}
				if subcribed && !ok {
					t.Error("the channel was closed")
				}
				if !subcribed && ok {
					t.Error("It was unsubscribed")
				}
			case <-timer.C:
				if !subcribed {
					return
				}
				t.Error("timeout")
			}
		}(listener, &waitListeners, !tounsusbcribe)
	}
	for _, l := range listeners {
		err := b.Unsubscribe(l)
		test.Nil(t, err)
	}
	close(waiter)
	err := b.Write(newdata)
	test.Nil(t, err)
	waitListeners.Wait()
	err = b.Stop()
	test.Nil(t, err)
}

func TestStopWrite(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	maxListeners := 10
	waiter := make(chan struct{})
	for i := 0; i < maxListeners; i++ {
		l, err := b.Subscribe()
		test.Nil(t, err)
		waitListeners.Add(1)
		go func(l *Listener[data], w *sync.WaitGroup) {
			defer w.Done()
			<-waiter
			timer := time.NewTimer(1 * time.Millisecond)
			for {
				select {
				case d, ok := <-l.C:
					if ok {
						if d.name != newdata.name {
							t.Errorf("name expected %s got %s", newdata.name, d.name)
						}
						if d.id != newdata.id {
							t.Errorf("id expected %d got %d", newdata.id, d.id)
						}
					}
					return
				case <-timer.C:
					t.Error("timeout")
					return
				}
			}
		}(l, &waitListeners)
	}
	close(waiter)
	err := b.Write(newdata)
	test.Nil(t, err)
	err = b.Stop()
	test.Nil(t, err)
	waitListeners.Wait()
}

func TestStopWriteSync(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	maxListeners := 100
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	for i := 0; i < maxListeners; i++ {
		l, err := b.Subscribe()
		test.Nil(t, err)
		waitListeners.Add(1)
		go func(l *Listener[data], w *sync.WaitGroup) {
			defer w.Done()
			<-waiter
			timer := time.NewTimer(1 * time.Millisecond)
			for {
				select {
				case d, ok := <-l.C:
					if ok {
						if d.name != newdata.name {
							t.Errorf("name expected %s got %s", newdata.name, d.name)
						}
						if d.id != newdata.id {
							t.Errorf("id expected %d got %d", newdata.id, d.id)
						}
					}
					return
				case <-timer.C:
					if b.IsStopped() {
						return
					}
					t.Error("timeout")
					return
				}
			}
		}(l, &waitListeners)
	}
	close(waiter)
	err := b.WriteSync(newdata)
	test.Nil(t, err)
	err = b.Stop()
	test.Nil(t, err)
	waitListeners.Wait()
}

func TestStoppedError(t *testing.T) {
	newdata := data{
		name: "Customer 1",
		id:   1,
	}
	b := Start[data](context.Background())
	l, err := b.Subscribe()
	test.Nil(t, err)
	err = b.Stop()
	test.Nil(t, err)
	err = b.Stop()
	test.ErrorNotIs(t, err, ErrStopped)
	err = b.Write(newdata)
	test.ErrorNotIs(t, err, ErrStopped)
	err = b.WriteSync(newdata)
	test.ErrorNotIs(t, err, ErrStopped)
	err = b.Unsubscribe(l)
	test.ErrorNotIs(t, err, ErrStopped)
	_, err = b.Subscribe()
	test.ErrorNotIs(t, err, ErrStopped)
	_, err = b.IsSubscribed(l)
	test.ErrorNotIs(t, err, ErrStopped)
}

func TestUnsubscribeUnsubscribe(t *testing.T) {
	b := Start[data](context.Background())
	l, err := b.Subscribe()
	test.Nil(t, err)
	err = b.Unsubscribe(l)
	test.Nil(t, err)
	err = b.Unsubscribe(l)
	test.Nil(t, err)
	err = b.Stop()
	test.Nil(t, err)
}

func TestMultiWriters(t *testing.T) {
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 200
	maxListeners := 10
	finished := atomic.Bool{}
	finished.Store(false)
	timeoutTime := 4 * time.Second
	for i := 0; i < maxListeners; i++ {
		waitListeners.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer waitListeners.Done()
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
	waitListeners.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}
}

func TestMultiWritersSync(t *testing.T) {
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 20
	maxListeners := 100
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < maxListeners; i++ {
		waitListeners.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer waitListeners.Done()
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
					if n >= maxProducers {
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
			if err := b.WriteSync(newdata); err != nil {
				t.Errorf("Error not expected:%s", err)
				return
			}
		}(i)
	}
	finished.Store(true)
	waitListeners.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}
}

func TestMultiWritersSyncStop(t *testing.T) {
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 20
	maxListeners := 100
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < maxListeners; i++ {
		waitListeners.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer waitListeners.Done()
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
	stopID := randomInt(t, maxProducers)
	waitProducers := sync.WaitGroup{}
	for i := 0; i < maxProducers; i++ {
		waitProducers.Add(1)
		go func(id int) {
			defer waitProducers.Done()
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.WriteSync(newdata); err != nil && !errors.Is(err, ErrStopped) {
				t.Errorf("Error not expected:%s", err)
				return
			}
			if id == stopID {
				if err := b.Stop(); err != nil && !errors.Is(err, ErrStopped) {
					t.Errorf("Error not expected:%s", err)
					return
				}
			}
		}(i)
	}
	finished.Store(true)
	waitProducers.Wait()
	waitListeners.Wait()
}

func TestMultiWritersStop(t *testing.T) {
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 20
	maxListeners := 100
	finished := atomic.Bool{}
	finished.Store(false)
	for i := 0; i < maxListeners; i++ {
		waitListeners.Add(1)
		listener, err := b.Subscribe()
		if err != nil {
			t.Errorf("Broadcaster.Subscribe: %s", err)
			return
		}
		go func() {
			defer waitListeners.Done()
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
	stopID := randomInt(t, maxProducers)
	waitProducers := sync.WaitGroup{}
	for i := 0; i < maxProducers; i++ {
		waitProducers.Add(1)
		go func(id int) {
			defer waitProducers.Done()
			newdata := data{
				name: "Customer 1",
				id:   id,
			}
			if err := b.Write(newdata); err != nil && !errors.Is(err, ErrStopped) {
				t.Errorf("Error not expected:%s", err)
				return
			}
			if id == stopID {
				if err := b.Stop(); err != nil && !errors.Is(err, ErrStopped) {
					t.Errorf("Error not expected:%s", err)
					return
				}
			}
		}(i)
	}
	finished.Store(true)
	waitProducers.Wait()
	waitListeners.Wait()
}

func TestMultiWritersUnsubscribe(t *testing.T) {
	b := Start[data](context.Background())
	var waitListeners sync.WaitGroup
	waiter := make(chan struct{})
	maxProducers := 200
	maxListeners := 100
	maxClosers := 200
	finished := atomic.Bool{}
	finished.Store(false)
	listeners := make([]*listenerSync, maxListeners)
	for i := 0; i < maxListeners; i++ {
		waitListeners.Add(1)
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
			defer waitListeners.Done()
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
	waitProducers := sync.WaitGroup{}
	for i := 0; i < maxProducers; i++ {
		waitProducers.Add(1)
		go func(id int) {
			defer waitProducers.Done()
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
	for i := 0; i < maxClosers; i++ {
		go func() {
			i := randomInt(t, maxListeners)
			listeners[i].stopit(t)
		}()
	}
	finished.Store(true)
	waitListeners.Wait()
	waitProducers.Wait()
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected:%s", err)
	}
}

func TestMultiWritersMultiStop(t *testing.T) {
	b := Start[data](context.Background())
	var w sync.WaitGroup
	waiter := make(chan struct{})
	maxListeners := 100
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
						t.Errorf("name expected %s got %s", newdata.name, d.name)
					}
					if d.id != newdata.id {
						t.Errorf("id expected %d got %d", newdata.id, d.id)
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
	if err := b.Stop(); err != nil {
		t.Errorf("Error not expected: %s", err)
	}
	w.Wait()
	ww.Wait()
	cancel()
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
		if err := l.b.Unsubscribe(l.l); err != nil {
			t.Errorf("error not expected: %s", err)
		}
	}
}

func randomInt(t *testing.T, max int) int {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		t.Errorf("rand.Int:%s", err)
	}
	return int(i.Uint64())
}
