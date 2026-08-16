// Package ratelimiter provides in-memory and Redis-backed rate limiting strategies.
package ratelimiter

// RateLimiter is the single interface shared by all strategies.
type RateLimiter interface {
	Allow(userID string) bool
}
