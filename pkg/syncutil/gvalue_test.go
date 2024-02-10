package syncutil_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/andrescosta/goico/pkg/syncutil"
	"github.com/andrescosta/goico/pkg/test"
)

func TestOk(t *testing.T) {
	e := syncutil.NewGValue(-1)
	i := 1
	e.Store(i)
	ii := e.Load()
	test.Equals(t, i, ii)
}

func TestError(t *testing.T) {
	err := errors.New("test")
	e := syncutil.GValue[error]{}
	e.Store(err)
	errt := e.Load()
	test.ErrorIs(t, errt, err)
}

func TestNilError(t *testing.T) {
	var e syncutil.GValue[error]
	test.Nil(t, e.Load())
}

func TestEmptyString(t *testing.T) {
	var e syncutil.GValue[string]
	test.Equals(t, e.Load(), "")
}

func TestStringNilPtr(t *testing.T) {
	var e syncutil.GValue[*string]
	v := e.Load()
	j := (*string)(nil)
	if v != j {
		t.Error("expected (*string)nil")
	}
}

func TestRace(t *testing.T) {
	var w sync.WaitGroup
	v := syncutil.NewGValue(-1)
	ch := make(chan struct{})
	for i := 0; i < 200; i++ {
		w.Add(1)
		if i%2 == 0 {
			go func(i int) {
				defer w.Done()
				<-ch
				_ = v.Load()
				v.Store(i)
			}(i)
		} else {
			go func(i int) {
				defer w.Done()
				<-ch
				j := i + 1
				v.CompareAndSwap(v.Load(), j)
			}(i)
		}
	}
	close(ch)
	w.Wait()
}

func TestRaceBool(t *testing.T) {
	var w sync.WaitGroup
	v := syncutil.NewGValue(false)
	ch := make(chan struct{})
	for i := 0; i < 200; i++ {
		w.Add(1)
		if i%2 == 0 {
			go func(i int) {
				defer w.Done()
				<-ch
				_ = v.Load()
				v.Store(true)
			}(i)
		} else {
			go func(i int) {
				defer w.Done()
				<-ch
				v.Swap(false)
			}(i)
		}
	}
	close(ch)
	w.Wait()
}
