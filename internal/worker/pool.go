package worker

import (
	"sync"
)

type Pool struct {
	JobQueue   chan Job
	MaxWorkers int
	wg         sync.WaitGroup
}

func NewPool(maxWorkers int, queueSize int) *Pool {
	return &Pool{
		JobQueue:   make(chan Job, queueSize),
		MaxWorkers: maxWorkers,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.MaxWorkers; i++ {
		p.wg.Add(i)
	}
}
