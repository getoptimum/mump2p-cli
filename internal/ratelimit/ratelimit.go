package ratelimit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
)

// RateLimiter tracks and enforces rate limits
// The CLI records locally the limit, and the proxy records it as well.
type RateLimiter struct {
	mu          sync.Mutex
	tokenClaims *auth.TokenClaims
	usageFile   string
	usage       *UsageData
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(claims *auth.TokenClaims) (*RateLimiter, error) {
	return NewRateLimiterWithDir(claims, "")
}

// NewRateLimiterWithDir creates a new rate limiter with custom directory
func NewRateLimiterWithDir(claims *auth.TokenClaims, customDir string) (*RateLimiter, error) {
	var usageDir string

	if customDir != "" {
		usageDir = customDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine home directory: %v", err)
		}
		usageDir = filepath.Join(homeDir, ".mump2p")
	}

	identifier := "default"
	if claims.Subject != "" {
		identifier = claims.Subject
	}

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
	now := time.Now()
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

	// per-second check
	if now.Sub(r.usage.LastSecondTime) >= time.Second {
		r.usage.LastSecondTime = now
		r.usage.SecondPublishCount = 0
	}
	if r.usage.SecondPublishCount >= r.tokenClaims.MaxPublishPerSec {
		return fmt.Errorf("per-second limit reached (%d/sec)", r.tokenClaims.MaxPublishPerSec)
	}
	r.usage.SecondPublishCount++

	// Save the updated per-second counter
	if err := r.saveUsage(); err != nil {
		// Log error but don't fail the publish
		fmt.Printf("Warning: failed to save usage data: %v\n", err)
	}

	// Per-hour check
	if r.usage.PublishCount >= r.tokenClaims.MaxPublishPerHour {
		next := r.usage.LastReset.Add(24 * time.Hour)
		return fmt.Errorf("per-hour limit reached (%d/hour), resets in %s",
			r.tokenClaims.MaxPublishPerHour, time.Until(next).Round(time.Minute))
	}

	// daily quota
	if r.usage.BytesPublished+messageSize > r.tokenClaims.DailyQuota {
		next := r.usage.LastReset.Add(24 * time.Hour)
		return fmt.Errorf("daily quota exceeded (%d/%d bytes), resets in %s",
			r.usage.BytesPublished+messageSize, r.tokenClaims.DailyQuota, time.Until(next).Round(time.Minute))
	}

	return nil
}

// RecordPublish records a successful publish operation
func (r *RateLimiter) RecordPublish(size int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.usage.PublishCount++
	r.usage.BytesPublished += size
	r.usage.LastPublishTime = time.Now()
	return r.saveUsage()
}

// GetUsageStats returns current usage statistics
func (r *RateLimiter) GetUsageStats() UsageStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkAndResetCounters()

	next := r.usage.LastReset.Add(24 * time.Hour)

	return UsageStats{
		PublishCount:        r.usage.PublishCount,
		PublishLimitPerHour: r.tokenClaims.MaxPublishPerHour,
		PublishLimitPerSec:  r.tokenClaims.MaxPublishPerSec,
		SecondPublishCount:  r.usage.SecondPublishCount,
		BytesPublished:      r.usage.BytesPublished,
		DailyQuota:          r.tokenClaims.DailyQuota,
		NextReset:           next,
		TimeUntilReset:      time.Until(next).Truncate(time.Second),
		LastPublishTime:     r.usage.LastPublishTime,
		LastSubscribeTime:   r.usage.LastSubTime,
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
	data, err := os.ReadFile(r.usageFile)
	if err != nil {
		return nil, err
	}
	var u UsageData
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// saveUsage saves usage data to disk
func (r *RateLimiter) saveUsage() error {
	data, err := json.Marshal(r.usage)
	if err != nil {
		return err
	}
	return os.WriteFile(r.usageFile, data, 0600)
}
