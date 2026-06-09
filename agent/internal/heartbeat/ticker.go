package heartbeat

import "time"

type ticker interface {
	Stop()
}

func newTicker(interval time.Duration) *time.Ticker {
	return time.NewTicker(interval)
}
