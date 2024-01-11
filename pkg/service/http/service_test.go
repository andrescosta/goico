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
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	httpsvc "github.com/andrescosta/goico/pkg/service/http"
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

var debug = false

const nobody = ""

type scenario struct {
	name               string
	handlers           []httptest.PathHandler
	healthCheckHandler func(context.Context) (map[string]string, error)
	runner             func(*testing.T, *httptest.Service)
	setenv             func()
	stackLevel         httpsvc.StackLevel
}

func TestService(t *testing.T) {
	scenarios := []scenario{
		get200(),
		get500(),
		post200(),
		post500(),
		recoverfrompanic(httpsvc.StackLevelFullStack),
		recoverfrompanic(httpsvc.StackLevelSimple),
		healthcheckerror(),
		healthcheckok(),
		disabledhealthcheck(),
		getmetadata(),
		disabledmetadata(),
		gettimeout(),
	}
	run(t, scenarios)
}

// Scenarios definition
func get200() scenario {
	return verb(http.MethodGet, http.StatusOK)
}

func post200() scenario {
	return verb(http.MethodPost, http.StatusOK)
}

func get500() scenario {
	return verb(http.MethodGet, http.StatusInternalServerError)
}

func post500() scenario {
	return verb(http.MethodPost, http.StatusInternalServerError)
}

func verb(verb string, statuscode int) scenario {
	body := randomstring(20)
	return scenario{
		name:     fmt.Sprintf("%s%d", verb, statuscode),
		handlers: handlerbody("/", body, statuscode),
		runner: func(t *testing.T, s *httptest.Service) {
			validateverb(t, s, verb, statuscode, body)
		},
	}
}

func getmetadata() scenario {
	return scenario{
		name: "getmetadata",
		runner: func(t *testing.T, s *httptest.Service) {
			runmetadata(t, "rest", s)
		},
	}
}

func disabledmetadata() scenario {
	return scenario{
		name:     "disabledmetadata",
		handlers: handlerbody("/", "??", http.StatusOK),
		runner: func(t *testing.T, s *httptest.Service) {
			validateurl(t, s, s.URL+"/meta", http.MethodGet, http.StatusNotFound, nobody)
		},
		setenv: func() {
			httptest.MetadataOff()
		},
	}
}

func recoverfrompanic(stackLevel httpsvc.StackLevel) scenario {
	body := "mirror %s"
	return scenario{
		name:       "recoverfrompanic",
		stackLevel: stackLevel,
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
					dowrite(rw, http.StatusOK, fmt.Sprintf(body, id))
				},
			},
		},
		runner: func(t *testing.T, s *httptest.Service) {
			runfaulty(t, s, body)
		},
	}
}

func healthcheckok() scenario {
	h := health{
		Status: "alive",
		Details: map[string]string{
			"customer": "OK",
			"identity": "OK",
			"database": "OK",
		},
		hasErrors: false,
	}
	return healthcheck("healthcheckok", h)
}

func healthcheckerror() scenario {
	h := health{
		Status: "error",
		Details: map[string]string{
			"customer": "ERROR!",
			"identity": "OK",
			"database": "ERROR!",
		},
		hasErrors: true,
	}
	return healthcheck("healthcheckerror", h)
}

func healthcheck(name string, h health) scenario {
	return scenario{
		name: name,
		healthCheckHandler: func(ctx context.Context) (map[string]string, error) {
			var err error
			if h.hasErrors {
				err = errors.New("Some errors")
			}
			return h.Details, err
		},
		runner: func(t *testing.T, s *httptest.Service) {
			runhealthcheck(t, s, h)
		},
	}
}

func disabledhealthcheck() scenario {
	return scenario{
		name: "disabledhealthcheck",
		runner: func(t *testing.T, s *httptest.Service) {
			validateurl(t, s, s.URL+"/health", http.MethodGet, http.StatusNotFound, nobody)
		},
	}
}

func gettimeout() scenario {
	body := randomstring(10)
	return scenario{
		name: "gettimeout",
		handlers: []httptest.PathHandler{
			{
				Scheme: "http",
				Path:   "/timeout",
				Handler: func(rw http.ResponseWriter, r *http.Request) {
					time.Sleep(60 * time.Second)
					dowrite(rw, http.StatusOK, body)
				},
			},
		},
		runner: func(t *testing.T, s *httptest.Service) {
			rungettimeout(t, s)
		},
		setenv: func() {
			httptest.SetArgs("http.shutdown.timeout", (1 * time.Second).String())
		},
	}
}

//
// Scenarios execution
//

