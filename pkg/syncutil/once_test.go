package syncutil_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/andrescosta/goico/pkg/syncutil"
	"github.com/andrescosta/goico/pkg/test"
)

func TestDo(t *testing.T) {
	r := syncutil.NewOnceDisposable()
	ch := make(chan struct{})
	v := 0
	w := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			<-ch
			err := r.Do(context.Background(), func(_ context.Context) error {
				v++
				return nil
			})
			test.Nil(t, err)
		}()
	}
	close(ch)
	w.Wait()
	test.Equals(t, v, 1)
}

func TestDoDispose(t *testing.T) {
	init := syncutil.NewOnceDisposable()
	chDo := make(chan struct{})
	v := 0
	wDo := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDo.Add(1)
		go func() {
			defer wDo.Done()
			<-chDo
			err := init.Do(context.Background(), func(_ context.Context) error {
				v++
				return nil
			})
			test.Nil(t, err)
		}()
	}
	close(chDo)
	wDo.Wait()
	test.Equals(t, v, 1)
	chDispose := make(chan struct{})
	wDisponse := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDisponse.Add(1)
		go func() {
			defer wDisponse.Done()
			<-chDispose
			err := init.Dispose(context.Background(), func(_ context.Context) error {
				v++
				return nil
			})
			test.Nil(t, err)
		}()
	}
	close(chDispose)
	wDisponse.Wait()
	test.Equals(t, v, 2)
}

func TestDoError(t *testing.T) {
	init := syncutil.NewOnceDisposable()
	chDo := make(chan struct{})
	v := 0
	errDo := errors.New("error")
	wDo := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDo.Add(1)
		go func() {
			defer wDo.Done()
			<-chDo
			err := init.Do(context.Background(), func(_ context.Context) error {
				return errDo
			})
			test.ErrorIs(t, err, errDo)
		}()
	}
	close(chDo)
	wDo.Wait()
	chDispose := make(chan struct{})
	wDisponse := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDisponse.Add(1)
		go func() {
			defer wDisponse.Done()
			<-chDispose
			err := init.Dispose(context.Background(), func(_ context.Context) error {
				v++
				return nil
			})
			test.Nil(t, err)
		}()
	}
	close(chDispose)
	wDisponse.Wait()
	test.Equals(t, v, 1)
}

func TestDiposeError(t *testing.T) {
	init := syncutil.NewOnceDisposable()
	chDo := make(chan struct{})
	v := 0
	errDispose := errors.New("error")
	wDo := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDo.Add(1)
		go func() {
			defer wDo.Done()
			<-chDo
			err := init.Do(context.Background(), func(_ context.Context) error {
				v++
				return nil
			})
			test.Nil(t, err)
		}()
	}
	close(chDo)
	wDo.Wait()
	test.Equals(t, v, 1)
	chDispose := make(chan struct{})
	wDisponse := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDisponse.Add(1)
		go func() {
			defer wDisponse.Done()
			<-chDispose
			err := init.Dispose(context.Background(), func(_ context.Context) error {
				return errDispose
			})
			test.ErrorIs(t, err, errDispose)
		}()
	}
	close(chDispose)
	wDisponse.Wait()
}

func TestBothErrors(t *testing.T) {
	init := syncutil.NewOnceDisposable()
	chDo := make(chan struct{})
	errDispose := errors.New("error dispose")
	errDo := errors.New("error do")
	wDo := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDo.Add(1)
		go func() {
			defer wDo.Done()
			<-chDo
			err := init.Do(context.Background(), func(_ context.Context) error {
				return errDo
			})
			test.ErrorIs(t, err, errDo)
		}()
	}
	close(chDo)
	wDo.Wait()
	chDispose := make(chan struct{})
	wDisponse := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wDisponse.Add(1)
		go func() {
			defer wDisponse.Done()
			<-chDispose
			err := init.Dispose(context.Background(), func(_ context.Context) error {
				return errDispose
			})
			test.ErrorIs(t, err, errDispose)
		}()
	}
	close(chDispose)
	wDisponse.Wait()
}

func TestErrorCallDispose(t *testing.T) {
	r := syncutil.NewOnceDisposable()
	v := 0
	err := r.Dispose(context.Background(), func(_ context.Context) error {
		v++
		return nil
	})
	test.NotNil(t, err)
	test.Equals(t, v, 0)
}

func TestErrorCallDo(t *testing.T) {
	r := syncutil.NewOnceDisposable()
	v := 0
	err := r.Do(context.Background(), func(_ context.Context) error {
		v++
		return nil
	})
	test.Nil(t, err)
	err = r.Dispose(context.Background(), func(_ context.Context) error {
		v++
		return nil
	})
	test.Nil(t, err)
	test.Equals(t, v, 2)
	err = r.Do(context.Background(), func(_ context.Context) error {
		v++
		return nil
	})
	test.NotNil(t, err)
	test.Equals(t, v, 2)
}
