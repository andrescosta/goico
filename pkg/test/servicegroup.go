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
		HealthCheckClient() (service.HealthChecker, error)
	}

	activeService struct {
		stopCh  chan *ErrWhileStopping
		starter Starter
		Stopped bool
	}

	ServiceGroup struct {
		activeServices    map[Starter]*activeService
		httpClientBuilder service.HTTPClient
	}
)

var ErrTimeout = errors.New("timeout while waiting channel")

type (
	ErrNotHealthy struct {
		Addr string
	}

	ErrWhileStopping struct {
		Starter Starter
		Err     error
	}
)

func (e ErrNotHealthy) Error() string {
	return fmt.Sprintf("service at %s not healthy", e.Addr)
}

func (e ErrWhileStopping) Error() string {
	return fmt.Sprintf("error stopping %s: %v", e.Starter.Addr(), e.Err)
}

func NewServiceGroup(cliBuilder service.HTTPClient) *ServiceGroup {
	return &ServiceGroup{
		activeServices:    make(map[Starter]*activeService),
		httpClientBuilder: cliBuilder,
	}
}

func (s *ServiceGroup) Start(starters []Starter) error {
	for _, st := range starters {
		stt := st
		s.startStarter(stt)
	}
	return s.waitUntilHealthy(starters)
}

func (s *ServiceGroup) waitUntilHealthy(starters []Starter) error {
	var w sync.WaitGroup
	w.Add(len(starters))
	q := collection.NewQueue[error]()
	for _, st := range starters {
		go func(st Starter) {
			defer w.Done()
			err := s.waitForHealthy(st)
			if err != nil {
				q.Queue(err)
			}
		}(st)
	}
	w.Wait()
	return errors.Join(q.Slice()...)
}

func (s *ServiceGroup) startStarter(st Starter) {
	ch := make(chan *ErrWhileStopping)
	a := &activeService{ch, st, false}
	s.activeServices[st] = a
	go func() {
		defer close(a.stopCh)
		if err := a.starter.Start(); err != nil {
			ch <- &ErrWhileStopping{Starter: a.starter, Err: err}
		}
		a.Stopped = true
	}()
}

func (s *ServiceGroup) waitForHealthy(st Starter) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	cli, err := st.HealthCheckClient()
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

func (s *ServiceGroup) WaitToStop() error {
	errs := make([]error, 0)
	var w sync.WaitGroup
	w.Add(len(s.activeServices))
	for _, svc := range s.activeServices {
		go func(st *activeService) {
			defer w.Done()
			err := waitToStopService(st)
			if err != nil {
				errs = append(errs, err)
			}
		}(svc)
	}
	w.Wait()
	return errors.Join(errs...)
}

func (s *ServiceGroup) Stop(ctx context.Context, st Starter) error {
	_, ok := s.activeServices[st]
	if !ok {
		return nil
	}
	delete(s.activeServices, st)
	return nil
	// return waitToStopService(svc)
}

func waitToStopService(svc *activeService) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errStop, errTimeout := WaitFor(ctx, svc.stopCh)
	if errTimeout != nil {
		return ErrWhileStopping{Starter: svc.starter, Err: errTimeout}
	}
	if errStop != nil {
		return *errStop
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
