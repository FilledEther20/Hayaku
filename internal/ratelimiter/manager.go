package ratelimiter

import (
	"context"
	"sync"
)

type Manager struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket
	rate    int64
	cap     int64
}

func (m *Manager) Allow(userID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	bucket, exists := m.buckets[userID]

	if !exists {
		bucket = NewTokenBucket(m.cap, m.rate)
		bucket.Start(context.Background())
		m.buckets[userID] = bucket
	}

	select {
	case <-bucket.tokensPresent:
		return true
	default:
		return false
	}
}
