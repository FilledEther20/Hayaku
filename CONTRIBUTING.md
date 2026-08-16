# Contributing to Hayaku

Thanks for your interest! Here's how to get started.

## Prerequisites

- Go 1.23+
- Redis 7+ (for integration tests)

## Running tests

```bash
redis-server &
go test -race ./...
```

## Submitting changes

1. Fork the repository and create a branch from `main`.
2. Make your changes with tests covering the new behaviour.
3. Run `go vet ./...` and `go test -race ./...` — both must pass.
4. Open a pull request with a clear description of what changed and why.

## Reporting bugs

Open a GitHub issue with:
- Go version (`go version`)
- Redis version if relevant
- Minimal reproduction steps

## Code style

Follow standard Go conventions (`gofmt`, `go vet`). Keep comments short and meaningful.