func runmetadata(t *testing.T, kind string, s *httptest.Service) {
	resp, err := s.Get(s.URL + "/meta")
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	validatemetadata(t, kind, resp)
}

func runfaulty(t *testing.T, s *httptest.Service, body string) {
	w := &sync.WaitGroup{}
	start := make(chan struct{})
	for i := 0; i < 100; i++ {
		w.Add(1)
		go func(id string) {
			defer w.Done()
			<-start
			resp, err := s.Get(s.URL + "/good?id=" + id)
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			defer func() { _ = resp.Body.Close() }()
			validateresponse(t, resp, http.StatusOK, fmt.Sprintf(body, id))
		}(strconv.Itoa(i))
	}
	for i := 0; i < 100; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			<-start
			resp, err := s.Get(s.URL + "/faulty")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			defer func() { _ = resp.Body.Close() }()
			validateresponse(t, resp, http.StatusInternalServerError, nobody)
		}()
	}
	close(start)
	w.Wait()
}

func rungettimeout(t *testing.T, s *httptest.Service) {
	for i := 0; i < 10; i++ {
		go func(id string) {
			resp, err := s.Get(s.URL + "/timeout")
			if err == nil {
				_ = resp.Body.Close()
			}
			if resp.StatusCode == http.StatusOK {
				t.Errorf("expected status code 504")
			}
		}(strconv.Itoa(i))
	}
	// we are giving some time to call /timeout
	// before returning and stopping the service
	time.Sleep(1 * time.Second)
}

func runhealthcheck(t *testing.T, s *httptest.Service, h health) {
	w := &sync.WaitGroup{}
	start := make(chan struct{})
	for i := 0; i < 1; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			<-start
			resp, err := s.Get(s.URL + "/health")
			if err != nil {
				t.Errorf("unexpected error getting from server: %v", err)
				return
			}
			defer func() { _ = resp.Body.Close() }()
			validatehelthcheck(h, resp, t)
		}()
	}
	close(start)
	w.Wait()
}

//
// Scenarios result validation
//

func validateverb(t *testing.T, s *httptest.Service, verb string, statusCode int, body string) {
	validateurl(t, s, s.URL, verb, statusCode, body)
}

func validateurl(t *testing.T, s *httptest.Service, url string, verb string, statusCode int, body string) {
	resp, err := s.Verb(url, verb, nil)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	validateresponse(t, resp, statusCode, body)
}

func validatemetadata(t *testing.T, kind string, resp *http.Response) {
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
	if m.Kind != kind {
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

func validatehelthcheck(h health, resp *http.Response, t *testing.T) {
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

func validateresponse(t *testing.T, resp *http.Response, statuscode int, body string) {
	if resp.StatusCode != statuscode {
		t.Errorf("expected a status code of %d, got %d", statuscode, resp.StatusCode)
		return
	}
	if body != nobody || debug {
		rbody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
			return
		}
		if debug {
			fmt.Printf("body:\n%s\n", string(rbody))
		}
		rbody = bytes.TrimSpace(rbody)
		if body != nobody {
			if !bytes.Equal(rbody, []byte(body)) {
				t.Errorf("response should be '%s', was: '%s'", body, string(rbody))
				return
			}
		}
	}
}

//
// scenarios runner
//

func run(t *testing.T, scenarios []scenario) {
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			b := env.Backup()
			t.Cleanup(func() {
				env.Restore(b)
			})
			if s.setenv != nil {
				s.setenv()
			} else {
				setdefaultenv()
			}
			ctx, cancel := context.WithCancel(context.Background())
			svc, err := httptest.NewService(ctx, s.handlers, s.healthCheckHandler, s.stackLevel)
			if err != nil {
				t.Fatalf("Not expected error:%s", err)
				return
			}
			if !debug {
				log.DisableLog()
			}
			s.runner(t, svc)
			cancel()
			err = <-svc.Servedone
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

//
// helpers
//

func handlerbody(path string, body string, statuscode int) []httptest.PathHandler {
	return []httptest.PathHandler{
		{
			Scheme: "http",
			Path:   path,
			Handler: func(rw http.ResponseWriter, r *http.Request) {
				dowrite(rw, statuscode, body)
			},
		},
	}
}

func dowrite(rw http.ResponseWriter, statuscode int, body string) {
	rw.WriteHeader(statuscode)
	_, err := rw.Write([]byte(body))
	if err != nil {
		panic(fmt.Sprintf("Failed writing HTTP response: %v", err))
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

func setdefaultenv() {
	httptest.MetadataOn()
	httptest.SetHTTPServerTimeouts(1 * time.Second)
}
