package config

const (
	// Maximum publish message per hour.
	DefaultMaxPublishPerHour = 100
	// Maximum publish message per second.
	DefaultMaxPublishPerSec = 2
	// Maximum message size in bytes each operation.
	DefaultMaxMessageSize = 2 << 20 // 2MB
	// Maximum message bytes per day.
	DefaultDailyQuota = 100 << 20 // 100MB

)

// These will be injected at build time using -ldflags
var (
	Domain     string
	ClientID   string
	Audience   string
	ServiceURL string
)

// AuthConfig holds Auth0 configuration
type Config struct {
	AuthDomain   string
	AuthClientID string
	AuthAudience string
	ServiceUrl   string
}

// LoadConfig loads configuration from a file
func LoadConfig() *Config {
	return &Config{
		AuthDomain:   Domain,
		AuthClientID: ClientID,
		AuthAudience: Audience,
		ServiceUrl:   ServiceURL,
	}
}
