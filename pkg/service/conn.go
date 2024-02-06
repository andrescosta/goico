package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	rpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	netConn             = NetConn{}
	DefaultGrpcDialer   = netConn
	DefaultGrpcListener = netConn
	DefaultHTTPListener = netConn
	DefaultHTTPClient   = netConn
)

var ErrEmptyAddress = errors.New("address is empty")

type (
	GrpcDialer interface {
		Dial(ctx context.Context, addr string) (*rpc.ClientConn, error)
	}
	GrpcListener interface {
		Listen(addr string) (net.Listener, error)
	}
	HTTPListener interface {
		Listen(addr string) (net.Listener, error)
	}
	HTTPTranporter interface {
		Tranport(addr string) (*http.Transport, error)
	}
	HTTPClient interface {
		NewHTTPClient(addr string) (*http.Client, error)
	}
)

type GrpcConn struct {
	Dialer   GrpcDialer
	Listener GrpcListener
}

type HTTPConn struct {
	ClientBuilder HTTPClient
	Listener      HTTPListener
}

func (s HTTPConn) ClientBuilderOrDefault() HTTPClient {
	if s.ClientBuilder == nil {
		return DefaultHTTPClient
	}
	return s.ClientBuilder
}

func (s HTTPConn) ListenerOrDefault() HTTPListener {
	if s.Listener == nil {
		return DefaultHTTPListener
	}
	return s.Listener
}

type NetConn struct{}

func (NetConn) Dial(_ context.Context, addr string) (*rpc.ClientConn, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}
	c, err := rpc.Dial(addr, rpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (NetConn) Listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func (NetConn) NewHTTPClient(_ string) (*http.Client, error) {
	transport := http.DefaultTransport
	return &http.Client{
		Timeout:   1 * time.Second,
		Transport: transport,
	}, nil
}

type BufConn struct {
	timeout   time.Duration
	mu        *sync.Mutex
	listeners map[string]*bufconn.Listener
}

func NewBufConn() *BufConn {
	return NewBufConnWithTimeout(*env.Duration("dial.timeout"))
}

func NewBufConnWithTimeout(timeout time.Duration) *BufConn {
	return &BufConn{
		timeout:   timeout,
		listeners: make(map[string]*bufconn.Listener),
		mu:        &sync.Mutex{},
	}
}

func (t *BufConn) Dial(_ context.Context, addr string) (*rpc.ClientConn, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}
	l := t.listenerFor(addr)
	ctxDialerOp := rpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) { return l.DialContext(ctx) })
	timeOutOp := rpc.WithUnaryInterceptor(
		timeout.UnaryClientInterceptor(t.timeout),
	)
	c, err := rpc.Dial(addr, rpc.WithTransportCredentials(insecure.NewCredentials()), ctxDialerOp, timeOutOp)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (t *BufConn) Listen(addr string) (net.Listener, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}
	return t.listenerFor(addr), nil
}

func (t *BufConn) Tranport(addr string) (*http.Transport, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}
	l := t.listenerFor(addr)
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return l.DialContext(context.Background())
		},
	}, nil
}

func (t *BufConn) listenerFor(addr string) *bufconn.Listener {
	t.mu.Lock()
	defer t.mu.Unlock()
	l, ok := t.listeners[addr]
	if !ok {
		l = bufconn.Listen(1000)
		t.listeners[addr] = l
	}
	return l
}

func (t *BufConn) CloseAll() {
	for _, v := range t.listeners {
		_ = v.Close()
	}
}

func (t *BufConn) Close(addr string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	l, ok := t.listeners[addr]
	if ok {
		delete(t.listeners, addr)
		return l.Close()
	}
	return nil
}

func (t *BufConn) NewHTTPClient(addr string) (*http.Client, error) {
	transport, err := t.Tranport(addr)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout:   t.timeout,
		Transport: transport,
	}, nil
}
