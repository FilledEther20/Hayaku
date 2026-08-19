<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/Redis-v9-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis Version" />
  <img src="https://img.shields.io/badge/OpenTelemetry-Compatible-F5A800?style=for-the-badge&logo=opentelemetry&logoColor=white" alt="OpenTelemetry" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License" />
</p>

---

**Hayaku** (`早く` — *fast/early*) is a high-performance Go rate limiting library with a built-in zero-allocation asynchronous telemetry pipeline. It answers two critical questions for any backend service:

1. 🚦 **Is this request rate limited?**
2. 📊 **What are the real-time metrics (region, latency, failure reason) for telemetry and auditing?**

---

## 🌟 Key Features

* 🎯 **Multi-Tier Configurable Policies** — Define burst capacity and refill rates per user tier (`Free`, `Pro`, `Enterprise`, `RBAC`) with dynamic $\mathcal{O}(1)$ policy resolution.
* 🔄 **Dual Rate Limiting Strategies** — In-memory Token Bucket for microsecond local checks and Redis-backed Sliding Window for distributed consistency.
* ⚡ **Zero-Latency Async Telemetry** — Non-blocking buffered event loop offloads metric emission from the hot request path.
* 🔌 **Pluggable Metric Emitters** — Built-in atomic memory store with instant snapshotting and native OpenTelemetry integration for Datadog, Prometheus, and Grafana.
* 🧩 **Pluggable `RateLimiter` Interface** — Hot-swap underlying algorithms or policy managers without refactoring application business logic.
* 🧹 **Automatic Bucket Sweeper** — Background TTL worker dynamically reclaims inactive memory and handles tier migrations gracefully.

---

## 🏗️ Architecture


```

```
                         Client Request
                               │
                               ▼
          InstrumentedLimiter.Allow(userID, ip, region)
                               │
   ┌───────────────────────────┴───────────────────────────┐
   ▼                                                       ▼

```

[ inner.Allow(userID) ]                                [ Async Telemetry ]
│                                                       │
┌────┴──────────────────────────────────────────┐    Non-blocking push
│ 🔹 Strategy A: Multi-Policy Token Bucket      │    to buffered channel
│    (In-Memory, O(1) dynamic tier resolution)  │            │
├───────────────────────────────────────────────┤            ▼
│ 🔹 Strategy B: Single-Tier Token Bucket       │    Background drain() loop
│    (In-Memory, uniform global quotas)         │            │
├───────────────────────────────────────────────┤            ▼
│ 🔹 Strategy C: Sliding Window Redis           │    [ Emitter.Emit(event) ]
│    (Distributed, Atomic Lua script)           │     ├── InMemoryEmitter
└───────────────────────────────────────────────┘     └── OpenTelemetry Emitter
│
▼
Return bool immediately

```

---

## 📂 Project Structure

```text
hayaku/
├── 📁 cmd/
│   └── 📁 hayaku/
│       └── main.go                   # Example entrypoint & server wiring
├── 📁 ratelimiter/
│   ├── limiter.go                    # RateLimiter & PolicyRateLimiter interfaces
│   ├── policy.go                     # Policy struct, validation & PolicyResolver
│   ├── policy_manager.go             # Multi-tier PolicyManager + sweeper lifecycle
│   ├── policy_test.go                # Dynamic migration & race-condition tests
│   ├── token_bucket.go               # Channel/Atomic token bucket implementation
│   ├── token_bucket_test.go          # Token bucket concurrency unit tests
│   ├── manager.go                    # Single-tier bucket manager
│   ├── sliding_window_redis.go       # Redis ZSET + atomic Lua script backend
│   └── sliding_window_redis_test.go  # Redis integration tests
├── 📁 metrics/
│   ├── event.go                      # RequestEvent & DenyReason definitions
│   ├── collector.go                  # Thread-safe MetricStore & Snapshots
│   └── instrumented.go               # InstrumentedLimiter decorator
├── 📁 internal/
│   ├── 📁 api/
│   │   └── handler.go                # HTTP middleware (429 Too Many Requests / 202)
│   └── 📁 core/
│       ├── job.go                    # Job execution interfaces
│       └── queue.go                  # Queue orchestration interfaces
├── go.mod
└── go.sum

```

