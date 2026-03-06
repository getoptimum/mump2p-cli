package cmd

import (
	"net/http"
	"regexp"
	"time"
)

// httpClient is a shared HTTP client with a sensible timeout.
// Use this instead of http.DefaultClient to prevent indefinite hangs
// when the remote server is unreachable or slow to respond.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// extractIPFromURL extracts IP address from URL string
func extractIPFromURL(url string) string {
	ipRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	return ipRegex.FindString(url)
}
