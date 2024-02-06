package collection

import (
	"reflect"
	"sync"
)

// TODO: improve it. This queue implementation is very rudimentary and only used for testing.
type SyncQueue[T any] struct {
	data  []T
	mutex *sync.Mutex
}

func NewQueue[T any]() *SyncQueue[T] {
	return &SyncQueue[T]{
		data:  make([]T, 0),
		mutex: &sync.Mutex{},
	}
}

func (s *SyncQueue[T]) Queue(t T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data = append(s.data, t)
}

func (s *SyncQueue[T]) Dequeue() T {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	d := s.data[0]
	s.data = s.data[1:]
	return d
}

func (s *SyncQueue[T]) Peek(n int) []T {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if n == -1 {
		n = len(s.data)
	}
	if n > len(s.data) {
		n = len(s.data)
	}
	newq := make([]T, n)
	copy(newq, s.data)
	return newq
}

func (s *SyncQueue[T]) Slice() []T {
	return s.Peek(-1)
}

func (s *SyncQueue[T]) Write(p T) (n int, err error) {
	s.Queue(p)
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Array {
		return v.Len(), nil
	}
	return 0, nil
}

func (s *SyncQueue[T]) Size() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.data)
}

func (s *SyncQueue[T]) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data = make([]T, 0)
}