---

## 🚀 Quickstart & Usage

### 1. Multi-Tier Policies (In-Memory)

Map incoming users or API keys directly to dynamic service tiers:

```go
package main

import (
	"fmt"

	"[github.com/FilledEther20/Hayaku/ratelimiter](https://github.com/FilledEther20/Hayaku/ratelimiter)"
)

func main() {
	limiter, err := ratelimiter.NewPolicyManager(ratelimiter.PolicyManagerConfig{
		DefaultPolicy: ratelimiter.Policy{
			Name:     "free",
			Capacity: 100, // Max burst tokens
			Rate:     2,   // Tokens refilled per second
		},
		Policies: []ratelimiter.Policy{
			{Name: "pro", Capacity: 1000, Rate: 20},
			{Name: "enterprise", Capacity: 10000, Rate: 200},
		},
		Resolver: func(userID string) string {
			// Extract tier from JWT claims, session, or database cache
			return getUserPlan(userID)
		},
	})
	if err != nil {
		panic(err)
	}

	// Evaluates quota against resolved policy
	if !limiter.Allow("user_9842") {
		fmt.Println("HTTP 429: Rate limit exceeded")
	}
}

```

> [!TIP]
> **Explicit Policy Overrides:** If rate limits depend on dynamic route cost instead of user tier, invoke policy evaluation directly:
> ```go
> allowed := limiter.AllowWithPolicy("user_9842", "enterprise")
> 
> ```
> 
> 

---

### 2. Role-Based Access Control (RBAC)

```go
limiter, err := ratelimiter.NewPolicyManager(ratelimiter.PolicyManagerConfig{
	DefaultPolicy: ratelimiter.Policy{Name: "guest", Capacity: 50, Rate: 1},
	Policies: []ratelimiter.Policy{
		{Name: "user", Capacity: 500, Rate: 10},
		{Name: "moderator", Capacity: 5000, Rate: 100},
		{Name: "admin", Capacity: 50000, Rate: 1000},
	},
	Resolver: func(userID string) string {
		return authService.GetRole(userID) // e.g. "admin", "moderator", "user"
	},
})

```

---

### 3. Distributed Sliding Window (Redis)

For horizontally scaled architectures requiring cluster-wide synchronization:

```go
package main

import (
	"time"

	"[github.com/FilledEther20/Hayaku/ratelimiter](https://github.com/FilledEther20/Hayaku/ratelimiter)"
	"[github.com/redis/go-redis/v9](https://github.com/redis/go-redis/v9)"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 5 requests allowed per 2-second sliding window
	limiter := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)

	allowed := limiter.Allow("user_123")
	if !allowed {
		// handle rate limit
	}
}

```

---

### 4. Zero-Overhead Async Telemetry

Wrap any standard `RateLimiter` with non-blocking metric collection:

```go
package main

import (
	"fmt"
	"time"

	"[github.com/FilledEther20/Hayaku/metrics](https://github.com/FilledEther20/Hayaku/metrics)"
	"[github.com/FilledEther20/Hayaku/ratelimiter](https://github.com/FilledEther20/Hayaku/ratelimiter)"
	"[github.com/redis/go-redis/v9](https://github.com/redis/go-redis/v9)"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	base := ratelimiter.NewSlidingWindowRedis(rdb, 2*time.Second, 5)

	// Buffer up to 1024 events in memory before non-blocking drop
	limiter := metrics.NewInstrumentedLimiter(base, 1024)
	defer limiter.Stop()

	// Record request with metadata
	allowed := limiter.Allow("user_123", "203.0.113.1", "IN")

	// Export thread-safe telemetry snapshot
	snap := limiter.GetMetrics()
	fmt.Printf("Total: %d | Allowed: %d | Rejected: %d\n", snap.Total, snap.Allowed, snap.Rejected)
	fmt.Println("Traffic by Region:", snap.ByRegion) // map[IN:9 US:3]
	fmt.Println("Denial Reasons:", snap.DenialReasons)
}

```

---

