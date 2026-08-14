package main

import (
	"fmt"
	"time"

	"github.com/FilledEther20/Hayaku/internal/ratelimiter"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	limiter := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)

	for i := 0; i < 9; i++ {
		allowed := limiter.Allow("user_1")
		fmt.Printf("Is request allowed %t\n", allowed)
	}
}
