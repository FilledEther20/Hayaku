package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// PolicyRateLimiter extends RateLimiter to support explicit policy selection.
type PolicyRateLimiter interface {
	RateLimiter
	AllowWithPolicy(userID, policyName string) bool
}

type userPolicyBucket struct {
	bucket     *TokenBucket
	policyName string
	lastSeen   time.Time
	cancel     context.CancelFunc
}

// PolicyManager provides dynamic, multi-tier rate limiting with O(1) policy lookup.
type PolicyManager struct {
	mu       sync.RWMutex
	policies map[string]Policy
	buckets  map[string]*userPolicyBucket
	resolver PolicyResolver
}

// PolicyManagerConfig configures a PolicyManager instance.
type PolicyManagerConfig struct {
	DefaultPolicy Policy
	Policies      []Policy
	Resolver      PolicyResolver
}

// NewPolicyManager creates a thread-safe PolicyManager.
func NewPolicyManager(cfg PolicyManagerConfig) (*PolicyManager, error) {
	if cfg.DefaultPolicy.Name == "" {
		cfg.DefaultPolicy.Name = DefaultPolicyName
	}
	if err := cfg.DefaultPolicy.Validate(); err != nil {
		return nil, err
	}

	policyMap := make(map[string]Policy, len(cfg.Policies)+1)
	policyMap[cfg.DefaultPolicy.Name] = cfg.DefaultPolicy

	for _, p := range cfg.Policies {
		if err := p.Validate(); err != nil {
			return nil, err
		}
		policyMap[p.Name] = p
	}

	resolver := cfg.Resolver
	if resolver == nil {
		resolver = func(userID string) string {
			return DefaultPolicyName
		}
	}

	return &PolicyManager{
		policies: policyMap,
		buckets:  make(map[string]*userPolicyBucket),
		resolver: resolver,
	}, nil
}

// Allow resolves the policy for the user and checks token availability.
// Satisfies the RateLimiter interface for backward compatibility.
func (pm *PolicyManager) Allow(userID string) bool {
	policyName := pm.resolver(userID)
	return pm.AllowWithPolicy(userID, policyName)
}

// AllowWithPolicy evaluates rate limiting against a specific policy name.
func (pm *PolicyManager) AllowWithPolicy(userID, policyName string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	policy, exists := pm.policies[policyName]
	if !exists {
		policy = pm.policies[DefaultPolicyName]
		policyName = DefaultPolicyName
	}

	ub, exists := pm.buckets[userID]
	if !exists || ub.policyName != policyName {
		if exists && ub.cancel != nil {
			ub.cancel()
		}

		ctx, cancel := context.WithCancel(context.Background())
		bucket := NewTokenBucket(policy.Capacity, policy.Rate)
		bucket.Start(ctx)

		ub = &userPolicyBucket{
			bucket:     bucket,
			policyName: policyName,
			cancel:     cancel,
		}
		pm.buckets[userID] = ub
	}

	ub.lastSeen = time.Now()

	select {
	case <-ub.bucket.tokensPresent:
		return true
	default:
		return false
	}
}

// Sweep removes inactive user buckets older than ttl.
func (pm *PolicyManager) sweep(ttl time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	for userID, ub := range pm.buckets {
		if now.Sub(ub.lastSeen) > ttl {
			if ub.cancel != nil {
				ub.cancel()
			}
			delete(pm.buckets, userID)
		}
	}
}

// StartSweeper starts a background garbage-collection goroutine.
func (pm *PolicyManager) StartSweeper(ctx context.Context, sweepInterval, ttl time.Duration) {
	ticker := time.NewTicker(sweepInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pm.sweep(ttl)
			case <-ctx.Done():
				return
			}
		}
	}()
}
