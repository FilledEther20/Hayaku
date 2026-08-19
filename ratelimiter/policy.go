package ratelimiter

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrEmptyPolicyName indicates that a policy was registered without a name.
	ErrEmptyPolicyName = errors.New("policy name cannot be empty")
	// ErrInvalidCapacity indicates that capacity is less than or equal to zero.
	ErrInvalidCapacity = errors.New("policy capacity must be greater than zero")
	// ErrInvalidRate indicates that refill rate is less than or equal to zero.
	ErrInvalidRate = errors.New("policy refill rate must be greater than zero")
	// ErrDefaultPolicyMissing indicates that no default policy was provided.
	ErrDefaultPolicyMissing = errors.New("default policy must be configured")
)

// DefaultPolicyName defines the fallback policy identifier.
const DefaultPolicyName = "default"

// Policy defines rate limiting parameters for a tier or class of requests.
type Policy struct {
	Name     string        `json:"name" yaml:"name"`
	Capacity int64         `json:"capacity" yaml:"capacity"`
	Rate     int64         `json:"rate" yaml:"rate"` // tokens refilled per second
	Window   time.Duration `json:"window,omitempty" yaml:"window,omitempty"`
}

// Validate verifies that the policy parameters are positive and valid.
func (p Policy) Validate() error {
	if p.Name == "" {
		return ErrEmptyPolicyName
	}
	if p.Capacity <= 0 {
		return fmt.Errorf("%w: policy %q capacity is %d", ErrInvalidCapacity, p.Name, p.Capacity)
	}
	if p.Rate <= 0 {
		return fmt.Errorf("%w: policy %q rate is %d", ErrInvalidRate, p.Name, p.Rate)
	}
	return nil
}

// PolicyResolver dynamically resolves the policy name for an incoming identifier.
type PolicyResolver func(userID string) string
