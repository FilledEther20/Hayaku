package job

import "time"

type JobID string

type Job struct {
	ID         JobID
	Payload    string
	CreatedAt  time.Time
	RetryCount int
}
