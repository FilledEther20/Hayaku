# Hayaku — Design Document

*Last updated: August 2026*

---

## Why I built this

I kept running into the same gap: rate limiting libraries tell you whether a request was blocked, but not much else. If you're seeing a spike in 429s at 2am you want to know — which users, which region, what time window. Getting that out of most libraries means bolting on your own logging, which usually ends up blocking the request path or getting skipped entirely.

Hayaku is my attempt to make observability a first-class part of rate limiting rather than an afterthought.

---

## The two strategies

### Token Bucket (in-memory)

I modelled the bucket as a buffered channel of empty structs. Each slot in the channel is a token. This means:

- Consuming a token is just a non-blocking channel receive — no mutex needed on the hot path.
- The channel's own capacity enforces the burst limit.
- A background goroutine feeds tokens in on a ticker, one per interval.

The `Manager` wraps this to give each user their own bucket, created lazily on first request. A sweeper goroutine walks the map periodically and cancels the context of any bucket that hasn't seen traffic within the TTL, which stops the refill goroutine and frees memory.

This strategy is the right call when you don't need cross-instance coordination — single server, low complexity, no Redis dependency.

### Sliding Window (Redis)

For distributed deployments where multiple instances share state, the token bucket breaks down — each instance has its own bucket so the limits aren't global. The sliding window strategy fixes this by keeping state in Redis.

The implementation uses a sorted set (ZSET) per user, where the score is the request timestamp in milliseconds. On every request:

1. Remove entries outside the window (`ZREMRANGEBYSCORE`)
2. Count what's left (`ZCARD`)
3. Add the new entry if under the limit (`ZADD`)
4. Reset the TTL (`EXPIRE`)

All four steps run inside a single Lua script, which Redis executes atomically. Without this, two concurrent requests on different machines could both read the count, both see it as under limit, and both get allowed — a classic TOCTOU race. The Lua script eliminates that.

The script is pre-loaded on startup with `SCRIPT LOAD` and called via `EVALSHA` on every request, which is cheaper than sending the full script each time. If the load fails (Redis not up yet, for instance), it falls back to inline `EVAL`.

Member IDs in the ZSET are `timestamp:uuid` — the UUID part handles the case where two requests arrive at the exact same millisecond and would otherwise collide on score + member uniqueness.

---

## The telemetry pipeline

The core idea: `Allow()` should return in microseconds regardless of what happens to the metric. The solution is to decouple recording from the request path entirely.

`InstrumentedLimiter` wraps any `RateLimiter` and on each `Allow()` call:

1. Calls the inner limiter — this is the only thing that can block
2. Builds a `RequestEvent` (timestamp, userID, IP, region, allowed, reason, latency)
3. Does a non-blocking send onto a buffered channel — if the channel is full, the event is dropped silently

A single background goroutine reads from that channel and writes into a `MetricStore` protected by a `sync.RWMutex`. Because only one goroutine ever writes, there's no write contention — the mutex is only there for `Snapshot()` reads.

`Snapshot()` returns a deep copy of the current state, so callers can read it safely while the limiter keeps running.

The buffer size is caller-controlled. A bigger buffer means fewer drops under burst traffic but more memory. A reasonable default for most services is 1024.

---

## What I chose not to do (yet)

**Persistent metrics** — the in-memory store resets on restart. That's intentional for now. The OTel emitter (on the roadmap) is the right place for durability.

**Distributed token bucket** — I could implement this with Redis + atomic decrement, but it has subtler failure modes than the sliding window and doesn't add enough over what the sliding window already provides.

**HTTP middleware** — I deliberately kept Hayaku framework-agnostic. Wrapping it in a `net/http` middleware or a Gin/Echo plugin is a one-liner and doesn't belong in the core library.

---

## Roadmap

- OpenTelemetry emitter (`metrics/otel`) — plug into any existing observability stack
- Benchmark suite — p99 latency numbers under concurrent load, with and without Redis
- `NOSCRIPT` retry — if Redis flushes scripts after a restart, fall back to `EVAL` automatically
- More concurrency tests on the `Manager` sweeper path
