# Hayaku

**Hayaku** is a Go rate limiting library with a built-in async telemetry pipeline. It answers two questions for any backend service:
1. **Is this request rate limited?**
2. **What are the metrics — region, time, failure reason — for analysis?**

---

## Features

- **Two rate limiting strategies** — Token Bucket (in-memory, per-user) and Sliding Window (Redis-backed, distributed)
- **Async telemetry pipeline** — every request fires a non-blocking event; a background goroutine writes metrics without adding latency to the request path
- **Pluggable emitters** — default in-memory store for `GetMetrics()`; optional OpenTelemetry integration for Grafana/Datadog dashboards (coming)
- **Pluggable `RateLimiter` interface** — swap strategies without touching application code
- **Automatic bucket cleanup** — Manager sweeper reclaims goroutines and memory for inactive users

---

## Architecture

```
Client Request
     │
     ▼
InstrumentedLimiter.Allow(userID, ip, region)
     │
     ├── inner.Allow(userID)           ← actual rate limit check
     │        │
     │   ┌────┴────────────────────────────────┐
     │   │  Strategy A: Token Bucket           │
     │   │  In-memory, per-user, no deps       │
     │   ├─────────────────────────────────────┤
     │   │  Strategy B: Sliding Window Redis   │
     │   │  Distributed, Lua atomic script     │
     │   └─────────────────────────────────────┘
     │
     ├── fire RequestEvent to buffered channel (non-blocking, drop if full)
     │
     └── return bool immediately

Background drain() goroutine:
     reads channel → Emitter.Emit(event)
          │
     ┌────┴──────────────────────────────┐
     │  InMemoryEmitter (default)        │  → GetMetrics() Snapshot
     ├───────────────────────────────────┤
     │  OTelEmitter (optional import)    │  → Grafana / Datadog / Prometheus
     └───────────────────────────────────┘
```

---

## Project Structure

```
hayaku/
├── cmd/hayaku/
│   └── main.go
├── ratelimiter/
│   ├── limiter.go               # RateLimiter interface
│   ├── token_bucket.go          # Channel-based token bucket
│   ├── manager.go               # Per-user bucket manager + sweeper
│   └── sliding_window_redis.go  # Redis ZSET + atomic Lua script
├── metrics/
│   ├── event.go                 # RequestEvent, DenyReasonEnum
│   ├── collector.go             # MetricStore, Snapshot
│   └── instrumented.go         # InstrumentedLimiter — wraps any RateLimiter
├── internal/
│   ├── api/
│   │   └── handler.go           # HTTP handler example — rate limit → 429/202
│   └── core/
│       ├── job.go               # Job interface
│       └── queue.go             # Queue interface
└── go.mod
```

---

## Usage

### Basic — no metrics

```go
import "github.com/FilledEther20/Hayaku/ratelimiter"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
limiter := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)

allowed := limiter.Allow("user_123")
```

### With metrics

```go
import (
    "github.com/FilledEther20/Hayaku/ratelimiter"
    "github.com/FilledEther20/Hayaku/metrics"
)

base    := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)
limiter := metrics.NewInstrumentedLimiter(base, 1024) // bufferSize=1024
defer limiter.Stop()

allowed := limiter.Allow("user_123", "203.0.113.1", "IN")

snap := limiter.GetMetrics()
fmt.Println(snap.Total, snap.Allowed, snap.Rejected)
fmt.Println(snap.ByRegion)   // map[IN:9]
fmt.Println(snap.ByHour)     // map[14:9]  ← UTC hour
fmt.Println(snap.DenialReasons)
```

### With OpenTelemetry (coming)

```go
// optional import — users who don't want OTel never compile it
import hkotel "github.com/FilledEther20/Hayaku/metrics/otel"

limiter := metrics.NewInstrumentedLimiter(base, 1024,
    metrics.WithEmitter(hkotel.New(otel.GetMeterProvider())),
)
// metrics flow to Grafana / Datadog / Prometheus automatically
```

---

## Rate Limiting Strategies

### Token Bucket (in-memory)

```go
bucket := ratelimiter.NewTokenBucket(capacity, rate) // capacity=burst, rate=tokens/sec
bucket.Start(ctx)        // starts background refill goroutine

bucket.Wait(ctx)         // blocks until token available
```

The `Manager` gives each user their own bucket, created lazily on first request:

```go
manager.Allow("user_123")            // creates bucket on first call
manager.StartSweeper(ctx, 1*time.Hour) // cancels and removes inactive buckets
```

### Sliding Window (Redis)

```go
limiter := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)
limiter.Allow("user_123")
```

Atomic Lua script per request:
1. `ZREMRANGEBYSCORE` — remove entries outside the window
2. `ZCARD` — count remaining
3. `ZADD` — add if under limit
4. `EXPIRE` — set TTL to bound memory

Redis key: `ratelimit:<userID>`

---

## Telemetry

Every call to `Allow()` records:

| Field | Description |
|-------|-------------|
| `Timestamp` | UTC time of request |
| `UserID` | from `X-User-ID` header or caller |
| `IP` | client IP |
| `Region` | e.g. `CF-IPCountry` header value |
| `Allowed` | bool |
| `Reason` | `None` / `RateLimited` / `BackendDown` / `NetworkError` |
| `Latency` | time spent in `Allow()` |

`GetMetrics()` returns a deep-copy snapshot — safe to read while the limiter continues writing.

---

## HTTP API

```
POST /jobs/submit
Header: X-User-ID: <id>

202 Accepted          — allowed
429 Too Many Requests — rate limited
503 Service Unavailable — queue full
```

---

## Getting Started

```bash
git clone https://github.com/FilledEther20/Hayaku.git
cd Hayaku
go mod download
redis-server &
go run ./cmd/hayaku
```

**Prerequisites**: Go 1.23+, Redis

---

## Roadmap

- [ ] OpenTelemetry emitter (`metrics/otel`)
- [ ] Functional options on `NewInstrumentedLimiter`
- [ ] Benchmark suite — p99 latency under concurrent load
- [ ] `NOSCRIPT` retry on Redis restart
- [ ] HTTP server wired in `main.go`

---

## Dependencies

| Module | Purpose |
|--------|---------|
| `github.com/redis/go-redis/v9` | Sliding window rate limit + optional Redis metrics |
| `github.com/google/uuid` | Unique member IDs in sliding window ZSET |

---

## License

MIT © 2025 Chaitanya Gairola
