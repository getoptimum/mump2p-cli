package cmd

import (
	"regexp"
)

// extractIPFromURL extracts IP address from URL string
func extractIPFromURL(url string) string {
	ipRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	return ipRegex.FindString(url)
}
