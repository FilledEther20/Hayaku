package ratelimiter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FilledEther20/Hayaku/internal/ratelimiter"
	"github.com/redis/go-redis/v9"
)

func TestSlidingWindowRedis(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	// unique key so leftover state from a previous run doesn't interfere
	userID := fmt.Sprintf("test-user-%d", time.Now().UnixNano())
	t.Cleanup(func() { rdb.Del(context.Background(), "ratelimit:"+userID) })

	const limit = 5
	rl := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, limit)

	for i := 1; i <= limit; i++ {
		if !rl.Allow(userID) {
			t.Fatalf("request %d should be allowed", i)
		}
	}

	if rl.Allow(userID) {
		t.Fatal("request over limit should be denied")
	}
}
