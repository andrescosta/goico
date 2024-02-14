package collection_test

import (
	"testing"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/test"
)

func Test(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}
	for _, i := range s {
		test.Equals(t, i, q.Dequeue())
	}
	test.Equals(t, q.Size(), 0)
}

func TestDequeAll(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}
	d := q.DequeueAll()
	test.Equals(t, q.Size(), 0)
	test.Equals(t, d, s)
}

func TestPeekSlice(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}

	test.Equals(t, q.PeekSlice(1), s[0:1])
}

func TestPeekSliceAll(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}

	test.Equals(t, q.PeekSlice(-1), s)
	test.Equals(t, len(q.PeekSlice(-2)), 0)
}

func TestClear(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}

	test.Equals(t, q.PeekSlice(1), []int{0})
	q.Clear()
	test.Equals(t, q.Size(), 0)
}

func TestPeek(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}

	test.Equals(t, q.Peek(), s[0])
	test.Equals(t, q.Size(), len(s))
	v := q.Dequeue()
	test.Equals(t, v, s[0])
	test.Equals(t, q.Peek(), s[1])
	test.Equals(t, q.Size(), len(s)-1)
}

func TestDequeueSlice(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}
	d := q.DequeueN(50)
	test.Len(t, d, 21)
	test.Equals(t, s, d)
	test.Equals(t, q.Size(), 0)
}

func TestDequeueSliceMax(t *testing.T) {
	q := NewSyncQueue[int]()
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for _, i := range s {
		q.Queue(i)
	}
	d := q.DequeueN(10)
	test.Len(t, d, 10)
	test.Equals(t, s[0:10], d)
	test.Equals(t, q.Size(), 11)
	d = q.DequeueN(10)
	test.Len(t, d, 10)
	test.Equals(t, s[10:20], d)
	test.Equals(t, q.Size(), 1)
	d = q.DequeueN(10)
	test.Len(t, d, 1)
	test.Equals(t, s[20:], d)
	test.Equals(t, q.Size(), 0)
}
