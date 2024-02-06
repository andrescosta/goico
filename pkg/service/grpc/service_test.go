package grpc_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/grpc"
	"github.com/andrescosta/goico/pkg/service/grpc/svcmeta"
	"github.com/andrescosta/goico/pkg/service/grpc/testing/echo"
	"github.com/andrescosta/goico/pkg/service/http/httptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	status "google.golang.org/grpc/status"
)

var debug = false

type (
	scenario interface {
		sname() string
		exec(context.Context, *testing.T, *echo.Service, echo.EchoClient)
		env() []string
		server() *Server
		healthCheckFn() grpc.HealthCheckFn
	}

	config struct {
		envv           []string
		grpcserver     *Server
		healthCheckFnn grpc.HealthCheckFn
		name           string
	}
)

type (
	echos struct {
		config
		body string
		code uint32
	}

	upsPanic struct {
		config
	}
	upsTimeout struct {
		config
	}
	getMetadata struct {
		config
		kind    string
		enabled bool
	}
	healthCheck struct {
		config
		wait         time.Duration
		nocheck      bool
		returnsError bool
	}
)

type Server struct {
	closed   bool
	echofn   func(*echo.EchoRequest) (*echo.EchoResponse, error)
	noechofn func(*echo.EchoRequest) (*echo.Void, error)
	echo.UnimplementedEchoServer
}

func (s *Server) Close() error {
	s.closed = true
	return nil
}

var (
	serverEcho = &Server{
		echofn: func(er *echo.EchoRequest) (*echo.EchoResponse, error) {
			if codes.Code(er.Code) != codes.OK {
				return nil, status.Errorf(codes.Code(er.Code), er.Message)
			}
			return &echo.EchoResponse{Code: er.Code, Message: er.Message}, nil
		},
		noechofn: func(er *echo.EchoRequest) (*echo.Void, error) {
			if codes.Code(er.Code) != codes.OK {
				return nil, status.Errorf(codes.Code(er.Code), er.Message)
			}
			return &echo.Void{}, nil
		},
	}
	serverPanic = &Server{
		echofn: func(er *echo.EchoRequest) (*echo.EchoResponse, error) {
			return &echo.EchoResponse{Code: er.Code, Message: er.Message}, nil
		},
		noechofn: func(er *echo.EchoRequest) (*echo.Void, error) {
			panic(er.Message)
		},
	}
	serverTimeout = &Server{
		echofn: func(er *echo.EchoRequest) (*echo.EchoResponse, error) {
			time.Sleep(1 * time.Minute)
			return &echo.EchoResponse{Code: er.Code, Message: er.Message}, nil
		},
		noechofn: func(er *echo.EchoRequest) (*echo.Void, error) {
			return &echo.Void{}, nil
		},
	}
)

func (s *Server) Echo(_ context.Context, e *echo.EchoRequest) (*echo.EchoResponse, error) {
	return s.echofn(e)
}

func (s *Server) NoEcho(_ context.Context, e *echo.EchoRequest) (*echo.Void, error) {
	return s.noechofn(e)
}

func Test(t *testing.T) {
	t.Parallel()
	run(t, []scenario{
		echos{
			config: config{
				name:       "get200",
				grpcserver: serverEcho,
			},
			body: "get200",
			code: uint32(codes.OK),
		},
		echos{
			config: config{
				name:       "invalid_argument",
				grpcserver: serverEcho,
			},
			body: "Invalid Argument",
			code: uint32(codes.InvalidArgument),
		},
		upsPanic{
			config: config{
				name:       "upsPanic",
				grpcserver: serverPanic,
			},
		},
		upsTimeout{
			config: config{
				name:       "timeout",
				grpcserver: serverTimeout,
			},
		},
		getMetadata{
			kind:    "grpc",
			enabled: true,
			config: config{
				name:       "metadataenabled",
				grpcserver: serverEcho,
				envv:       []string{"metadata.enabled=true"},
			},
		},
		getMetadata{
			kind:    "grpc",
			enabled: false,
			config: config{
				name:       "metadatadisabled",
				grpcserver: serverEcho,
				envv:       []string{"metadata.enabled=false"},
			},
		},
		healthCheck{
			returnsError: false,
			config: config{
				name:           "healthcheck",
				healthCheckFnn: func(context.Context) error { return nil },
				grpcserver:     serverEcho,
				envv:           []string{fmt.Sprintf("grpc.healthcheck=%s", 1*time.Microsecond)},
			},
		},
		healthCheck{
			returnsError: true,
			wait:         50 * time.Microsecond,
			nocheck:      true,
			config: config{
				name: "healthcheck+-",
				healthCheckFnn: func(context.Context) error {
					if randomInt(2) == 1 {
						return nil
					}
					return errors.New("db error")
				},
				grpcserver: serverEcho,
				envv:       []string{fmt.Sprintf("grpc.healthcheck=%s", 1*time.Microsecond)},
			},
		},
		healthCheck{
			returnsError: true,
			config: config{
				name:           "healthcheckerror",
				healthCheckFnn: func(context.Context) error { return errors.New("db error") },
				grpcserver:     serverEcho,
				envv:           []string{fmt.Sprintf("grpc.healthcheck=%s", 1*time.Microsecond)},
			},
		},
	})
}

