package grpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/grpc/testing/echo"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/service/grpc"
)

func TestListenNoError(t *testing.T) {
	localhost := "127.0.0.1:0"
	ctx, cancel := context.WithCancel(context.Background())
	svc, err := New(
		WithListener(service.DefaultGrpcListener),
		WithName("echo"),
		WithAddr(&localhost),
		WithContext(ctx),
		WithServiceDesc(&echo.Echo_ServiceDesc),
		WithNewServiceFn(func(ctx context.Context) (any, error) {
			return echo.EchoServer(&ServerNop{}), nil
		}),
	)
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
	defer svc.Dispose()
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

type ServerNop struct {
	echo.UnimplementedEchoServer
}

func TestWithNewServiceFnError(t *testing.T) {
	localhost := "127.0.0.1:0"
	_, err := New(
		WithName("echo"),
		WithAddr(&localhost),
		WithContext(context.Background()),
		WithServiceDesc(&echo.Echo_ServiceDesc),
		WithNewServiceFn(func(ctx context.Context) (any, error) {
			return nil, errors.New("error creating service")
		}),
	)
	if err == nil {
		t.Errorf("expected error got <nil>")
	}
}

func TestSamePort(t *testing.T) {
	localhost := "127.0.0.1:9090"
	ctx, cancel := context.WithCancel(context.Background())
	svc1, err := New(
		WithName("echo"),
		WithListener(service.DefaultGrpcListener),
		WithAddr(&localhost),
		WithContext(ctx),
		WithServiceDesc(&echo.Echo_ServiceDesc),
		WithNewServiceFn(func(ctx context.Context) (any, error) {
			return echo.EchoServer(&ServerNop{}), nil
		}),
	)
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
	defer svc1.Dispose()
	errch := make(chan error)
	go func() {
		errch <- svc1.Serve()
		close(errch)
	}()
	time.Sleep(10 * time.Microsecond)
	svc2, err := New(
		WithName("echo"),
		WithAddr(&localhost),
		WithListener(service.DefaultGrpcListener),
		WithContext(ctx),
		WithServiceDesc(&echo.Echo_ServiceDesc),
		WithNewServiceFn(func(ctx context.Context) (any, error) {
			return echo.EchoServer(&ServerNop{}), nil
		}),
	)
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
	defer svc2.Dispose()
	errch2 := make(chan error)
	go func() {
		errch2 <- svc2.Serve()
		close(errch2)
	}()
	if nil == <-errch2 {
		t.Errorf("expected error got <nil>")
	}
	cancel()
	err = <-errch
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
}
