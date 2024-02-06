package service

import "context"

type HealthChecker interface {
	CheckOk(ctx context.Context) error
	Close() error
}
