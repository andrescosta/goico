package collection_test

import (
	"fmt"
	"sync"
	"testing"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/collection"
)

func TestSyncMap(t *testing.T) {
	m := NewSyncMap[string, string]()
	w := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		w.Add(1)
		go testMap(t, fmt.Sprintf("goroutine=%d", i), m, w)
	}
	w.Wait()
}

func testMap(t *testing.T, preffix string, m *SyncMap[string, string], w *sync.WaitGroup) {
	defer w.Done()
	k1 := preffix + "-key1"
	v1 := preffix + "-val1"
	v11 := preffix + "-val11"
	k2 := preffix + "-key2"
	v2 := preffix + "-val2"
	m.Store(k1, v1)
	m.Store(k2, v2)
	v1r, ok := m.Load(k1)
	if !ok {
		t.Errorf("%s not found", k1)
	}
	if v1r != v1 {
		t.Errorf("%s values are different %s - %s", k1, v1, v1r)
	}
	m.Delete(k1)
	_, ok = m.Load(k1)
	if ok {
		t.Errorf("%s  found", k1)
	}
	m.Store(k1, v1)
	m.Swap(k1, v11)
	v1r, ok = m.Load(k1)
	if !ok {
		t.Errorf("%s not found", k1)
	}
	if v1r != v11 {
		t.Errorf("%s values are different %s - %s", k1, v11, v1r)
	}
	m.Store(k2, v2)
	m.Swap(k1, v1)
	set1 := NewSet[string]()
	m.Range(func(t, s string) bool {
		set1.Add(t + s)
		return true
	})
	if !set1.Has(k1 + v1) {
		t.Errorf("%s/%s not found", k1, v1)
	}
	if !set1.Has(k2 + v2) {
		t.Errorf("%s/%s not found", k1, v1)
	}

	var value1 string
	m.Range(func(t, s string) bool {
		value1 = s
		return t != k2
	})

	if value1 != v2 {
		t.Errorf("expected %s got %s", v2, value1)
	}
}
