package utilico

import "sync"

type SyncMap[T comparable, S any] struct {
	mymap map[T]S
	lock  *sync.RWMutex
}

func NewSyncMap[T comparable, S any]() *SyncMap[T, S] {
	return &SyncMap[T, S]{
		mymap: make(map[T]S),
		lock:  &sync.RWMutex{},
	}
}
func (s *SyncMap[T, S]) Swap(k T, v S) {
	s.Delete(k)
	s.Store(k, v)
}
func (s *SyncMap[T, S]) Store(k T, v S) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.mymap[k] = v
}

func (s *SyncMap[T, S]) Load(k T) (S, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	v, ok := s.mymap[k]
	return v, ok
}

func (s *SyncMap[T, S]) Delete(k T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.mymap, k)
}

func (s *SyncMap[T, S]) Range(f func(T, S) bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for k, v := range s.mymap {
		if !f(k, v) {
			return
		}
	}
}
