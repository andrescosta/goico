package http_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/andrescosta/goico/pkg/service/http/httptest"
)

type scenario struct {
	name     string
	handlers []httptest.PathHandler
	checker  func(*testing.T, *httptest.Service)
}

func TestGetCalls(t *testing.T) {
	scenarios := []scenario{
		{
			name: "test HTTP 200",
			handlers: []httptest.PathHandler{
				{
					Scheme: "http",
					Path:   "/",
					Handler: func(rw http.ResponseWriter, r *http.Request) {
						_, err := rw.Write([]byte("hello http world"))
						if err != nil {
							panic(fmt.Sprintf("Failed writing HTTP response: %v", err))
						}
					},
				},
			},
			checker: func(t *testing.T, s *httptest.Service) {
				validateGetCall(t, s, 200, "hello http world")
			},
		},
		{
			name: "test http 500",
			handlers: []httptest.PathHandler{
				{
					Scheme: "http",
					Path:   "/",
					Handler: func(rw http.ResponseWriter, r *http.Request) {
						http.Error(rw, "test error 500", http.StatusInternalServerError)
					},
				},
			},
			checker: func(t *testing.T, s *httptest.Service) {
				validateGetCall(t, s, 500, "test error 500")
			},
		},
	}

	runThem(t, scenarios)
}

func runThem(t *testing.T, scenarios []scenario) {
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(func() {
				cancel()
			})
			svc, err := httptest.NewService(ctx, s.handlers)
			if err != nil {
				t.Fatalf("Not expected error:%s", err)
				return
			}
			s.checker(t, svc)
		})
	}
}

func validateGetCall(t *testing.T, s *httptest.Service, statusCode int, body string) {
	resp, err := s.Client.Get(s.URL)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	if resp.StatusCode != statusCode {
		t.Errorf("expected a status code of %q, got %q", statusCode, resp.StatusCode)
		return
	}
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error reading body: %v", err)
		return
	}
	rbody = bytes.TrimSpace(rbody)
	if !bytes.Equal(rbody, []byte(body)) {
		t.Errorf("response should be '%s', was: '%s'", body, string(rbody))
		return
	}
}
