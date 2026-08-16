// Package metrics wraps any RateLimiter and records per-request telemetry.
package metrics

import (
	"time"

	"github.com/FilledEther20/Hayaku/ratelimiter"
)

type InstrumentedLimiter struct {
	inner  ratelimiter.RateLimiter
	events chan RequestEvent
	store  *MetricStore
	done   chan struct{}
}

// NewInstrumentedLimiter wraps inner and records telemetry; bufferSize controls event queue depth before drops.
func NewInstrumentedLimiter(inner ratelimiter.RateLimiter, bufferSize int) *InstrumentedLimiter {
	il := &InstrumentedLimiter{
		inner:  inner,
		events: make(chan RequestEvent, bufferSize),
		store:  NewMetricStore(),
		done:   make(chan struct{}),
	}
	go il.drain()
	return il
}

func (il *InstrumentedLimiter) Allow(userID, ip, region string) bool {
	start := time.Now()
	allowed := il.inner.Allow(userID)

	reason := None
	if !allowed {
		reason = RateLimited
	}

	event := RequestEvent{
		Timestamp: start,
		UserID:    userID,
		IP:        ip,
		Region:    region,
		Allowed:   allowed,
		Reason:    reason,
		Latency:   time.Since(start),
	}

	// drop the event if the buffer is full — never block the request path
	select {
	case il.events <- event:
	default:
	}

	return allowed
}

func (il *InstrumentedLimiter) GetMetrics() Snapshot {
	return il.store.Snapshot()
}

// Stop drains remaining events and shuts down the background goroutine.
func (il *InstrumentedLimiter) Stop() {
	close(il.events)
	<-il.done
}

func (il *InstrumentedLimiter) drain() {
	defer close(il.done)
	for e := range il.events {
		il.store.Record(e)
	}
}
