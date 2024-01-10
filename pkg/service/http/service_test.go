package http_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/http/httptest"
)

type health struct {
	Status    string            `json:"status"`
	Details   map[string]string `json:"details"`
	hasErrors bool
}
type metadata struct {
	Addr      string
	Kind      string
	Name      string
	StartTime string
}

var debug bool = false

type scenario struct {
	name               string
	handlers           []httptest.PathHandler
	healthCheckHandler func(context.Context) (map[string]string, error)
	runner             func(*testing.T, *httptest.Service)
}

func TestGetCalls(t *testing.T) {
	scenarios := []scenario{
		getok(t),
		getinternalerror(t),
		recover(t),
		healthcheckerror(t),
		healthcheckok(t),
		metadataok(t),
		gettimeout(t),
	}
	run(t, scenarios)
}

func getok(t *testing.T) scenario {
	return get(t, http.StatusOK)
}
func getinternalerror(t *testing.T) scenario {
	return get(t, http.StatusInternalServerError)
}
func metadataok(t *testing.T) scenario {
	body := randomstring(20)
	return scenario{
		name: "Metadata",
		handlers: []httptest.PathHandler{
			{
				Scheme: "http",
				Path:   "/",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					dowrite(rw, r, http.StatusOK, body)
				},
			},
		},
		runner: func(t *testing.T, s *httptest.Service) {
			runmetadata(t, s, body)
		},
	}

}
func recover(t *testing.T) scenario {
	body := "mirror %s"
	return scenario{
		name: "recover",
		handlers: []httptest.PathHandler{
			{
				Scheme: "http",
				Path:   "/faulty",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					panic("ohhps something bad happened")
				},
			},
			{
				Scheme: "http",
				Path:   "/good",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					id := r.URL.Query().Get("id")
					dowrite(rw, r, http.StatusOK, fmt.Sprintf(body, id))
				},
			},
		},
		runner: func(t *testing.T, s *httptest.Service) {
			runfaulty(t, s, body)
		},
	}
}
func healthcheckok(t *testing.T) scenario {
	h := health{
		Status: "alive",
		Details: map[string]string{
			"customer": "OK",
			"identity": "OK",
			"database": "OK",
		},
		hasErrors: false,
	}
	return healthcheck(t, "Health check OK", h)
}
func healthcheckerror(t *testing.T) scenario {
	h := health{
		Status: "error",
		Details: map[string]string{
			"customer": "ERROR!",
			"identity": "OK",
			"database": "ERROR!",
		},
		hasErrors: true,
	}
	return healthcheck(t, "Health check Not OK", h)
}

func get(t *testing.T, statuscode int) scenario {
	body := randomstring(20)
	return scenario{
		name: fmt.Sprintf("test HTTP %d", statuscode),
		handlers: []httptest.PathHandler{
			{
				Scheme: "http",
				Path:   "/",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					dowrite(rw, r, statuscode, body)
				},
			},
		},
		runner: func(t *testing.T, s *httptest.Service) {
			runget(t, s, statuscode, body)
		},
	}
}
func healthcheck(t *testing.T, name string, h health) scenario {
	body := randomstring(10)
	return scenario{
		name: name,
		handlers: []httptest.PathHandler{
			{
				Scheme: "http",
				Path:   "/hchk",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					dowrite(rw, r, http.StatusOK, body)
				},
			},
		},
		healthCheckHandler: func(ctx context.Context) (map[string]string, error) {
			var err error
			if h.hasErrors {
				err = errors.New("Some errors")
			}
			return h.Details, err
		},
		runner: func(t *testing.T, s *httptest.Service) {
			runhealthcheck(t, s, h, body)
		},
	}
}

func gettimeout(t *testing.T) scenario {
	body := randomstring(10)
	return scenario{
		name: "timeout",
		handlers: []httptest.PathHandler{
			{
				Scheme: "http",
				Path:   "/timeout",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					time.Sleep(60 * time.Second)
					dowrite(rw, r, http.StatusOK, body)
				},
			},
		},
		runner: func(t *testing.T, s *httptest.Service) {
			rungettimeout(t, s, body)
		},
	}
}

func runget(t *testing.T, s *httptest.Service, statusCode int, body string) {
	resp, err := s.Client.Get(s.URL)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	expectresponse(t, resp, statusCode, body)
}

func runmetadata(t *testing.T, s *httptest.Service, body string) {
	w := &sync.WaitGroup{}
	start := make(chan struct{})
	for i := 0; i < 10; i++ {
		w.Add(1)
		go func(id string) {
			defer w.Done()
			<-start
			resp, err := s.Client.Get(s.URL + "/")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			expectresponse(t, resp, http.StatusOK, body)
		}(strconv.Itoa(i))
	}
	for i := 0; i < 1; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			<-start
			resp, err := s.Client.Get(s.URL + "/meta")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			expectresponsemetadata(resp, t)
		}()
	}
	close(start)
	w.Wait()
}

