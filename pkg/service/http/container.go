package http

import (
	"errors"
	"fmt"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service"
)

type Container struct {
	service.HttpConn
	Svc  *Service
	Name string
}

func (c *Container) HealthCheckClient() (service.HealthChecker, error) {
	if c.Svc == nil {
		return nil, errors.New("service not set")
	}
	return c.Svc.HelthCheckClient(c.ClientBuilder), nil
}

func (c *Container) Addr() string {
	addrEnv := c.Name + ".addr"
	return env.String(addrEnv, "")
}

func (c *Container) AddrOrPanic() string {
	a := c.Addr()
	if a == "" {
		panic(fmt.Sprintf(".addr not configured for %s", c.Name))
	}
	return a
}
