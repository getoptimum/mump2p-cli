package ratelimit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getoptimum/optcli/internal/auth"
	"github.com/stretchr/testify/require"
)

func createTestClaims() *auth.TokenClaims {
	return &auth.TokenClaims{
		Subject:           "testuser",
		IsActive:          true,
		MaxPublishPerHour: 5,
		MaxPublishPerSec:  2,
		MaxMessageSize:    1024 * 1024,     // 1MB
		DailyQuota:        5 * 1024 * 1024, // 5MB
	}
}

func cleanupUsageFile(claims *auth.TokenClaims) {
	homeDir, _ := os.UserHomeDir()
	path := filepath.Join(homeDir, ".optimum", claims.Subject+"_usage.json")
	_ = os.Remove(path)
}

// TestRateLimiter test NewRateLimiter function.
func TestRateLimiter(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*RateLimiter)
		messageSize    int64
		expectErr      bool
		expectLimitErr string
	}{
		{
			name:        "valid publish within limits",
			messageSize: 512 * 1024, // 512KB
			expectErr:   false,
		},
		{
			name:           "exceeds message size limit",
			messageSize:    2 * 1024 * 1024, // 2MB
			expectErr:      true,
			expectLimitErr: "message size exceeds limit",
		},
		{
			name: "exceeds daily quota",
			setupFunc: func(r *RateLimiter) {
				r.usage.BytesPublished = r.tokenClaims.DailyQuota - (100 * 1024) // only 100KB remaining
			},
			messageSize:    512 * 1024, // 512KB (valid size)
			expectErr:      true,
			expectLimitErr: "daily quota exceeded",
		},
		{
			name: "exceeds publish per hour",
			setupFunc: func(r *RateLimiter) {
				r.usage.PublishCount = r.tokenClaims.MaxPublishPerHour
			},
			messageSize:    512 * 1024,
			expectErr:      true,
			expectLimitErr: "per-hour limit reached",
		},
		{
			name: "auto resets after 24h",
			setupFunc: func(r *RateLimiter) {
				r.usage.LastReset = time.Now().Add(-25 * time.Hour)
				r.usage.PublishCount = r.tokenClaims.MaxPublishPerHour
			},
			messageSize: 512 * 1024,
			expectErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims := createTestClaims()
			// clean slate before test
			cleanupUsageFile(claims)
			defer cleanupUsageFile(claims)

			rl, err := NewRateLimiter(claims)
			require.NoError(t, err)
			require.NotNil(t, rl)

			if tc.setupFunc != nil {
				tc.setupFunc(rl)
			}

			err = rl.CheckPublishAllowed(tc.messageSize)
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectLimitErr)
			} else {
				require.NoError(t, err)
				err = rl.RecordPublish(tc.messageSize)
				require.NoError(t, err)
			}
		})
	}
}
