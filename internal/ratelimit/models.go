package ratelimit

import "time"

// UsageData represents persistent usage metrics
type UsageData struct {
	PublishCount    int       `json:"publish_count"`
	BytesPublished  int64     `json:"bytes_published"`
	LastReset       time.Time `json:"last_reset"`
	LastPublishTime time.Time `json:"last_publish_time,omitempty"`
	LastSubTime     time.Time `json:"last_subscribe_time,omitempty"`
}

// LimitError represents a rate limit exceeded error
type LimitError struct {
	Message      string
	LimitType    string // "publish", "message_size", "daily_quota"
	CurrentUsage interface{}
	Limit        interface{}
	ResetTime    time.Time
}

// Error returns the error message
func (e *LimitError) Error() string {
	return e.Message
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	_, ok := err.(*LimitError)
	return ok
}
