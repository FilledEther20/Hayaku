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

	// bucket := ratelimiter.NewTokenBucket(5, 2)

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	// bucket.Start(ctx)

	for i := 0; i < 7; i++ {
		if limiter.Allow("user_1") {
			fmt.Printf("Request %d: Allowed\n", i+1)
		} else {
			fmt.Printf("Request %d: Denied\n", i+1)
		}
	}

	// for i := 0; i < 10; i++ {
	// 	startAt := time.Now()
	// 	time.Sleep(1000 * time.Millisecond)
	// 	bucket.Wait(context.Background())
	// 	processedAt := time.Now()
	// 	fmt.Printf("Processing request[%v] at [%v] processed at [%v]\n", i+1, startAt, processedAt)
	// }
}
