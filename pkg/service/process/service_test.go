package process_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/http/httptest"
	"github.com/andrescosta/goico/pkg/service/process"
	"github.com/andrescosta/goico/pkg/test"
)

var debug = false

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
		exec(*testing.T, string)
		env() []string
	}

	config struct {
		envv []string
		name string
	}

	getMetadata struct {
		config
		kind    string
		enabled bool
	}
	healthCheck struct {
		config
		health health
	}
)

func Test(t *testing.T) {
	t.Parallel()
	run(t, []scenario{
		getMetadata{
			config: config{
				name: "metadata-enabled",
			},
			kind:    "headless",
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
	})
}

func (s getMetadata) exec(t *testing.T, url string) {
	url = url + "/meta"
	resp, err := Get(url)
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

func (s healthCheck) exec(t *testing.T, url string) {
	url = url + "/health"
	resp, err := Get(url)
	if err != nil {
		t.Errorf("unexpected error getting from server: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if !s.health.HasErrors && resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code of %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}
	if s.health.HasErrors && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code of %d, got %d", http.StatusOK, resp.StatusCode)
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

func run(t *testing.T, ss []scenario) {
	for _, s := range ss {
		t.Run(s.sname(), func(t *testing.T) {
			localhost := "127.0.0.1:0"
			b := env.Backup()
			t.Cleanup(func() {
				env.Restore(b)
			})
			setEnv(s.env())
			_, _, err := env.Load("echo")
			test.Nil(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			started := make(chan bool, 1)
			proc, err := process.New(
				process.WithContext(ctx),
				process.WithName("executor"),
				process.WithAddr(localhost),
				process.WithHealthCheckFN(getHealthCheckHandler(s)),
				process.WithStarter(func(ctx context.Context) error {
					started <- true
					<-ctx.Done()
					started <- false
					return nil
				}),
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				cancel()
				return
			}
			if !debug {
				log.DisableLog()
			}
			listener, err := net.Listen("tcp", localhost)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				cancel()
				return
			}
			url := "http://" + listener.Addr().String()
			errorsvc := make(chan error)
			go func() {
				errorsvc <- proc.DoServe(listener)
			}()
			st := <-started
			if !st {
				t.Errorf("not started")
			}
			s.exec(t, url)
			cancel()
			if nil != <-errorsvc {
				t.Errorf("unexpected error: %v", err)
			}
			st = <-started
			if st {
				t.Errorf("not shutdown")
			}
		})
	}
}

func (c config) sname() string { return c.name }
func (c config) env() []string { return c.envv }

func setEnv(e []string) {
	if e != nil {
		env.Setargs(e...)
		return
	}
	env.Setargs("metadata.enabled=true")
	httptest.SetHTTPServerTimeouts(1 * time.Second)
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

func decodejson(b []byte, d any) error {
	buffer := bytes.NewBuffer(b)
	if err := json.NewDecoder(buffer).Decode(&d); err != nil {
		return err
	}
	return nil
}

func Get(url string) (*http.Response, error) {
	return Verb(url, http.MethodGet, nil)
}

func Verb(url string, verb string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), verb, url, body)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}