func (s getMetadata) exec(ctx context.Context, t *testing.T, svc *echo.Service, c echo.EchoClient) {
	if _, err := c.NoEcho(ctx, &echo.EchoRequest{Code: 0, Message: "hello"}); err != nil {
		t.Errorf("not expected error:%v", err)
	}
	cli, err := svc.InfoClient(ctx)
	if err != nil {
		t.Errorf("not expected error:%v", err)
	}
	info, err := cli.InfoAsMap(ctx, &svcmeta.GrpcMetadataRequest{Service: svc.Name()})
	if !s.enabled {
		if err == nil {
			t.Error("expected error got <nil>")
		}
		return
	}
	if s.enabled && err != nil {
		t.Errorf("not expected error:%v", err)
		return
	}
	if info["Kind"] != s.kind {
		t.Errorf("expected %s got %s", s.kind, info["Kind"])
	}
	if info["Name"] != svc.Name() {
		t.Errorf("expected %s got %s", svc.Name(), info["Name"])
	}
	if svc.Addr() != "" && info["Addr"] != svc.Addr() {
		t.Errorf("expected %s got %s", svc.Addr(), info["Addr"])
	}
	_, ok := info["StartTime"]
	if !ok {
		t.Error("expected StartTime go <nil>")
	}
}

func (s upsTimeout) exec(_ context.Context, t *testing.T, _ *echo.Service, c echo.EchoClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.Echo(ctx, &echo.EchoRequest{Code: 0, Message: "ups"})
	if err == nil {
		t.Error("expected error got <nil>")
	}
}

func (s echos) exec(ctx context.Context, t *testing.T, _ *echo.Service, c echo.EchoClient) {
	r, err := c.Echo(ctx, &echo.EchoRequest{Code: s.code, Message: s.body})
	if s.code != uint32(codes.OK) {
		if err == nil {
			t.Error("expected error got <nil>")
		}
		return
	}
	if err != nil {
		t.Errorf("not expected error:%v", err)
		return
	}
	if r.Message != s.body {
		t.Errorf("expected %s got %s", r.Message, "hello")
	}
	if r.Code != s.code {
		t.Errorf("expected %d got %d", r.Code, 1)
	}
	if _, err = c.NoEcho(ctx, &echo.EchoRequest{Code: s.code, Message: s.body}); err != nil {
		t.Errorf("not expected error:%v", err)
	}
}

func (s upsPanic) exec(ctx context.Context, t *testing.T, _ *echo.Service, c echo.EchoClient) {
	if _, err := c.NoEcho(ctx, &echo.EchoRequest{Code: 1, Message: "hello"}); err == nil {
		t.Errorf("expected error got <nil>")
	}
	_, err := c.Echo(ctx, &echo.EchoRequest{Code: 0, Message: "notpanic"})
	if err != nil {
		t.Errorf("not expected error:%v", err)
	}
}

func (s healthCheck) exec(ctx context.Context, t *testing.T, svc *echo.Service, c echo.EchoClient) {
	if _, err := c.NoEcho(ctx, &echo.EchoRequest{Code: 0, Message: "hello"}); err != nil {
		t.Errorf("not expected error:%v", err)
	}
	if s.wait != 0 {
		time.Sleep(s.wait)
	}
	if !s.nocheck {
		hc, err := svc.HealthCheckClient(ctx, svc.Name())
		if err != nil {
			t.Errorf("not expected error: %v", err)
			return
		}
		h, err := hc.Check(ctx)
		if err != nil {
			t.Errorf("not expected error: %v", err)
			return
		}
		if !s.returnsError && *h.Enum() != *grpc_health_v1.HealthCheckResponse_SERVING.Enum() {
			t.Errorf("expected SERVING got: %s", h.String())
		}
		if s.returnsError && *h.Enum() != *grpc_health_v1.HealthCheckResponse_NOT_SERVING.Enum().Enum() {
			t.Errorf("expected NOT SERVING got: %s", h.String())
		}
	}
}

func run(t *testing.T, ss []scenario) {
	for _, s := range ss {
		t.Run(s.sname(), func(t *testing.T) {
			errch := make(chan error)
			b := env.Backup()
			t.Cleanup(func() {
				env.Restore(b)
			})
			setEnv(s.env())
			ctx, cancel := context.WithCancel(context.Background())
			svc, err := echo.NewWithServer(ctx, s.server(), s.healthCheckFn())
			if err != nil {
				t.Errorf("not expected error:%v", err)
			}
			go func() {
				errch <- svc.Serve()
			}()
			if !debug {
				log.DisableLog()
			}
			c, err := svc.Client(ctx)
			if err != nil {
				t.Errorf("not expected error:%v", err)
			}
			s.exec(ctx, t, svc, c)
			cancel()
			err = <-errch
			if err != nil {
				t.Errorf("not expected error:%v", err)
			}
			svc.Dispose()
			if !s.server().closed {
				t.Error("server not closed")
			}
		})
	}
}

// impls
func (c config) server() *Server                   { return c.grpcserver }
func (c config) sname() string                     { return c.name }
func (c config) env() []string                     { return c.envv }
func (c config) healthCheckFn() grpc.HealthCheckFn { return c.healthCheckFnn }

func setEnv(e []string) {
	if e != nil {
		env.Setargs(e...)
		return
	}
	env.Setargs("metadata.enabled=true")
	httptest.SetHTTPServerTimeouts(1 * time.Second)
}

func randomInt(max int) int {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(i.Uint64())
}
