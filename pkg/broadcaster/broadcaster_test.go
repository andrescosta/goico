package broadcaster_test

import (
	"context"
	"testing"

	. "github.com/andrescosta/goico/pkg/broadcaster"
)

type data struct {
	id   int
	name string
}

// Stop
// Subscribe
// Unsubscribe
// Write

func TestBroadcaster(t *testing.T) {
	var b *Broadcaster[data] = Start[data](context.Background())
	print(b)
}
