package process_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/process"
)

func TestErrorProcess(t *testing.T) {
	localhost := "127.0.0.1:0"
	proc, err := process.New(
		process.WithContext(context.Background()),
		process.WithSidecarListener(service.DefaultHTTPListener),
		process.WithName("executor"),
		process.WithAddr(&localhost),
		process.WithEnableSidecar(true),
		process.WithHealthCheckFN(getHealthCheckHandlerR()),
		process.WithServeHandler(func(ctx context.Context) error {
			return errors.New("process error")
		}),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	listener, err := net.Listen("tcp", localhost)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	errorsvc := make(chan error)
	go func() {
		errorsvc <- proc.DoServe(listener)
	}()
	if nil == <-errorsvc {
		t.Errorf("expected error got <nil>")
	}
}

func getHealthCheckHandlerR() func(ctx context.Context) (map[string]string, error) {
	return func(ctx context.Context) (map[string]string, error) {
		return map[string]string{
			"customer": "ERROR!",
			"identity": "OK",
			"database": "ERROR!",
		}, nil
	}
}
