# Hayaku

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


##\ High-Level Architecture
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
