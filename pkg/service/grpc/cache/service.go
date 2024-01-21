package cache

import (
	"context"
	"errors"

	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/grpc"
	"github.com/andrescosta/goico/pkg/service/grpc/cache/event"
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

type Service struct {
	service *grpc.Service
}

func NewService[K comparable, V any](ctx context.Context, listener service.GrpcListener, cache *Cache[K, V]) (*Service, error) {
	svc, err := grpc.New(
		grpc.WithName("cache_"+cache.Name()),
		grpc.WithListener(listener),
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
	return &Service{
		service: svc,
	}, nil
}

func (s *Service) Serve() error {
	return s.service.Serve()
}

func NewCacheServiceClient(ctx context.Context, addr string, d service.GrpcDialer) (event.CacheServiceClient, error) {
	conn, err := d.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	client := event.NewCacheServiceClient(conn)
	return client, nil
}
