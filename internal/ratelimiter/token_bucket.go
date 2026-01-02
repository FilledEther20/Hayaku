package ratelimiter

import (
	"context"
	"time"
)

type tokens chan struct{}

type TokenBucket struct {
	size          int64
	tokensPresent tokens
	ticker        *time.Ticker
}

func NewTokenBucket(capacity int64, rate int64) *TokenBucket {
	tokens := make(chan struct{}, capacity)

	for i := int64(0); i < capacity; i++ {
		tokens <- struct{}{}
	}

	interval := time.Second / time.Duration(rate)

	return &TokenBucket{
		size:          capacity,
		tokensPresent: tokens,
		ticker:        time.NewTicker(interval),
	}
}

// To spin up a new process to fill the bucket
func (tb *TokenBucket) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-tb.ticker.C:
				select {
				case tb.tokensPresent <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				tb.ticker.Stop()
				return
			}
		}
	}()
}

// To make the process wait until a token is there
func (tb *TokenBucket) Wait(ctx context.Context) error {
	select {
	case <-tb.tokensPresent:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
