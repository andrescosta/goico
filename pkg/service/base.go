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

type Setter func(*Base)

// Base provides common functionality for processes that run in background.
type Base struct {
	meta         *meta.Data
	Addr         *string
	OtelProvider *obs.OtelProvider
	Ctx          context.Context
	done         context.CancelFunc
}

var (
	ErrOtelStack  = errors.New("obs.New: error initializing otel stack")
	ErrEnvLoading = errors.New("env.Populate: error initializing otel stack")
)

func New(opts ...Setter) (*Base, error) {
	// Instantiate with default values
	svc := &Base{
		Ctx: context.Background(),
		meta: &meta.Data{
			StartTime: time.Now(),
		},
		OtelProvider: nil,
	}

	for _, opt := range opts {
		opt(svc)
	}

	// .env files loading
	if err := env.Load(svc.meta.Name); err != nil {
		if !errors.Is(err, env.ErrNoEnvFileLoaded) {
			return nil, errors.Join(err, ErrEnvLoading)
		}
	}
	// OS signal handling
	svc.Ctx, svc.done = signal.NotifyContext(svc.Ctx, syscall.SIGINT, syscall.SIGTERM)

	// log initialization
	logger := log.NewWithContext(map[string]string{"service": svc.meta.Name})
	svc.Ctx = logger.WithContext(svc.Ctx)

	if svc.Addr == nil {
		addrEnv := svc.meta.Name + ".addr"
		svc.Addr = env.StringOrNil(addrEnv)
	}

	// observability provider controlled by envs obs.*
	o, err := obs.New(svc.Ctx, *svc.meta)
	if err != nil {
		return nil, errors.Join(err, ErrOtelStack)
	}
	svc.OtelProvider = o

	go svc.waitForDoneAndEndTheWorld()

	return svc, nil
}

func (s *Base) Started() {
	s.meta.StartTime = time.Now()
}

func (s *Base) Stopped() {
	s.meta.StartTime = time.Time{}
}

func (s *Base) Name() string {
	return s.meta.Name
}

func (s *Base) WhenStarted() time.Time {
	return s.meta.StartTime
}

func (s *Base) Metadata() map[string]string {
	m := map[string]string{
		"Name":      s.meta.Name,
		"Addr":      *s.Addr,
		"StartTime": s.WhenStarted().Format(time.UnixDate),
		"Kind":      s.meta.Kind,
	}
	return m
}

// Waits for the done signal and stops dependant providers.
func (s *Base) waitForDoneAndEndTheWorld() {
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
	return func(s *Base) {
		s.meta = meta
	}
}

func WithName(name string) Setter {
	return func(s *Base) {
		s.meta.Name = name
	}
}

func WithKind(kind string) Setter {
	return func(s *Base) {
		s.meta.Kind = kind
	}
}

func WithAddr(addr *string) Setter {
	return func(s *Base) {
		s.Addr = addr
	}
}

func WithOtelProvider(p *obs.OtelProvider) Setter {
	return func(s *Base) {
		s.OtelProvider = p
	}
}

func WithContext(ctx context.Context) Setter {
	return func(s *Base) {
		s.Ctx = ctx
	}
}
