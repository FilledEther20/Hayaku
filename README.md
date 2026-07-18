# Hayaku

**Hayaku** is a Go backend infrastructure library that combines a **distributed rate limiter** with an **async job queue and worker pool**. It provides the core building blocks for protecting APIs from abuse and processing tasks asynchronously at scale.

---

## Features

- **Two rate limiting strategies** — Token Bucket (in-memory, per-user) and Sliding Window (Redis-backed, distributed)
- **Pluggable interfaces** — `RateLimiter` and `Queue` are defined as interfaces; swap implementations without touching business logic
- **Concurrent worker pool** — Processes queued jobs across configurable goroutine workers
- **Automatic cleanup** — Manager sweeper reclaims memory for inactive user buckets
- **Atomic Redis operations** — Sliding window uses a pre-loaded Lua script for race-free counting

---

## Architecture

```
Client Request
     │
     ▼
┌─────────────────────┐
│   HTTP Handler      │  ← Extracts X-User-ID header
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐          ┌──────────────────────────┐
│   Rate Limiter      │          │  Strategy A: Token Bucket │
│   (core.RateLimiter)│ ──────►  │  In-memory, per-user     │
└────────┬────────────┘          ├──────────────────────────┤
         │                       │  Strategy B: Sliding      │
         │ allowed               │  Window (Redis ZSET)      │
         ▼                       └──────────────────────────┘
┌─────────────────────┐
│   Job Queue         │  ← core.Queue interface
│   (core.Queue)      │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│   Worker Pool       │  ← Concurrent job execution
└─────────────────────┘
```

---

## Project Structure

```
hayaku/
├── cmd/hayaku/
│   └── main.go                    # Entry point / test harness
├── internal/
│   ├── api/
│   │   └── handler.go             # HTTP handler (HandleSubmitJob)
│   ├── core/
│   │   ├── job.go                 # Job interface
│   │   ├── limiter.go             # RateLimiter interface
│   │   └── queue.go               # Queue interface
│   ├── ratelimiter/
│   │   ├── token_bucket.go        # Token Bucket algorithm
│   │   ├── manager.go             # Per-user bucket manager with sweeper
│   │   └── sliding_window_redis.go# Redis-backed sliding window
│   └── worker/
│       ├── job.go                 # Worker-local Job interface
│       └── pool.go                # Goroutine worker pool
└── go.mod
```

---

## Rate Limiting Strategies

### Token Bucket (in-memory)

Each user gets a dedicated token bucket created on first request. Tokens are refilled at a fixed rate and capped at the configured capacity.

```go
// capacity = max burst, rate = tokens refilled per second
bucket := ratelimiter.NewTokenBucket(capacity int64, rate int64)

// Start the background refill goroutine
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
bucket.Start(ctx)

// Non-blocking check (used by Allow)
// <-bucket.tokensPresent

// Blocking wait — caller suspends until a token is available
err := bucket.Wait(ctx)
```

The `Manager` wraps `TokenBucket` to provide per-user isolation, lazy initialization, and automatic cleanup:

```go
manager := &ratelimiter.Manager{ /* rate, cap */ }
manager.StartSweeper(ctx, 1*time.Hour) // clean up inactive users

allowed := manager.Allow("user_123") // true / false
```

**Sweeper**: runs every 5 minutes and cancels + removes buckets for users not seen within the TTL.

### Sliding Window (Redis)

Tracks requests inside a rolling time window using a Redis sorted set. Works correctly across multiple application instances.

```go
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
limiter := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)
// → max 5 requests per 2-second window per user

allowed := limiter.Allow("user_123")
```

**How it works (Lua script, atomic):**
1. Remove members older than `now - window` (`ZREMRANGEBYSCORE`)
2. Count remaining members (`ZCARD`)
3. If count < limit → add current request (`ZADD`) and set TTL (`EXPIRE`), return `1`
4. Otherwise return `0`

Redis key format: `ratelimit:<userID>`

---

## HTTP API

### `POST /jobs/submit`

| Component | Detail |
|-----------|--------|
| Header | `X-User-ID: <id>` (required) |
| Success | `202 Accepted` — job enqueued |
| Rate limited | `429 Too Many Requests` |
| Queue full | `503 Service Unavailable` |

