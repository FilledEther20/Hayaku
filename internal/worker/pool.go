package worker

import (
	"sync"

	"github.com/FilledEther20/Hayaku/internal/job"
)

type Pool struct {
	JobQueue   chan job.Job
	MaxWorkers int
	wg         sync.WaitGroup
}

func NewPool(maxWorkers int, queueSize int) *Pool {
	return &Pool{
		JobQueue:   make(chan job.Job, queueSize),
		MaxWorkers: maxWorkers,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.MaxWorkers; i++ {
		p.wg.Add(i)
		go p.worker(i)
	}
}

