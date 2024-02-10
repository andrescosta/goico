package service

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/meta"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/rs/zerolog"
)

type Kind int

const (
	GRPC Kind = iota + 1
	HTTP
	Process
)

type Option func(*Base)

type ListenerFn func(ctx context.Context, addr string) (net.Listener, error)

// Base provides common functionality for processes that run in background.
type Base struct {
	meta         *meta.Data
	Addr         string
	OtelProvider *obs.OtelProvider
	Ctx          context.Context
	cancel       context.CancelFunc
	logWriter    io.WriteCloser
}

func New(opts ...Option) (*Base, error) {
	// Instantiate with default values
	svc := &Base{
		Ctx:          context.Background(),
		meta:         &meta.Data{},
		OtelProvider: nil,
	}

	for _, opt := range opts {
		opt(svc)
	}

	svc.Ctx, svc.cancel = context.WithCancel(svc.Ctx)

	// log initialization
	logger, logWriter := log.NewWithContext(map[string]string{"service": svc.meta.Name})
	svc.Ctx = logger.WithContext(svc.Ctx)
	svc.logWriter = logWriter

	// observability provider controlled by envs obs.*
	o, err := obs.New(svc.Ctx, *svc.meta)
	if err != nil {
		return nil, errors.Join(err, ErrOtelStack)
	}
	svc.OtelProvider = o

	go svc.waitForDoneAndEndTheWorld()

	return svc, nil
}

func (s *Base) Stop() {
	s.cancel()
}

func (s *Base) SetStartedNow() {
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
		"Addr":      s.Addr,
		"StartTime": s.WhenStarted().Format(time.UnixDate),
		"Kind":      s.meta.Kind,
	}
	return m
}

// Waits for the done signal and stops dependant providers.
func (s *Base) waitForDoneAndEndTheWorld() {
	defer s.cancel()

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
	if s.logWriter != nil {
		if err := s.logWriter.Close(); err != nil {
			println(err)
		}
	}
}

// Setters
func WithMetaInfo(meta *meta.Data) Option {
	return func(s *Base) {
		s.meta = meta
	}
}

func WithName(name string) Option {
	return func(s *Base) {
		s.meta.Name = name
	}
}

func WithKind(kind string) Option {
	return func(s *Base) {
		s.meta.Kind = kind
	}
}

func WithAddr(addr string) Option {
	return func(s *Base) {
		s.Addr = addr
	}
}

func WithOtelProvider(p *obs.OtelProvider) Option {
	return func(s *Base) {
		s.OtelProvider = p
	}
}

func WithContext(ctx context.Context) Option {
	return func(s *Base) {
		s.Ctx = ctx
	}
}
