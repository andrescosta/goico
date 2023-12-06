package obs

import (
	"github.com/andrescosta/goico/pkg/env"
	//"github.com/andrescosta/goico/pkg/service"
)

const (
	ProviderConfig  = ""
	ExportersConfig = ""
	ProviderOtel    = "otel"
	ExpConsole      = "console"
	ExpHTTP         = "http"
)

type ExporterService struct {
	//svc *service.HttpService
}

func MyInit() {
	if env.GetAsString("provider", ProviderOtel) == ProviderOtel {

	}
	//exporters := env.GetCommaArray("exporters", "")

}

func NewMetricsService(name, resource string) (*ExporterService, error) {
	/*svc, err := service.NewHttpService(context.Background(), name, resource,
		func(ctx context.Context) (http.Handler, error) {
			exporter, err := prometheus.NewExporter(prometheus.Options{})
			if err != nil {
				return nil, err
			}
			return exporter, nil
		})
	if err != nil {
		return nil, err
	}
	return &MetricsService{svc: svc}, nil
	*/
	return nil, nil
}

func (s ExporterService) Serve() error {
	/*if err := s.svc.Serve(); err != nil {
		return err
	}*/
	return nil
}

func TryEnableMetrics(config string) (bool, error) {
	/*if env.GetAsBool(config+".enabled", false) {
		svc, err := NewMetricsService(config, "metrics")
		if err != nil {
			return false, err
		}
		go func() {
			err := svc.Serve()
			if err != nil {
				log.Fatal(err)
			}
		}()
		return true, nil
	}
	return false, nil
	*/
	return true, nil
}
