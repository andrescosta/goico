package httptest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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

func SetArgs(name string, value string) {
	os.Args = append(os.Args, fmt.Sprintf("--env:%s=%s", name, value))
}

func NewService(ctx context.Context, handlers []PathHandler, hfn httpsvc.HealthCheckFn, stackLevel httpsvc.StackLevel) (*Service, error) {
	localhost := "127.0.0.1:0"
	svc, err := httpsvc.New(
		httpsvc.WithContext(ctx),
		httpsvc.WithAddr(&localhost),
		httpsvc.WithName("listener-test"),
		httpsvc.WithStackLevelOnError[*httpsvc.ServiceOptions](stackLevel),
		httpsvc.WithHealthCheck[*httpsvc.ServiceOptions](hfn),
		httpsvc.WithInitRoutesFn(func(ctx context.Context, r *mux.Router) error {
			for _, h := range handlers {
				r.HandleFunc(h.Path, h.Handler).Schemes(h.Scheme)
			}
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	addr, servedone := start(svc)
	return &Service{
		URL:       "http://" + addr,
		Client:    &http.Client{Transport: &http.Transport{}},
		Servedone: servedone,
	}, nil
}

func SetHTTPServerTimeouts(t time.Duration) {
	timeout := t.String()
	SetArgs("http.timeout.write", timeout)
	SetArgs("http.timeout.read", timeout)
	SetArgs("http.timeout.idle", timeout)
	SetArgs("http.timeout.handler", timeout)
}

func MetadataOn() {
	SetArgs("metadata.enabled", "true")
}

func MetadataOff() {
	SetArgs("metadata.enabled", "false")
}

func NewSidecar(ctx context.Context, hfn httpsvc.HealthCheckFn) (*Service, error) {
	localhost := "127.0.0.1:0"
	service, err := service.New(
		service.WithName("sidecar-test"),
		service.WithContext(ctx),
		service.WithKind("headless"),
		service.WithAddr(&localhost),
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
	addr, servedone := start(svc)
	return &Service{
		URL:       "http://" + addr,
		Client:    &http.Client{Transport: &http.Transport{}},
		Servedone: servedone,
	}, nil
}

func start(svc *httpsvc.Service) (string, chan error) {
	servedone := make(chan error, 1)
	addrdone := make(chan string)
	go func() {
		a := <-svc.AddressReady
		addrdone <- a
	}()
	go func() {
		servedone <- svc.Serve()
		close(servedone)
	}()
	return <-addrdone, servedone
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
