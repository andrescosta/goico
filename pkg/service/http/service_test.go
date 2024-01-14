package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	httpsvc "github.com/andrescosta/goico/pkg/service/http"
	"github.com/andrescosta/goico/pkg/service/http/httptest"
)

type (
	health struct {
		Status    string            `json:"status"`
		Details   map[string]string `json:"details"`
		HasErrors bool              `json:"-"`
	}
	metadata struct {
		Addr      string
		Kind      string
		Name      string
		StartTime string
	}
	scenario interface {
		sname() string
		exec(*testing.T, *httptest.Service)
		env() []string
	}
)

type (
	scenarioSidecar interface {
		isSidecar() bool
	}

	config struct {
		envv []string
		name string
	}

	echo struct {
		config
		body string
		verb string
		code int
	}

	upsPanic struct {
		config
		stackLevel httpsvc.StackLevel
	}
	upsTimeout struct {
		config
	}
	getMetadata struct {
		config
		kind    string
		enabled bool
		sidecar bool
	}
	healthCheck struct {
		config
		health  health
		sidecar bool
	}
)

var debug = false

const nobody = ""

var handlers = []httptest.PathHandler{
	{
		Scheme: "http",
		Path:   "/echo",
		Handler: func(rw http.ResponseWriter, r *http.Request) {
			c, err := strconv.Atoi(r.URL.Query().Get("code"))
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			message := r.URL.Query().Get("message")
			if message != "" {
				_, _ = rw.Write([]byte(message))
			}
			if c != http.StatusOK {
				rw.WriteHeader(c)
			}
		},
	},
	{
		Scheme: "http",
		Path:   "/panic",
		Handler: func(rw http.ResponseWriter, r *http.Request) {
			panic("panic!!!")
		},
	},
	{
		Scheme: "http",
		Path:   "/timeout",
		Handler: func(rw http.ResponseWriter, r *http.Request) {
			time.Sleep(1 * time.Minute)
			rw.WriteHeader(http.StatusOK)
		},
	},
}

func Test(t *testing.T) {
	t.Parallel()
	run(t, []scenario{
		echo{
			config: config{
				name: "get200",
			},
			body: "get200",
			verb: http.MethodGet,
			code: http.StatusOK,
		},
		echo{
			config: config{
				name: "post200",
			},
			body: "post200",
			verb: http.MethodPost,
			code: http.StatusOK,
		},
		echo{
			config: config{
				name: "get500",
			},
			body: nobody,
			verb: http.MethodGet,
			code: http.StatusInternalServerError,
		},
		echo{
			config: config{
				name: "post500",
			},
			body: nobody,
			verb: http.MethodPost,
			code: http.StatusInternalServerError,
		},
		upsPanic{
			config: config{
				name: "panic-full-stack",
			},
			stackLevel: httpsvc.StackLevelFullStack,
		},
		upsPanic{
			config: config{
				name: "panic-simple-stack",
			},
			stackLevel: httpsvc.StackLevelSimple,
		},
		upsTimeout{
			config: config{
				name: "timeout",
			},
		},
		getMetadata{
			config: config{
				name: "metadata-enabled",
			},
			kind:    "rest",
			enabled: true,
		},
		getMetadata{
			config: config{
				name: "metadata-disabled",
				// by default, it is enabled
				envv: []string{"metadata.enabled=false"},
			},
			enabled: false,
		},
		getMetadata{
			config: config{
				name: "metadata-sidecar-enabled",
			},
			kind:    "headless",
			enabled: true,
			sidecar: true,
		},
		healthCheck{
			config: config{
				name: "healthcheck-ok",
			},
			health: health{
				Details: map[string]string{
					"customer": "OK",
					"identity": "OK",
					"database": "OK",
				},
				HasErrors: false,
				Status:    "alive",
			},
		},
		healthCheck{
			config: config{
				name: "healcheck-error",
			},
			health: health{
				Details: map[string]string{
					"customer": "ERROR!",
					"identity": "OK",
					"database": "ERROR!",
				},
				HasErrors: true,
				Status:    "error",
			},
		},
		healthCheck{
			config: config{
				name: "healthcheck-sidecar-ok",
			},
			sidecar: true,
			health: health{
				Details: map[string]string{
					"workers_running": "10",
					"workers_stopped": "0",
				},
				HasErrors: false,
				Status:    "alive",
			},
		},
		healthCheck{
			config: config{
				name: "healcheck-sidecar-error",
			},
			sidecar: true,
			health: health{
				Details: map[string]string{
					"workers_running": "5",
					"workers_stopped": "5",
				},
				HasErrors: true,
				Status:    "error",
			},
		},
	},
	)
}