### 5. OpenTelemetry Integration *(Upcoming)*

```go
import (
	"go.opentelemetry.io/otel"
	hkotel "[github.com/FilledEther20/Hayaku/metrics/otel](https://github.com/FilledEther20/Hayaku/metrics/otel)"
)

limiter := metrics.NewInstrumentedLimiter(base, 1024,
	metrics.WithEmitter(hkotel.New(otel.GetMeterProvider())),
)
// Metrics pipe directly to Grafana / Datadog / Prometheus agent

```

---

## ⚙️ Rate Limiting Mechanics

| Strategy | Backend | Concurrency Model | Best For |
| --- | --- | --- | --- |
| **Multi-Policy Token Bucket** | In-Memory | Lock-free / RWMutex atomic sync | Multi-tenant SaaS with per-tier quotas |
| **Token Bucket** | In-Memory | Pure Go channel select / tickers | Single-instance microservices & Daemons |
| **Sliding Window** | Redis (ZSET) | Atomic Lua script execution | Distributed APIs & horizontally scaled pods |

### 🛠️ Redis Sliding Window Lua Lifecycle

When using `SlidingWindowRedis`, each request executes an atomic script:

1. `ZREMRANGEBYSCORE` — Evicts expired timestamps outside the rolling window.
2. `ZCARD` — Counts current requests within the active interval.
3. `ZADD` — Appends current millisecond timestamp if capacity permits.
4. `EXPIRE` — Refreshes key TTL to ensure dead keys are automatically purged.

---

## 📊 Telemetry Schema

Every request evaluated via `InstrumentedLimiter` collects:

| Field | Type | Description |
| --- | --- | --- |
| `Timestamp` | `time.Time` | UTC timestamp of request arrival |
| `UserID` | `string` | Identifier extracted from header or context |
| `IP` | `string` | Client IP address for geo and abuse tracking |
| `Region` | `string` | Edge region header (e.g. `CF-IPCountry`, `X-Region`) |
| `Allowed` | `bool` | Immediate evaluation verdict |
| `Reason` | `DenyReason` | `None`, `RateLimited`, `BackendDown`, `NetworkError` |
| `Latency` | `time.Duration` | Microsecond execution time of `Allow()` |

---

## 📡 HTTP Status Codes Reference

```http
POST /jobs/submit
Header: X-User-ID: usr_8371a

HTTP/1.1 202 Accepted             -> Request permitted and queued
HTTP/1.1 429 Too Many Requests    -> Rate limit quota exhausted
HTTP/1.1 503 Service Unavailable  -> Upstream executor or queue saturated

```

---

## 🛠️ Getting Started

### Prerequisites

* **Go**: `1.23` or higher
* **Redis**: `v7.0+` (optional, only required for distributed sliding window)

```bash
# Clone the repository
git clone [https://github.com/FilledEther20/Hayaku.git](https://github.com/FilledEther20/Hayaku.git)
cd Hayaku

# Install dependencies
go mod download

# Run tests
go test -v -race ./...

# Run example binary
go run ./cmd/hayaku

```

---

## 🗺️ Roadmap

* [x] Configurable multi-tier rate limiting policies (`ratelimiter.PolicyManager`)
* [x] In-memory Token Bucket with automated TTL sweeper
* [x] Distributed Redis Sliding Window with atomic Lua scripting
* [ ] OpenTelemetry native exporter module (`metrics/otel`)
* [ ] Functional options constructor pattern for `NewInstrumentedLimiter`
* [ ] Benchmark suite measuring p99 latency under 100k concurrent req/sec
* [ ] Automatic `NOSCRIPT` fallback and reload recovery on Redis restart

---

## 📦 Dependencies

| Dependency | Version | Purpose |
| --- | --- | --- |
| [`github.com/redis/go-redis/v9`](https://github.com/redis/go-redis) | `v9.x` | Redis client for distributed sliding window state |
| [`github.com/google/uuid`](https://github.com/google/uuid) | `v1.x` | High-entropy unique member IDs in Redis ZSET |

---

## 📄 License

Distributed under the **MIT License**. See `LICENSE` for more information.
