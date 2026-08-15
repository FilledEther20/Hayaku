package metrics

import "time"

type DenyReasonEnum string

const (
	None         DenyReasonEnum = "None"
	RateLimited  DenyReasonEnum = "Rate Limited"
	BackendDown  DenyReasonEnum = "Backend down"
	NetworkError DenyReasonEnum = "Network Error"
)

type RequestEvent struct {
	Timestamp time.Time
	IP        string
	UserID    string
	Region    string
	Allowed   bool
	Reason    DenyReasonEnum
	Latency   time.Duration
}
