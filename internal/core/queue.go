package core

import "context"

type Queue interface {
	Enqueue(ctx context.Context, j Job) error
	Dequeue(ctx context.Context) (Job, error)
}
