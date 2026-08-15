package main

import (
	"fmt"
	"time"

	"github.com/FilledEther20/Hayaku/internal/metrics"
	"github.com/FilledEther20/Hayaku/internal/ratelimiter"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	base := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)
	limiter := metrics.NewInstrumentedLimiter(base, 1024)
	defer limiter.Stop()

	for i := 0; i < 9; i++ {
		allowed := limiter.Allow("user_1", "ip_1", "Reg_1")
		fmt.Printf("Is request allowed %t\n", allowed)
	}

	snap := limiter.GetMetrics()
	fmt.Printf("\n--- Metrics ---\n")
	fmt.Printf("Total:    %d\n", snap.Total)
	fmt.Printf("Allowed:  %d\n", snap.Allowed)
	fmt.Printf("Rejected: %d\n", snap.Rejected)
	fmt.Printf("By region: %v\n", snap.ByRegion)
	fmt.Printf("By hour:   %v\n", snap.ByHour)
	fmt.Printf("Reasons:   %v\n", snap.DenialReasons)
}
