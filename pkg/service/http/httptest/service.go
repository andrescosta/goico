package httptest

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service"
	httpsvc "github.com/andrescosta/goico/pkg/service/http"
	"github.com/gorilla/mux"
)

type HandlerFn func(rw http.ResponseWriter, r *http.Request)

type PathHandler struct {
	Scheme  string
	Path    string
	Handler HandlerFn
}
type Service struct {
	URL       string
	Client    *http.Client
	Servedone <-chan error
	Cancel    context.CancelFunc
}

func NewService(ctx context.Context, handlers []PathHandler, hfn httpsvc.HealthCheckFn, stackLevel httpsvc.StackLevel) (*Service, error) {
	localhost := "127.0.0.1:0"
	svc, err := httpsvc.New(
		httpsvc.WithContext(ctx),
		httpsvc.WithAddr(localhost),
		httpsvc.WithName("listener-test"),
		httpsvc.WithStackLevelOnError[*httpsvc.ServiceOptions](stackLevel),
		httpsvc.WithHealthCheck[*httpsvc.ServiceOptions](hfn),
		httpsvc.WithInitRoutesFn[*httpsvc.ServiceOptions](func(ctx context.Context, r *mux.Router) error {
			for _, h := range handlers {
				r.HandleFunc(h.Path, h.Handler).Schemes(h.Scheme)
			}
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	addr, servedone, err := start(localhost, svc)
	if err != nil {
		return nil, err
	}
	return &Service{
		URL:       "http://" + addr,
		Client:    &http.Client{Transport: &http.Transport{}},
		Servedone: servedone,
	}, nil
}

func SetHTTPServerTimeouts(t time.Duration) {
	timeout := t.String()
	env.SetargsV("http.timeout.write", timeout)
	env.SetargsV("http.timeout.read", timeout)
	env.SetargsV("http.timeout.idle", timeout)
	env.SetargsV("http.timeout.handler", timeout)
}

func NewSidecar(ctx context.Context, hfn httpsvc.HealthCheckFn) (*Service, error) {
	localhost := "127.0.0.1:0"
	service, err := service.New(
		service.WithName("sidecar-test"),
		service.WithContext(ctx),
		service.WithKind("headless"),
		service.WithAddr(localhost),
	)
	if err != nil {
		return nil, err
	}
	svc, err := httpsvc.NewSidecar(
		httpsvc.WithHealthCheck[*httpsvc.SidecarOptions](hfn),
		httpsvc.WithPrimaryService(service),
	)
	if err != nil {
		return nil, err
	}
	addr, servedone, err := start(localhost, svc)
	if err != nil {
		return nil, err
	}
	return &Service{
		URL:       "http://" + addr,
		Client:    &http.Client{Transport: &http.Transport{}},
		Servedone: servedone,
	}, nil
}

func start(addr string, svc *httpsvc.Service) (string, chan error, error) {
	servedone := make(chan error, 1)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	go func() {
		servedone <- svc.DoServe(listener)
		close(servedone)
	}()
	return listener.Addr().String(), servedone, nil
}

func (s *Service) Get(url string) (*http.Response, error) {
	return s.Verb(url, http.MethodGet, nil)
}

func (s *Service) Post(url string) (*http.Response, error) {
	return s.Verb(url, http.MethodPost, nil)
}

func (s *Service) Verb(url string, verb string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), verb, url, body)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}
