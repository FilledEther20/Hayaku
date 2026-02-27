package api

import (
	"net/http"

	"github.com/FilledEther20/Hayaku/internal/core"
)

type HayakuHandler struct {
	Limiter core.RateLimiter
	Queue   core.Queue
}

func (h *HayakuHandler) HandleSubmitJob(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	if !h.Limiter.Allow(userID) {
		http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
		return
	}

	job := parseJobFromRequest(r)

	
	err := h.Queue.Enqueue(r.Context(), job)
	if err != nil {
		http.Error(w, "503 Service Unavailable (Queue Full)", http.StatusServiceUnavailable)
		return
	}


	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Job accepted"))
}
