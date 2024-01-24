package test

import (
	"context"
	"errors"
	"sync"

	"github.com/andrescosta/goico/pkg/collection"
)

type Starter interface {
	Start(context.Context) error
}

type serviceCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
}

type ServiceGroup struct {
	w       *sync.WaitGroup
	ctxs    map[Starter]serviceCtx
	qerrors *collection.SyncQueue[error]
}

func NewServiceGroup() *ServiceGroup {
	return &ServiceGroup{
		w:       &sync.WaitGroup{},
		ctxs:    make(map[Starter]serviceCtx),
		qerrors: collection.NewQueue[error](),
	}
}

func (s *ServiceGroup) Start(services []Starter) {
	s.StartWithContext(context.Background(), services)
}

func (s *ServiceGroup) StartWithContext(ctx context.Context, services []Starter) {
	s.w.Add(len(services))
	for _, service := range services {
		ctx, cancel := context.WithCancel(ctx)
		ch := make(chan struct{})
		s.ctxs[service] = serviceCtx{ctx, cancel, ch}
		go func(service Starter) {
			defer s.w.Done()
			defer close(ch)
			if err := service.Start(ctx); err != nil {
				s.qerrors.Queue(err)
			}
		}(service)
	}
}

func (s *ServiceGroup) Errors() error {
	if s.qerrors.Size() > 0 {
		return errors.Join(s.qerrors.Slice()...)
	}
	return nil
}

func (s *ServiceGroup) ResetErrors() error {
	e := s.Errors()
	s.qerrors.Clear()
	return e
}

func (s *ServiceGroup) Stop() error {
	for _, v := range s.ctxs {
		v.cancel()
	}
	s.w.Wait()
	if s.qerrors.Size() > 0 {
		return errors.Join(s.qerrors.Slice()...)
	}
	return nil
}

func (s *ServiceGroup) StopService(st Starter) (<-chan struct{}, error) {
	c, ok := s.ctxs[st]
	if !ok {
		return nil, errors.New("service not found")
	}
	c.cancel()
	delete(s.ctxs, st)
	if s.qerrors.Size() > 0 {
		return c.stopCh, errors.Join(s.qerrors.Slice()...)
	}
	return c.stopCh, nil
}
