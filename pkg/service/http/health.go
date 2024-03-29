package http

import (
	"context"
	"net/http"
	"net/url"

	"github.com/andrescosta/goico/pkg/service"
)

type HealthCheckClient struct {
	ServerAddr string
	Builder    service.HTTPClientBuilder
}

func (c *HealthCheckClient) Close() error {
	return nil
}

func (c *HealthCheckClient) CheckOk(_ context.Context) error {
	return checkServiceHealth(c.Builder, c.ServerAddr)
}

func checkServiceHealth(s service.HTTPClientBuilder, addr string) error {
	url, err := url.Parse("http://" + addr + "/health")
	if err != nil {
		return err
	}
	cli, err := s.NewHTTPClient(addr)
	if err != nil {
		return err
	}
	r := http.Request{URL: url}
	res, err := cli.Do(&r)
	if err != nil {
		return err
	}
	if err := res.Body.Close(); err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return service.ErrNotHealthy{Addr: addr}
	}
	return nil
}
