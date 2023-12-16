package service

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/andrescosta/goico/pkg/service/svcmeta"
	"github.com/rs/zerolog"
)

type ServiceSetter func(*Service)

// A Service is a process that runs in the background.
type Service struct {
	Name         string
	Kind         string
	meta         *svcmeta.Info
	Addr         *string
	startTime    time.Time
	OtelProvider *obs.OtelProvider
	Ctx          context.Context
	done         context.CancelFunc
}

var (
	ErrOtelStack  = errors.New("obs.New: error initializing otel stack")
	ErrEnvLoading = errors.New("env.Populate: error initializing otel stack")
)

func New(opts ...ServiceSetter) (*Service, error) {
	// Instantiate with default values
	svc := &Service{
		Name:         "",
		Kind:         "",
		Ctx:          context.Background(),
		Addr:         nil,
		meta:         nil,
		startTime:    time.Now(),
		OtelProvider: nil,
	}

	for _, opt := range opts {
		opt(svc)
	}

	// .env files loading
	if err := env.Populate(); err != nil {
		return nil, errors.Join(err, ErrEnvLoading)
	}

	// OS signal handling
	svc.Ctx, svc.done = signal.NotifyContext(svc.Ctx, syscall.SIGINT, syscall.SIGTERM)

	// log initialization
	logger := log.NewWithContext(map[string]string{"service": svc.Name})
	svc.Ctx = logger.WithContext(svc.Ctx)

	addrEnv := svc.Name + ".addr"
	svc.Addr = env.OrNil(addrEnv)

	// meta info
	metainfo := svcmeta.Info{Name: svc.Name, Version: "1", Kind: svc.Kind}

	// observability provider controlled by envs obs.*
	o, err := obs.New(svc.Ctx, metainfo)
	if err != nil {
		return nil, errors.Join(err, ErrOtelStack)
	}
	svc.OtelProvider = o

	go svc.waitForDoneAndEndTheWorld()

	return svc, nil
}

func (s *Service) Started() {
	s.startTime = time.Now()
}

func (s *Service) WhenStarted() time.Time {
	return s.startTime
}

func (s *Service) Metadata() map[string]string {
	m := map[string]string{"Name": s.Name,
		"Addr":       *s.Addr,
		"Start Time": s.WhenStarted().Format(time.UnixDate),
		"Kind":       s.Kind}
	return m
}

// Waits for the done signal and stops dependant providers.
func (s *Service) waitForDoneAndEndTheWorld() {
	defer s.done()

	logger := zerolog.Ctx(s.Ctx)
	logger.Debug().Msg("Service: waiting")
	<-s.Ctx.Done()

	logger.Debug().Msg("Service closing")
	shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	err := s.OtelProvider.Shutdown(shutdownCtx)
	if err != nil {
		e := collection.UnwrapError(err)
		logger.Warn().Errs(zerolog.ErrorFieldName, e).Msg("OtelProvider.Shutdown: error stopping providers")
	} else {
		logger.Debug().Msg("Service: stopped without errors")
	}
}

// Setters
func WithMetaInfo(meta *svcmeta.Info) ServiceSetter {
	return func(s *Service) {
		s.meta = meta
	}
}
func WithName(name string) ServiceSetter {
	return func(s *Service) {
		s.Name = name
	}
}

func WithKind(kind string) ServiceSetter {
	return func(s *Service) {
		s.Kind = kind
	}
}

func WithAddr(addr *string) ServiceSetter {
	return func(s *Service) {
		s.Addr = addr
	}
}

func WithOtelProvider(p *obs.OtelProvider) ServiceSetter {
	return func(s *Service) {
		s.OtelProvider = p
	}
}

func WithContext(ctx context.Context) ServiceSetter {
	return func(s *Service) {
		s.Ctx = ctx
	}
}
