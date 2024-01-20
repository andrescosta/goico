package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	rpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	netConn             = NetConn{}
	DefaultGrpcDialer   = netConn
	DefaultGrpcListener = netConn
	DefaultHTTPListener = netConn
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
	TranportSetter interface {
		Set(addr string) error
	}
)

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

type BufConn struct {
	mu        *sync.Mutex
	listeners map[string]*bufconn.Listener
}

func NewBufConn() *BufConn {
	return &BufConn{
		listeners: make(map[string]*bufconn.Listener),
		mu:        &sync.Mutex{},
	}
}

func (t *BufConn) Dial(_ context.Context, addr string) (*rpc.ClientConn, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}
	l := t.listenerFor(addr)
	o := rpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) { return l.Dial() })
	c, err := rpc.Dial(addr, rpc.WithTransportCredentials(insecure.NewCredentials()), o)
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

func (t *BufConn) Set(addr string) error {
	if addr == "" {
		return ErrEmptyAddress
	}
	l := t.listenerFor(addr)
	http.DefaultTransport = &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return l.DialContext(context.Background())
		},
	}
	return nil
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
