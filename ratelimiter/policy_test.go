package ratelimiter

import (
	"fmt"
	"sync"
	"testing"
)

func TestPolicyValidation(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{"valid policy", Policy{Name: "pro", Capacity: 100, Rate: 10}, false},
		{"empty name", Policy{Name: "", Capacity: 10, Rate: 5}, true},
		{"zero capacity", Policy{Name: "free", Capacity: 0, Rate: 5}, true},
		{"negative rate", Policy{Name: "tier", Capacity: 10, Rate: -1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.policy.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolicyManager_AllowAndFallback(t *testing.T) {
	cfg := PolicyManagerConfig{
		DefaultPolicy: Policy{Name: "default", Capacity: 2, Rate: 1},
		Policies: []Policy{
			{Name: "pro", Capacity: 5, Rate: 1},
		},
		Resolver: func(userID string) string {
			if userID == "user-pro" {
				return "pro"
			}
			if userID == "user-unknown-policy" {
				return "non-existent"
			}
			return DefaultPolicyName
		},
	}

	pm, err := NewPolicyManager(cfg)
	if err != nil {
		t.Fatalf("failed to create PolicyManager: %v", err)
	}

	// Default user burst capacity (2)
	if !pm.Allow("user-default") || !pm.Allow("user-default") {
		t.Errorf("expected 2 requests allowed for default user")
	}
	if pm.Allow("user-default") {
		t.Errorf("expected 3rd request to be rate limited")
	}

	// Pro user burst capacity (5)
	for i := 0; i < 5; i++ {
		if !pm.Allow("user-pro") {
			t.Errorf("expected request %d to be allowed for pro user", i+1)
		}
	}
	if pm.Allow("user-pro") {
		t.Errorf("expected 6th request to be rate limited for pro user")
	}

	// Unknown policy fallback to default capacity (2)
	if !pm.Allow("user-unknown-policy") || !pm.Allow("user-unknown-policy") {
		t.Errorf("expected fallback policy to allow 2 requests")
	}
	if pm.Allow("user-unknown-policy") {
		t.Errorf("expected fallback policy 3rd request to be limited")
	}
}

func TestPolicyManager_DynamicPolicySwitching(t *testing.T) {
	currentPolicy := "free"
	pm, err := NewPolicyManager(PolicyManagerConfig{
		DefaultPolicy: Policy{Name: "free", Capacity: 2, Rate: 1},
		Policies: []Policy{
			{Name: "pro", Capacity: 10, Rate: 2},
		},
		Resolver: func(userID string) string {
			return currentPolicy
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize PolicyManager: %v", err)
	}

	// Exhaust free tier
	pm.Allow("user1")
	pm.Allow("user1")
	if pm.Allow("user1") {
		t.Fatalf("expected rate limit on free tier")
	}

	// Upgrade user to pro
	currentPolicy = "pro"
	if !pm.Allow("user1") {
		t.Errorf("expected immediate capacity after upgrading policy")
	}
}

func TestPolicyManager_ConcurrentAccess(t *testing.T) {
	pm, err := NewPolicyManager(PolicyManagerConfig{
		DefaultPolicy: Policy{Name: "default", Capacity: 1000, Rate: 100},
		Policies: []Policy{
			{Name: "tier-a", Capacity: 2000, Rate: 200},
			{Name: "tier-b", Capacity: 500, Rate: 50},
		},
		Resolver: func(userID string) string {
			if len(userID)%2 == 0 {
				return "tier-a"
			}
			return "tier-b"
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating policy manager: %v", err)
	}

	var wg sync.WaitGroup
	workers := 50
	requestsPerWorker := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			userID := fmt.Sprintf("user-%d", workerID%10)
			for j := 0; j < requestsPerWorker; j++ {
				_ = pm.Allow(userID)
			}
		}(i)
	}

	wg.Wait()
}
