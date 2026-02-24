package core

type RateLimiter interface {
	Allow(userID string) bool
}