func expectresponsemetadata(resp *http.Response, t *testing.T) {
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected a status code of 200, got %d", resp.StatusCode)
		return
	}
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error reading body: %v", err)
		return
	}
	m := metadata{}
	if err := decodejson(rbody, &m); err != nil {
		t.Errorf("unexpected error decoding json %s", err)
		return
	}
	if m.Kind != "rest" {
		t.Errorf("expected Kind value 'rest' got %s", err)
		return
	}
	if m.Addr == "" {
		t.Errorf("expected Addr got <empty>")
		return
	}
	if m.Name == "" {
		t.Errorf("expected Name got <empty>")
		return
	}
	if m.StartTime == "" {
		t.Errorf("expected StartTime got <empty>")
		return
	}
}

func runfaulty(t *testing.T, s *httptest.Service, body string) {
	w := &sync.WaitGroup{}
	start := make(chan struct{})
	for i := 0; i < 100; i++ {
		w.Add(1)
		go func(id string) {
			defer w.Done()
			<-start
			resp, err := s.Client.Get(s.URL + "/good?id=" + id)
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			expectresponse(t, resp, http.StatusOK, fmt.Sprintf(body, id))
		}(strconv.Itoa(i))
	}
	for i := 0; i < 100; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			<-start
			resp, err := s.Client.Get(s.URL + "/faulty")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			expectresponse(t, resp, http.StatusInternalServerError, "")
		}()
	}
	close(start)
	w.Wait()
}

func rungettimeout(t *testing.T, s *httptest.Service, body string) {
	for i := 0; i < 10; i++ {
		go func(id string) {
			_, _ = s.Client.Get(s.URL + "/timeout")
		}(strconv.Itoa(i))
	}
	// we are giving some time to the go routine to call /timeout
	// before stopping the service
	time.Sleep(1 * time.Second)
}

func runhealthcheck(t *testing.T, s *httptest.Service, h health, body string) {
	w := &sync.WaitGroup{}
	start := make(chan struct{})
	for i := 0; i < 10; i++ {
		w.Add(1)
		go func(id string) {
			defer w.Done()
			<-start
			resp, err := s.Client.Get(s.URL + "/hchk")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			expectresponse(t, resp, http.StatusOK, body)
		}(strconv.Itoa(i))
	}
	for i := 0; i < 10; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			<-start
			resp, err := s.Client.Get(s.URL + "/health")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			expectresponsehelthcheck(h, resp, t)
		}()
	}
	close(start)
	w.Wait()
}

func expectresponsehelthcheck(h health, resp *http.Response, t *testing.T) {
	if !h.hasErrors && resp.StatusCode != http.StatusOK {
		t.Errorf("expected a status code of %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}
	if h.hasErrors && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected a status code of %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error reading body: %v", err)
		return
	}
	h2 := health{}
	if err := decodejson(rbody, &h2); err != nil {
		t.Errorf("unexpected error decoding json %s", err)
		return
	}
	// we set h2.hasErrors to h.hasErrors to avoid getting an error in DeepEqual
	h2.hasErrors = h.hasErrors
	if !reflect.DeepEqual(h, h2) {
		t.Errorf("different %v %v", h, h2)
		return
	}
}

func run(t *testing.T, scenarios []scenario) {
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			// t.Cleanup(func() {
			// 	cancel()
			// })
			os.Args = append(os.Args, "--env:metadata.enabled=true")
			svc, err := httptest.NewService(ctx, s.handlers, s.healthCheckHandler)
			if !debug {
				log.DisableLog()
			}
			if err != nil {
				t.Fatalf("Not expected error:%s", err)
				return
			}
			s.runner(t, svc)
			cancel()
			if err := <-svc.Servedone; err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
		})
	}
}

func dowrite(rw http.ResponseWriter, r *http.Request, statuscode int, body string) {
	rw.WriteHeader(statuscode)
	_, err := rw.Write([]byte(body))
	if err != nil {
		panic(fmt.Sprintf("Failed writing HTTP response: %v", err))
	}

}

func expectresponse(t *testing.T, resp *http.Response, statuscode int, body string) {
	if resp.StatusCode != statuscode {
		t.Errorf("expected a status code of %d, got %d", statuscode, resp.StatusCode)
		return
	}
	if body != "" || debug {
		rbody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
			return
		}
		if debug {
			fmt.Printf("body:\n%s\n", string(rbody))
		}
		rbody = bytes.TrimSpace(rbody)
		if body != "" {
			if !bytes.Equal(rbody, []byte(body)) {
				t.Errorf("response should be '%s', was: '%s'", body, string(rbody))
				return
			}
		}
	}
}

func randomstring(size int) string {
	rb := make([]byte, size)
	_, _ = rand.Read(rb)
	rs := base64.URLEncoding.EncodeToString(rb)
	return rs
}

func decodejson(b []byte, d any) error {
	buffer := bytes.NewBuffer(b)
	if err := json.NewDecoder(buffer).Decode(&d); err != nil {
		return err
	}
	return nil
}
