package http_test

import (
	"context"
	"testing"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/http/httptest"
)

func TestSidecarService(t *testing.T) {
	scenarios := []scenario{
		healthchecksidecarok(),
		healthchecksidecarerror(),
		getmetadatasidecar(),
	}
	runsidecar(t, scenarios)
}

func healthchecksidecarok() scenario {
	h := health{
		Status: "alive",
		Details: map[string]string{
			"workers_running": "10",
			"workers_stopped": "0",
		},
		hasErrors: false,
	}
	return healthcheck("healthcheckok", h)
}

func healthchecksidecarerror() scenario {
	h := health{
		Status: "error",
		Details: map[string]string{
			"workers_running": "5",
			"workers_stopped": "5",
		},
		hasErrors: true,
	}
	return healthcheck("healthcheckerror", h)
}

func getmetadatasidecar() scenario {
	return scenario{
		name: "getmetadatasidecar",
		runner: func(t *testing.T, s *httptest.Service) {
			runmetadata(t, "headless", s)
		},
	}
}

func runsidecar(t *testing.T, scenarios []scenario) {
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
			if !debug {
				log.DisableLog()
			}
			ctx, cancel := context.WithCancel(context.Background())
			svc, err := httptest.NewSidecar(ctx, s.healthCheckHandler)
			if err != nil {
				t.Fatalf("Not expected error:%s", err)
				return
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
