package service

import "context"

type HealthChecker interface {
	CheckOk(ctx context.Context) error
	Close() error
}

type HealthStatus struct {
	Status        string `json:"status"`
	ServiceStatus string `json:"service_status"`
	StartedAt     string `json:"started_at"`
	Error         error
	Details       map[string]string `json:"details"`
}
