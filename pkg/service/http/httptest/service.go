package httptest

import (
	"context"
	"net"
	"net/http"
	"time"

	httpsvc "github.com/andrescosta/goico/pkg/service/http"
	"github.com/gorilla/mux"
)

type PathHandler struct {
	Scheme  string
	Path    string
	Handler func(http.ResponseWriter, *http.Request)
}

type Service struct {
	URL       string
	Client    *http.Client
	Servedone <-chan error
}

func NewService(ctx context.Context, handlers []PathHandler, hfn httpsvc.HealthChkFn) (*Service, error) {
	localhost := "127.0.0.1:0"
	ch := make(chan string)
	svc, err := httpsvc.New(
		httpsvc.WithContext(ctx),
		httpsvc.WithAddr(&localhost),
		httpsvc.WithName("listener-test"),
		httpsvc.WithHealthCheck[*httpsvc.RouterOptions](hfn),
		httpsvc.WithDoListener[*httpsvc.RouterOptions](func(addr string) (net.Listener, error) {
			l, err := net.Listen("tcp", addr)
			a := l.Addr().String()
			if err != nil {
				if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
					return nil, err
				}
			}
			ch <- a
			return l, err

		}),
		httpsvc.WithInitRoutesFn(func(ctx context.Context, r *mux.Router) error {
			for _, h := range handlers {
				r.HandleFunc(h.Path, h.Handler).Schemes(h.Scheme)
			}
			return nil
		}),
		httpsvc.WithHttpServerBuilder[*httpsvc.RouterOptions](GetHttpServerBuilder()),
	)
	if err != nil {
		return nil, err
	}
	servedone := make(chan error, 1)
	go func() {
		servedone <- svc.Serve()
		close(servedone)
	}()
	addr := <-ch
	return &Service{
		URL:       "http://" + addr,
		Client:    &http.Client{Transport: &http.Transport{}},
		Servedone: servedone,
	}, nil
}

func GetHttpServerBuilder() httpsvc.HttpServerBuilderFn {
	return func(r http.Handler) *http.Server {
		return &http.Server{
			WriteTimeout: time.Second * 1,
			ReadTimeout:  time.Second * 1,
			IdleTimeout:  time.Second * 1,
			Handler:      http.TimeoutHandler(r, time.Second, ""),
		}
	}
}
