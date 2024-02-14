package test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/service"
)

type (
	Starter interface {
		Start() error
		Addr() string
	}

	HealthCheckClientBuilder interface {
		NewHealthCheckClient() (service.HealthChecker, error)
	}

	activeService struct {
		stopCh  chan *ErrService
		starter Starter
		stopped bool
	}

	ServiceGroup struct {
		services map[Starter]*activeService
	}
)

var ErrTimeout = errors.New("timeout while waiting channel")

type (
	ErrNotHealthy struct {
		Addr string
	}

	ErrService struct {
		Starter Starter
		Err     error
	}
)

func (e ErrNotHealthy) Error() string {
	return fmt.Sprintf("service at %s not healthy", e.Addr)
}

func (e ErrService) Error() string {
	return fmt.Sprintf("error stopping %s: %v", e.Starter.Addr(), e.Err)
}

func NewServiceGroup() *ServiceGroup {
	return &ServiceGroup{
		services: make(map[Starter]*activeService),
	}
}

func (s *ServiceGroup) Start(starters ...Starter) error {
	for _, st := range starters {
		stt := st
		s.start(stt)
	}
	return s.waitUntilHealthy(starters)
}

func (s *ServiceGroup) waitUntilHealthy(starters []Starter) error {
	var w sync.WaitGroup
	w.Add(len(starters))
	q := collection.NewSyncQueue[error]()
	for _, st := range starters {
		go func(st Starter) {
			defer w.Done()
			err := s.waitForHealthy(st.(HealthCheckClientBuilder))
			if err != nil {
				q.Queue(err)
			}
		}(st)
	}
	w.Wait()
	return errors.Join(q.DequeueAll()...)
}

func (s *ServiceGroup) start(st Starter) {
	ch := make(chan *ErrService)
	a := &activeService{ch, st, false}
	s.services[st] = a
	go func() {
		defer close(a.stopCh)
		if err := a.starter.Start(); err != nil {
			ch <- &ErrService{Starter: a.starter, Err: err}
		}
		a.stopped = true
	}()
}

func (s *ServiceGroup) waitForHealthy(st HealthCheckClientBuilder) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	cli, err := st.NewHealthCheckClient()
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, cli.Close())
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err = cli.CheckOk(ctx)
			if err == nil {
				return nil
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
}

func (s *ServiceGroup) WaitUntilStopped() error {
	q := collection.NewSyncQueue[error]()
	var w sync.WaitGroup
	w.Add(len(s.services))
	for _, svc := range s.services {
		go func(st *activeService) {
			defer w.Done()
			err := waitUntilStopped(st)
			if err != nil {
				q.Queue(err)
			}
		}(svc)
	}
	w.Wait()
	return errors.Join(q.DequeueAll()...)
}

func waitUntilStopped(svc *activeService) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errStop, errTimeout := WaitFor(ctx, svc.stopCh)
	if errTimeout != nil {
		return ErrService{Starter: svc.starter, Err: errTimeout}
	}
	if errStop != nil {
		return errStop
	}
	return nil
}

func WaitForClosed[T any](ctx context.Context, ch <-chan T) error {
	_, err := WaitFor[T](ctx, ch)
	return err
}

func WaitFor[T any](ctx context.Context, ch <-chan T) (T, error) {
	select {
	case <-ctx.Done():
		var t T
		return t, ErrTimeout
	case t := <-ch:
		return t, nil
	}
}
