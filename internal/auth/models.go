package auth

import "time"

// TokenResponse represents the Auth0 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IdToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// StoredToken represents the token stored locally
type StoredToken struct {
	Token        string    `yaml:"token"`
	RefreshToken string    `yaml:"refresh_token,omitempty"`
	ExpiresAt    time.Time `yaml:"expires_at"`
}

// DeviceCodeResponse represents the initial device code response
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenClaims represents the parsed JWT token claims
type TokenClaims struct {
	// Standard claims
	Subject   string
	IssuedAt  time.Time
	ExpiresAt time.Time

	// Custom claims for rate limiting
	IsActive       bool   // Is active allows/rejects the request
	MaxPublishRate int    // Max publish operations per hour
	MaxMessageSize int64  // Maximum message size in bytes
	DailyQuota     int64  // Maximum bytes per day
	ClientID       string // Client ID that requested the token
	LimitsSetAt    int64  // Timestamp when limits were set
}
