package http_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/service/http"
)

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
	go func() {
		if err := svc1.Serve(); err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
	}()
	addr := <-svc1.AddressReady
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
