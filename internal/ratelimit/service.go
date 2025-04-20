package ratelimit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getoptimum/optcli/internal/auth"
)

// RateLimiter tracks and enforces rate limits
// The CLI records locally the limit, and the gateway records it as well.
type RateLimiter struct {
	mu          sync.Mutex
	tokenClaims *auth.TokenClaims
	usageFile   string
	usage       *UsageData
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(claims *auth.TokenClaims) (*RateLimiter, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %v", err)
	}

	identifier := "default"
	if claims.Subject != "" {
		identifier = claims.Subject
	}

	usageDir := filepath.Join(homeDir, ".optimum")
	usageFile := filepath.Join(usageDir, fmt.Sprintf("%s_usage.json", identifier))

	if err := os.MkdirAll(usageDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create usage directory: %v", err)
	}

	limiter := &RateLimiter{
		tokenClaims: claims,
		usageFile:   usageFile,
	}

	usage, err := limiter.loadUsage()
	if err != nil {
		// If file doesn't exist or is corrupted, create new usage data
		usage = &UsageData{
			LastReset: time.Now(),
		}
	}

	limiter.usage = usage

	limiter.checkAndResetCounters()

	return limiter, nil
}

// CheckPublishAllowed verifies if a publish operation is allowed
func (r *RateLimiter) CheckPublishAllowed(messageSize int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// reset counters if a day has passed
	r.checkAndResetCounters()

	// check if the user is active
	if !r.tokenClaims.IsActive {
		return fmt.Errorf("your token is inactive; please contact support or check your subscription status")
	}

	// message size limit
	if messageSize > r.tokenClaims.MaxMessageSize {
		return fmt.Errorf("message size exceeds limit of %d bytes", r.tokenClaims.MaxMessageSize)
	}

	// publish rate limit
	if r.usage.PublishCount >= r.tokenClaims.MaxPublishRate {
		nextReset := r.usage.LastReset.Add(24 * time.Hour)
		timeLeft := time.Until(nextReset).Round(time.Minute)
		return fmt.Errorf("publish rate limit reached (%d/%d), resets in %s",
			r.usage.PublishCount, r.tokenClaims.MaxPublishRate, timeLeft)
	}

	// daily quota
	if r.usage.BytesPublished+messageSize > r.tokenClaims.DailyQuota {
		nextReset := r.usage.LastReset.Add(24 * time.Hour)
		timeLeft := time.Until(nextReset).Round(time.Minute)
		return fmt.Errorf("daily quota exceeded (%d/%d bytes), resets in %s",
			r.usage.BytesPublished, r.tokenClaims.DailyQuota, timeLeft)
	}

	return nil
}

// RecordPublish records a successful publish operation
func (r *RateLimiter) RecordPublish(messageSize int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.usage.PublishCount++
	r.usage.BytesPublished += messageSize
	r.usage.LastPublishTime = time.Now()

	return r.saveUsage()
}

// GetUsageStats returns current usage statistics
func (r *RateLimiter) GetUsageStats() UsageStats {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checkAndResetCounters()

	nextReset := r.usage.LastReset.Add(24 * time.Hour)

	return UsageStats{
		PublishCount:      r.usage.PublishCount,
		PublishLimit:      r.tokenClaims.MaxPublishRate,
		BytesPublished:    r.usage.BytesPublished,
		DailyQuota:        r.tokenClaims.DailyQuota,
		NextReset:         nextReset,
		TimeUntilReset:    time.Until(nextReset).Round(time.Minute),
		LastPublishTime:   r.usage.LastPublishTime,
		LastSubscribeTime: r.usage.LastSubTime,
	}
}

// checkAndResetCounters resets usage counters if a day has passed
func (r *RateLimiter) checkAndResetCounters() {
	if time.Since(r.usage.LastReset) > 24*time.Hour {
		r.usage.PublishCount = 0
		r.usage.BytesPublished = 0
		r.usage.LastReset = time.Now()

		// save reset counters
		_ = r.saveUsage() // Ignore error, we'll try again later
	}
}

// loadUsage loads usage data from disk
func (r *RateLimiter) loadUsage() (*UsageData, error) {
	// check if file exists
	if _, err := os.Stat(r.usageFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("usage file not found")
	}

	data, err := os.ReadFile(r.usageFile)
	if err != nil {
		return nil, fmt.Errorf("error reading usage file: %v", err)
	}

	var usage UsageData
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, fmt.Errorf("error parsing usage data: %v", err)
	}

	return &usage, nil
}

// saveUsage saves usage data to disk
func (r *RateLimiter) saveUsage() error {
	data, err := json.Marshal(r.usage)
	if err != nil {
		return fmt.Errorf("error encoding usage data: %v", err)
	}

	if err := os.WriteFile(r.usageFile, data, 0600); err != nil {
		return fmt.Errorf("error saving usage data: %v", err)
	}

	return nil
}
