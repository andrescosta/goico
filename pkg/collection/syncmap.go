package collection

import "sync"

type SyncMap[T comparable, S any] struct {
	lock *sync.RWMutex
	mmap map[T]S
}

func NewSyncMap[T comparable, S any]() *SyncMap[T, S] {
	return &SyncMap[T, S]{
		mmap: make(map[T]S),
		lock: &sync.RWMutex{},
	}
}

func (s *SyncMap[T, S]) Swap(k T, v S) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.mmap, k)
	s.mmap[k] = v
}

func (s *SyncMap[T, S]) LoadOrStore(k T, v S) S {
	s.lock.Lock()
	defer s.lock.Unlock()
	va, ok := s.mmap[k]
	if !ok {
		va = v
		s.mmap[k] = va
	}
	return va
}

func (s *SyncMap[T, S]) Store(k T, v S) {
	s.Swap(k, v)
}

func (s *SyncMap[T, S]) Load(k T) (S, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	res, ok := s.mmap[k]
	return res, ok
}

func (s *SyncMap[T, S]) Delete(k T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.mmap, k)
}

func (s *SyncMap[T, S]) Range(f func(T, S) bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for k, v := range s.mmap {
		if !f(k, v) {
			return
		}
	}
}
