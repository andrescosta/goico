package cache

import (
	"context"
	"errors"

	"github.com/andrescosta/goico/pkg/broadcaster"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/grpc"
	"github.com/andrescosta/goico/pkg/service/grpc/cache/event"
	"github.com/andrescosta/goico/pkg/service/grpc/stream"
	rpc "google.golang.org/grpc"
)

type server[K comparable, V any] struct {
	event.UnimplementedCacheServiceServer
	cache *Cache[K, V]
}

var ErrStopped = errors.New("cache channel stopped")

func (s *server[K, V]) Events(_ *event.Empty, in event.CacheService_EventsServer) error {
	l, err := s.cache.Subscribe()
	if err != nil {
		return err
	}
	for {
		select {
		case <-in.Context().Done():
			_ = s.cache.Unsubscribe(l)
			return in.Context().Err()
		case d, ok := <-l.C:
			if !ok {
				return ErrStopped
			}
			err := in.SendMsg(d)
			if err != nil {
				return err
			}
		}
	}
}

type (
	Setter  func(*Service)
	Service struct {
		grpc.Container
	}
)

const name = "cache_listener"

func NewService[K comparable, V any](ctx context.Context, cache *Cache[K, V], ops ...Setter) (*Service, error) {
	s := &Service{
		Container: grpc.Container{
			Name: name,
			GrpcConn: service.GrpcConn{
				Dialer:   service.DefaultGrpcDialer,
				Listener: service.DefaultGrpcListener,
			},
		},
	}
	for _, op := range ops {
		op(s)
	}

	svc, err := grpc.New(
		grpc.WithName("cache_"+cache.Name()),
		grpc.WithListener(s.Listener),
		grpc.WithAddr(s.AddrOrPanic()),
		grpc.WithContext(ctx),
		grpc.WithServiceDesc(&event.CacheService_ServiceDesc),
		grpc.WithNewServiceFn(func(ctx context.Context) (any, error) {
			return &server[K, V]{
				cache: cache,
			}, nil
		}),
	)
	if err != nil {
		return nil, err
	}
	s.Svc = svc
	return s, nil
}

func (s *Service) Serve() (err error) {
	defer s.Svc.Dispose()
	return s.Svc.Serve()
}

func (s *Service) Dispose() {
	s.Svc.Dispose()
}

type Client struct {
	serverAddr       string
	conn             *rpc.ClientConn
	client           event.CacheServiceClient
	broadcasterEvent *broadcaster.Broadcaster[*event.Event]
}

func NewClient(ctx context.Context, addr string, d service.GrpcDialer) (*Client, error) {
	conn, err := d.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	client := event.NewCacheServiceClient(conn)
	return &Client{
		serverAddr: addr,
		conn:       conn,
		client:     client,
	}, nil
}

func (c *Client) Close() error {
	var err error
	if c.broadcasterEvent != nil {
		err = errors.Join(c.broadcasterEvent.Stop())
	}
	return errors.Join(c.conn.Close(), err)
}

func (c *Client) ListenerForEvents(ctx context.Context) (*broadcaster.Listener[*event.Event], error) {
	if c.broadcasterEvent == nil {
		if err := c.startListenerForEvents(ctx); err != nil {
			return nil, err
		}
	}
	return c.broadcasterEvent.Subscribe()
}

func (c *Client) startListenerForEvents(ctx context.Context) error {
	cb := broadcaster.Start[*event.Event](ctx)
	c.broadcasterEvent = cb
	s, err := c.client.Events(ctx, &event.Empty{})
	if err != nil {
		return err
	}
	go func() {
		_ = stream.Recv(ctx, s, cb)
	}()
	return nil
}

func WithGrpcConn(g service.GrpcConn) Setter {
	return func(s *Service) {
		s.Container.GrpcConn = g
	}
}