```go
handler := &api.HayakuHandler{
    Limiter: limiter, // any core.RateLimiter
    Queue:   queue,   // any core.Queue
}
http.HandleFunc("/jobs/submit", handler.HandleSubmitJob)
```

---

## Core Interfaces

```go
// Any job must implement:
type Job interface {
    ID() string
    Execute(ctx context.Context) error
}

// Any rate limiter must implement:
type RateLimiter interface {
    Allow(userID string) bool
}

// Any queue must implement:
type Queue interface {
    Enqueue(ctx context.Context, j Job) error
    Dequeue(ctx context.Context) (Job, error)
}
```

---

## Worker Pool

```go
pool := worker.NewPool(maxWorkers int, queueSize int)
pool.Start()
```

Workers pull jobs off `pool.JobQueue` and call `job.Execute(ctx)` concurrently.

---

## Getting Started

### Prerequisites

- Go 1.21+
- Redis (for sliding window rate limiter)

### Install

```bash
git clone https://github.com/FilledEther20/Hayaku.git
cd Hayaku
go mod download
```

### Run

```bash
# Start Redis (if using sliding window)
redis-server

go run ./cmd/hayaku
```

---

## Dependencies

| Module | Purpose |
|--------|---------|
| `github.com/redis/go-redis/v9` | Redis client for distributed rate limiting |
| `github.com/google/uuid` | Unique member IDs in sliding window ZSET |

---

## License

MIT © 2025 Chaitanya Gairola

## What is Hayaku?
Hayaku is a Golang based project solving the age old problem of efficiency in terms of security and job processing. Modern backend systems must protect themselves from abuse while processing tasks asynchronously at scale. This project implements a rate-limiting service combined with a job queue to simulate real-world backend infrastructure.

Abstract overview of how Hayaku works:  
- Client hits the layer 
- Hayaku validates the rate limit then sends it 
- Tasks are processed concurrently by the workers

## Why this project exists

Modern backend systems must:
- Protect themselves from abuse
- Handle tasks asynchronously
- Remain reliable under load

Hayaku simulates these real-world backend concerns in a single, focused system.


It is a Ongoing Product


## High-Level Architecture
- API Layer
- Rate Limiter
- Job Queue
- Worker Pool
- Storage (In-memory / Redis)

## Planned Components
- Token Bucket Rate Limiter
- Per-user quotas
- Job submission endpoint
- Worker pool with retries
- Failure handling and backoff

## Features
### Rate Limiting
- Token Bucket based rate limiter
- Per-user request quotas
- Configurable refill rates
- Rejects requests exceeding limits with clear errors

### Job Queue
- Asynchronous job submission
- In-memory queue with pluggable storage (Redis-ready)
- FIFO processing semantics

### Worker Pool
- Configurable number of workers
- Concurrent job execution
- Graceful shutdown handling

### Reliability
- Automatic retries for failed jobs
- Exponential backoff strategy
- Dead-letter handling (planned)

### Observability
- Basic runtime metrics (jobs processed, failures, rejections)
- Structured logging for debugging

### Interfaces
- CLI interface for submitting jobs and simulating load
- (Planned) HTTP API for external integration

## Rate Limiting (Token Bucket)

Hayaku uses an in-memory **Token Bucket** rate limiter to control how frequently a user can submit requests.

### How it works
- Each user is associated with a token bucket of fixed capacity.
- Tokens represent permission to process a request.
- A background goroutine refills tokens at a fixed rate.
- If no token is available, the request blocks or fails based on context.

### Why Token Bucket
- Allows short bursts while enforcing an average rate
- Simple and efficient for in-memory rate limiting
- Commonly used in real-world backend systems

### Current Guarantees
- Thread-safe token consumption using Go channels
- Bounded capacity (no token overflow)
- Context-aware waiting and cancellation
- Graceful shutdown of refill loop

### Limitations
- In-memory only (single-node)
- Not suitable for distributed rate limiting yet
- Token state is lost on process restart

Future versions may use Redis or another shared store to support distributed rate limiting.
