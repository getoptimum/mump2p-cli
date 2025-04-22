package ratelimit

import "time"

// UsageData represents persistent usage metrics
type UsageData struct {
	PublishCount       int
	BytesPublished     int64
	LastReset          time.Time
	LastPublishTime    time.Time
	LastSubTime        time.Time
	LastSecondTime     time.Time
	SecondPublishCount int
}

// UsageStats represents usage statistics and rate limits
type UsageStats struct {
	PublishCount        int
	PublishLimitPerHour int
	PublishLimitPerSec  int
	SecondPublishCount  int
	BytesPublished      int64
	DailyQuota          int64
	NextReset           time.Time
	TimeUntilReset      time.Duration
	LastPublishTime     time.Time
	LastSubscribeTime   time.Time
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
