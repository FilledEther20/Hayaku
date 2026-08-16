// Package ratelimiter provides in-memory and Redis-backed rate limiting strategies.
package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SlidingWindowRedis struct {
	client     *redis.Client
	window     time.Duration
	size       int
	scriptHash string
}

const script = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, '-inf' , now - window)

local current_count = redis.call('ZCARD', key)
if current_count < limit then
    -- 3. Add current request
    redis.call('ZADD', key, now, member)
    -- 4. Set expiry to save memory
    redis.call('EXPIRE', key, math.ceil(window / 1000) + 1)
    return 1
else
    return 0
end
`

func NewSlidingWindowRedis(rdb *redis.Client, window time.Duration, limit int) *SlidingWindowRedis {
	ctx := context.Background()
	hash, err := rdb.ScriptLoad(ctx, script).Result()

	if err != nil {
		fmt.Printf("Warning: Failed to pre-load Lua Script: %v\n", err)
	}

	return &SlidingWindowRedis{
		client:     rdb,
		window:     window,
		size:       limit,
		scriptHash: hash,
	}
}

func (s *SlidingWindowRedis) Allow(userID string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:%s", userID)
	now := time.Now().UnixMilli()
	windowMs := s.window.Milliseconds()

	member := fmt.Sprintf("%d:%s", now, uuid.New().String())

	var result interface{}
	var err error
	if s.scriptHash != "" {
		result, err = s.client.EvalSha(ctx, s.scriptHash, []string{key}, now, windowMs, s.size, member).Result()
	} else {
		result, err = s.client.Eval(ctx, script, []string{key}, now, windowMs, s.size, member).Result()
	}
	if err != nil {
		fmt.Printf("Redis Error during Rate Limit check: %v\n", err)
		return false
	}
	return result.(int64) == 1
}