func (s echo) exec(t *testing.T, svc *httptest.Service) {
	url := fmt.Sprintf("%s/echo?message=%s&code=%d", svc.URL, s.body, s.code)
	resp, err := svc.Verb(url, s.verb, nil)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	validateResponse(t, resp, s.code, s.body)
}

func (upsPanic) exec(t *testing.T, svc *httptest.Service) {
	url := svc.URL + "/panic"
	respp, err := svc.Verb(url, http.MethodGet, nil)
	if err != nil {
		t.Errorf("not expected error: %v", err)
		return
	}
	defer func() { _ = respp.Body.Close() }()
	validateResponse(t, respp, http.StatusInternalServerError, nobody)
	url = svc.URL + "/echo?message=test&code=200"
	resp, err := svc.Get(url)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	validateResponse(t, resp, http.StatusOK, "test")
}

func (upsTimeout) exec(t *testing.T, svc *httptest.Service) {
	url := svc.URL + "/timeout"
	resp, err := svc.Get(url)
	if err == nil {
		t.Errorf("expected EOF got <nil>")
		return
	}
	defer func() {
		// body should always be null but the lint ...
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
}

func (s getMetadata) exec(t *testing.T, svc *httptest.Service) {
	url := svc.URL + "/meta"
	resp, err := svc.Get(url)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if !s.enabled {
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 getting %d", resp.StatusCode)
		}
		return
	}

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
	if m.Kind != s.kind {
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

func (s healthCheck) exec(t *testing.T, svc *httptest.Service) {
	url := svc.URL + "/health"
	resp, err := svc.Get(url)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if !s.health.HasErrors && resp.StatusCode != http.StatusOK {
		t.Errorf("expected a status code of %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}
	if s.health.HasErrors && resp.StatusCode != http.StatusInternalServerError {
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
	// we set h2.HasErrors to h.HasErrors to avoid getting an error in DeepEqual
	h2.HasErrors = s.health.HasErrors
	if !reflect.DeepEqual(s.health, h2) {
		t.Errorf("different %v %v", s.health, h2)
		return
	}
}

func validateResponse(t *testing.T, resp *http.Response, statuscode int, body string) {
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

// runner
func run(t *testing.T, ss []scenario) {
	for _, s := range ss {
		t.Run(s.sname(), func(t *testing.T) {
			b := env.Backup()
			t.Cleanup(func() {
				env.Restore(b)
			})
			setEnv(s.env())
			ctx, cancel := context.WithCancel(context.Background())
			svc, err := getService(ctx, s)
			if err != nil {
				t.Fatalf("Not expected error:%s", err)
				return
			}
			if !debug {
				log.DisableLog()
			}
			s.exec(t, svc)
			cancel()
			err = <-svc.Servedone
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func getService(ctx context.Context, s scenario) (*httptest.Service, error) {
	ss, ok := s.(scenarioSidecar)
	if !ok || !ss.isSidecar() {
		return httptest.NewService(ctx, getHandlers(s), getHealthCheckHandler(s), getStackLevel(s))
	}
	return httptest.NewSidecar(ctx, getHealthCheckHandler(s))
}

func getHandlers(s scenario) []httptest.PathHandler {
	switch s.(type) {
	case echo, upsPanic, upsTimeout:
		return handlers
	default:
		return nil
	}
}

func getStackLevel(s scenario) httpsvc.StackLevel {
	ss, ok := s.(upsPanic)
	if ok {
		return ss.stackLevel
	}
	return httpsvc.StackLevelSimple
}

func getHealthCheckHandler(s scenario) func(ctx context.Context) (map[string]string, error) {
	ss, ok := s.(healthCheck)
	if ok {
		return func(ctx context.Context) (map[string]string, error) {
			var err error
			if ss.health.HasErrors {
				err = errors.New("Some errors")
			}
			return ss.health.Details, err
		}
	}
	return nil
}

// impls

func (c config) sname() string        { return c.name }
func (c config) env() []string        { return c.envv }
func (s healthCheck) isSidecar() bool { return s.sidecar }
func (s getMetadata) isSidecar() bool { return s.sidecar }

//
// helpers
//

func decodejson(b []byte, d any) error {
	buffer := bytes.NewBuffer(b)
	if err := json.NewDecoder(buffer).Decode(&d); err != nil {
		return err
	}
	return nil
}

func setEnv(e []string) {
	if e != nil {
		httptest.SetArgs(e)
		return
	}
	httptest.SetArgs([]string{"metadata.enabled=true"})
	httptest.SetHTTPServerTimeouts(1 * time.Second)
}
