package auth

import (
	"testing"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func generateFakeToken(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// dummy secret just to encode
	signed, _ := token.SignedString([]byte("fake"))
	return signed
}

// TestParseToken tests ParseToken function.
func TestParseToken(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name        string
		claims      jwt.MapClaims
		expectErr   bool
		expectProps TokenClaims
	}{
		{
			name: "full valid claims",
			claims: jwt.MapClaims{
				"sub":                  "user123",
				"iat":                  float64(now),
				"exp":                  float64(now + 3600),
				"client_id":            "cli-app",
				"limits_set_at":        float64(now - 100),
				"is_active":            true,
				"max_publish_per_hour": float64(100),
				"max_publish_per_sec":  float64(2),
				"max_message_size":     float64(1048576),
				"daily_quota":          float64(1073741824),
			},
			expectErr: false,
			expectProps: TokenClaims{
				Subject:           "user123",
				ClientID:          "cli-app",
				IsActive:          true,
				MaxPublishPerHour: 100,
				MaxPublishPerSec:  2,
				MaxMessageSize:    1048576,
				DailyQuota:        1073741824,
			},
		},
		{
			name: "missing optional claims",
			claims: jwt.MapClaims{
				"sub":       "anon",
				"is_active": false,
			},
			expectErr: false,
			expectProps: TokenClaims{
				Subject:           "anon",
				IsActive:          false,
				MaxPublishPerHour: config.DefaultMaxPublishPerHour,
				MaxPublishPerSec:  config.DefaultMaxPublishPerSec,
				MaxMessageSize:    config.DefaultMaxMessageSize,
				DailyQuota:        config.DefaultDailyQuota,
			},
		},
		{
			name:      "malformed token",
			claims:    nil,
			expectErr: true,
		},
	}

	parser := NewTokenParser()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var tokenStr string
			if tc.claims != nil {
				tokenStr = generateFakeToken(tc.claims)
			} else {
				tokenStr = "not.a.valid.token"
			}

			claims, err := parser.ParseToken(tokenStr)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, claims)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				require.Equal(t, tc.expectProps.Subject, claims.Subject)
				require.Equal(t, tc.expectProps.ClientID, claims.ClientID)
				require.Equal(t, tc.expectProps.IsActive, claims.IsActive)
				require.Equal(t, tc.expectProps.MaxPublishPerHour, claims.MaxPublishPerHour)
				require.Equal(t, tc.expectProps.MaxPublishPerSec, claims.MaxPublishPerSec)
				require.Equal(t, tc.expectProps.MaxMessageSize, claims.MaxMessageSize)
				require.Equal(t, tc.expectProps.DailyQuota, claims.DailyQuota)
			}
		})
	}
}
