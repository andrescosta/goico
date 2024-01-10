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
	"github.com/andrescosta/goico/pkg/service/meta"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/rs/zerolog"
)

type Setter func(*ServiceBase)

// A ServiceBase is a common struct for processes that run in background.
type ServiceBase struct {
	Name         string
	Kind         string
	meta         *meta.Data
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

func New(opts ...Setter) (*ServiceBase, error) {
	// Instantiate with default values
	svc := &ServiceBase{
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
	if err := env.Load(svc.Name); err != nil {
		if !errors.Is(err, env.ErrNoEnvFileLoaded) {
			return nil, errors.Join(err, ErrEnvLoading)
		}
	}
	// OS signal handling
	svc.Ctx, svc.done = signal.NotifyContext(svc.Ctx, syscall.SIGINT, syscall.SIGTERM)

	// log initialization
	logger := log.NewWithContext(map[string]string{"service": svc.Name})
	svc.Ctx = logger.WithContext(svc.Ctx)

	if svc.Addr == nil {
		addrEnv := svc.Name + ".addr"
		svc.Addr = env.StringOrNil(addrEnv)
	}

	// metadata info
	metainfo := meta.Data{Name: svc.Name, Version: "1", Kind: svc.Kind}

	// observability provider controlled by envs obs.*
	o, err := obs.New(svc.Ctx, metainfo)
	if err != nil {
		return nil, errors.Join(err, ErrOtelStack)
	}
	svc.OtelProvider = o

	go svc.waitForDoneAndEndTheWorld()

	return svc, nil
}

func (s *ServiceBase) Started() {
	s.startTime = time.Now()
}

func (s *ServiceBase) WhenStarted() time.Time {
	return s.startTime
}

func (s *ServiceBase) Metadata() map[string]string {
	m := map[string]string{
		"Name":       s.Name,
		"Addr":       *s.Addr,
		"Start Time": s.WhenStarted().Format(time.UnixDate),
		"Kind":       s.Kind,
	}
	return m
}

// Waits for the done signal and stops dependant providers.
func (s *ServiceBase) waitForDoneAndEndTheWorld() {
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
func WithMetaInfo(meta *meta.Data) Setter {
	return func(s *ServiceBase) {
		s.meta = meta
	}
}

func WithName(name string) Setter {
	return func(s *ServiceBase) {
		s.Name = name
	}
}

func WithKind(kind string) Setter {
	return func(s *ServiceBase) {
		s.Kind = kind
	}
}

func WithAddr(addr *string) Setter {
	return func(s *ServiceBase) {
		s.Addr = addr
	}
}

func WithOtelProvider(p *obs.OtelProvider) Setter {
	return func(s *ServiceBase) {
		s.OtelProvider = p
	}
}

func WithContext(ctx context.Context) Setter {
	return func(s *ServiceBase) {
		s.Ctx = ctx
	}
}
