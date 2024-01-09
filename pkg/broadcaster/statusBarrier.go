package broadcaster

import (
	"sync"
	"sync/atomic"
)

type statusBarrier struct {
	mutex   *sync.RWMutex
	started *atomic.Bool
}

func NewStatusBarrier() *statusBarrier {
	return &statusBarrier{
		mutex:   &sync.RWMutex{},
		started: &atomic.Bool{},
	}
}

func (b *statusBarrier) Entering() {
	b.mutex.RLock()
}

func (b *statusBarrier) Out() {
	b.mutex.RUnlock()
}

func (b *statusBarrier) EnteringStoppingArea() {
	b.mutex.Lock()
}

func (b *statusBarrier) OutOfStoppingArea() {
	b.mutex.Unlock()
}

func (b *statusBarrier) MarkStopped() {
	b.started.Store(false)
}

func (b *statusBarrier) MarkStarted() {
	b.started.Store(true)
}

func (b *statusBarrier) IsStopped() bool {
	return !b.started.Load()
}
