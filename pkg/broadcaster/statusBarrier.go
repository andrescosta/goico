package broadcaster

import (
	"sync"
)

type statusBarrier struct {
	mutex   *sync.RWMutex
	started bool
}

func newStatusBarrier() *statusBarrier {
	return &statusBarrier{
		mutex:   &sync.RWMutex{},
		started: false,
	}
}

func (b *statusBarrier) Entering() {
	b.mutex.RLock()
}

func (b *statusBarrier) Out() {
	b.mutex.RUnlock()
}

func (b *statusBarrier) EnteringStatusArea() {
	b.mutex.Lock()
}

func (b *statusBarrier) OutOfStatusArea() {
	b.mutex.Unlock()
}

func (b *statusBarrier) MarkStopped() {
	b.started = false
}

func (b *statusBarrier) MarkStarted() {
	b.started = true
}

func (b *statusBarrier) IsStopped() bool {
	return !b.started
}
