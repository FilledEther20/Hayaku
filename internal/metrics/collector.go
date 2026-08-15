package metrics

import (
	"sync"
)

type MetricStore struct {
	mu            sync.RWMutex
	total         int64
	allowed       int64
	rejected      int64
	byRegion      map[string]int64
	byHour        map[int]int64
	denialReasons map[DenyReasonEnum]int64
}

type Snapshot struct {
	Total         int64
	Allowed       int64
	Rejected      int64
	ByRegion      map[string]int64
	ByHour        map[int]int64
	DenialReasons map[DenyReasonEnum]int64
}

func NewMetricStore() *MetricStore {
	return &MetricStore{
		byRegion:      make(map[string]int64),
		byHour:        make(map[int]int64),
		denialReasons: make(map[DenyReasonEnum]int64),
	}
}

// Required to log the event.
func (ms *MetricStore) Record(e RequestEvent) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.total++
	if e.Allowed {
		ms.allowed++
	} else {
		ms.rejected++
		ms.denialReasons[e.Reason]++
	}

	if e.Region != "" {
		ms.byRegion[e.Region]++
	}

	hour := e.Timestamp.UTC().Hour()
	ms.byHour[hour]++
}

// Required to read the log.
func (ms *MetricStore) Snapshot() Snapshot {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	regions := make(map[string]int64, len(ms.byRegion))
	hours := make(map[int]int64, len(ms.byHour))
	reasons := make(map[DenyReasonEnum]int64, len(ms.denialReasons))

	for k, v := range ms.byRegion {
		regions[k] = v
	}

	for k, v := range ms.byHour {
		hours[k] = v
	}

	for k, v := range ms.denialReasons {
		reasons[k] = v
	}

	return Snapshot{
		Total:         ms.total,
		Allowed:       ms.allowed,
		Rejected:      ms.rejected,
		ByRegion:      regions,
		ByHour:        hours,
		DenialReasons: reasons,
	}
}
