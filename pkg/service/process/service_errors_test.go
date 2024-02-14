package process_test

import (
	"context"
	"errors"
	"testing"

	"github.com/andrescosta/goico/pkg/service/process"
)

func TestErrorProcess(t *testing.T) {
	localhost := "127.0.0.1:0"
	proc, err := process.New(
		process.WithContext(context.Background()),
		process.WithName("executor"),
		process.WithAddr(localhost),
		process.WithHealthCheckFN(getHealthCheckHandlerR()),
		process.WithStarter(func(_ context.Context) error {
			return errors.New("process error")
		}),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	errorsvc := make(chan error)
	go func() {
		errorsvc <- proc.Serve()
	}()
	if nil == <-errorsvc {
		t.Errorf("expected error got <nil>")
	}
}

func getHealthCheckHandlerR() func(ctx context.Context) (map[string]string, error) {
	return func(_ context.Context) (map[string]string, error) {
		return map[string]string{
			"customer": "ERROR!",
			"identity": "OK",
			"database": "ERROR!",
		}, nil
	}
}
