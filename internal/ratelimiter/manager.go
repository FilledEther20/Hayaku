package ratelimiter

import (
	"context"
	"sync"
	"time"
)

type UserBucket struct {
	bucket   *TokenBucket
	lastSeen time.Time
	cancel   context.CancelFunc
}

type Manager struct {
	mu      sync.RWMutex
	buckets map[string]*UserBucket
	rate    int64
	cap     int64
}

func (m *Manager) Allow(userID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	ub, exists := m.buckets[userID]
	if !exists {
		ctx, cancel := context.WithCancel(context.Background())

		bucket := NewTokenBucket(m.cap, m.rate)
		bucket.Start(ctx)

		ub = &UserBucket{
			bucket: bucket,
			cancel: cancel,
		}
		m.buckets[userID] = ub
	}

	ub.lastSeen = time.Now()

	select {
	case <-ub.bucket.tokensPresent:
		return true
	default:
		return false
	}
}

func (m *Manager) sweep(ttl time.Duration) {
	m.mu.Lock()

	defer m.mu.Unlock()

	now := time.Now()
	for userID, ub := range m.buckets {
		if now.Sub(ub.lastSeen) > ttl {
			ub.cancel()
			delete(m.buckets, userID)
		}
	}
}

func (m *Manager) StartSweeper(ctx context.Context, ttl time.Duration) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.sweep(ttl)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
