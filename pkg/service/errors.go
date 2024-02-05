package service

import (
	"errors"
	"fmt"
)

var (
	ErrOtelStack  = errors.New("obs.New: error initializing otel stack")
	ErrEnvLoading = errors.New("env.Populate: error initializing otel stack")
)

type ErrNotHealthy struct {
	Addr string
}

func (e ErrNotHealthy) Error() string {
	return fmt.Sprintf("service at %s not healthy", e.Addr)
}
