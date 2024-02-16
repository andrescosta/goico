package syncutil

import "time"

type Ticker interface {
	Chan() <-chan time.Time
	Stop()
	Tick()
}

type TimeTicker struct {
	Ticker *time.Ticker
}

func (t *TimeTicker) Chan() <-chan time.Time {
	return t.Ticker.C
}

func (t *TimeTicker) Stop() {
	t.Ticker.Stop()
}

func (t *TimeTicker) Tick() {
}

type ChannelTicker struct {
	C chan time.Time
}

func (t *ChannelTicker) Chan() <-chan time.Time {
	return t.C
}

func (t *ChannelTicker) Stop() {
	close(t.C)
}

func (t *ChannelTicker) Tick() {
	t.C <- time.Now()
}
