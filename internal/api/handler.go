package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/FilledEther20/Hayaku/internal/core"
	"github.com/FilledEther20/Hayaku/ratelimiter"
)

type HayakuHandler struct {
	Limiter ratelimiter.RateLimiter
	Queue   core.Queue
}

type jobRequest struct {
	Payload string `json:"payload"`
}

func (jr jobRequest) ID() string                      { return jr.Payload }
func (jr jobRequest) Execute(_ context.Context) error { return nil }

func parseJobFromRequest(r *http.Request) core.Job {
	var req jobRequest
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
	return req
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
