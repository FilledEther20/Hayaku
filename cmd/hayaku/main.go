package main

import (
	"context"
	"fmt"
	"time"
	"github.com/FilledEther20/Hayaku/internal/ratelimiter"
)

func main() {
	bucket := ratelimiter.NewTokenBucket(5, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bucket.Start(ctx)

	for i := 0; i < 10; i++ {
		startAt := time.Now()
		time.Sleep(1000 * time.Millisecond)
		bucket.Wait(context.Background())
		processedAt := time.Now()
		fmt.Printf("Processing request[%v] at [%v] processed at [%v]\n", i+1, startAt, processedAt)
	}
}