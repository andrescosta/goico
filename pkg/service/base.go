package service

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
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

type StatusType int

const (
	StatusStopped = iota + 1
	StatusStarted
	StatusStarting
)

var statusString = map[StatusType]string{
	StatusStopped:  "stopped",
	StatusStarted:  "started",
	StatusStarting: "starting",
}

type Option func(*Base)

type ListenerFn func(ctx context.Context, addr string) (net.Listener, error)

// Base provides common functionality for processes that run in background.
type Base struct {
	muxStatus    sync.RWMutex
	Meta         *meta.Data
	startTime    time.Time
	status       StatusType
	HnUpTime     string
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
		Meta:         &meta.Data{},
		status:       StatusStarting,
		OtelProvider: nil,
		muxStatus:    sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(svc)
	}

	svc.Ctx, svc.cancel = context.WithCancel(svc.Ctx)

	// log initialization
	logger, logWriter := log.NewWithContext(map[string]string{"service": svc.Meta.Name})
	svc.Ctx = logger.WithContext(svc.Ctx)
	svc.logWriter = logWriter

	// observability provider controlled by envs obs.*
	o, err := obs.New(svc.Ctx, *svc.Meta)
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
	s.muxStatus.Lock()
	defer s.muxStatus.Unlock()
	s.status = StatusStarted
	s.startTime = time.Now()
}

func (s *Base) Status() (string, time.Time) {
	s.muxStatus.RLock()
	defer s.muxStatus.RUnlock()
	return s.status.String(), s.startTime
}

func (s *Base) Stopped() {
	s.muxStatus.Lock()
	defer s.muxStatus.Unlock()
	s.status = StatusStopped
	s.startTime = time.Time{}
}

func (s *Base) Name() string {
	return s.Meta.Name
}

func (s *Base) Metadata() map[string]string {
	m := map[string]string{
		"Name": s.Meta.Name,
		"Addr": s.Addr,
		"Kind": s.Meta.Kind,
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

func (s StatusType) String() string {
	return statusString[s]
}

// Setters
func WithMetaInfo(meta *meta.Data) Option {
	return func(s *Base) {
		s.Meta = meta
	}
}

func WithName(name string) Option {
	return func(s *Base) {
		s.Meta.Name = name
	}
}

func WithKind(kind string) Option {
	return func(s *Base) {
		s.Meta.Kind = kind
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
