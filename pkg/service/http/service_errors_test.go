package http_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/service/http"
)

func TestListenNoError(t *testing.T) {
	localhost := "127.0.0.1:0"
	ctx, cancel := context.WithCancel(context.Background())
	svc, err := New(
		WithContext(ctx),
		WithAddr(&localhost),
		WithName("listener-test"),
	)
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
	errch := make(chan error)
	go func() {
		errch <- svc.Serve()
	}()
	cancel()
	err = <-errch
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
}

func TestInitRouteErrors(t *testing.T) {
	localhost := "127.0.0.1:0"
	_, err := New(
		WithContext(context.Background()),
		WithAddr(&localhost),
		WithName("listener-test"),
		WithInitRoutesFn(func(ctx context.Context, r *mux.Router) error {
			return errors.New("errror")
		}),
	)
	if err == nil {
		t.Error("expected error go <nil>")
	}
}

func TestSamePort(t *testing.T) {
	localhost := "127.0.0.1:0"
	svc1, err := New(
		WithContext(context.Background()),
		WithAddr(&localhost),
		WithName("listener-test"),
		WithInitRoutesFn(func(ctx context.Context, r *mux.Router) error {
			r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {}).Schemes("http")
			return nil
		}),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	listener, err := net.Listen("tcp", localhost)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	addr := listener.Addr().String()
	go func() {
		if err := svc1.DoServe(listener); err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
	}()
	svc2, err := New(
		WithContext(context.Background()),
		WithAddr(&addr),
		WithName("listener-test"),
		WithInitRoutesFn(func(ctx context.Context, r *mux.Router) error {
			r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {}).Schemes("http")
			return nil
		}),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if err := svc2.Serve(); err == nil {
		t.Error("expected error got <nil>")
		return
	}
}
