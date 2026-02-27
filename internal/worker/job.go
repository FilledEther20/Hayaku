package worker

import "context"

type Job interface {
	ID() string
	Execute(ctx context.Context) error
}


