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
