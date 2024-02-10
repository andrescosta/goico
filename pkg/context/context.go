package context

import (
	"context"
	"os/signal"
	"syscall"
)

func ForEndSignals() (context.Context, context.CancelFunc) {
	// OS signal handling
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

func ForEndSignalsWithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// OS signal handling
	return signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
}
