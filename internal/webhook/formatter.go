package webhook

import (
	"encoding/json"
	"net/url"
	"strings"
)

// WebhookType represents the type of webhook service
type WebhookType int

const (
	// WebhookTypeGeneric represents a generic HTTP endpoint
	WebhookTypeGeneric WebhookType = iota
	// WebhookTypeDiscord represents Discord webhook
	WebhookTypeDiscord
	// WebhookTypeSlack represents Slack webhook
	WebhookTypeSlack
)

// DiscordPayload represents the JSON structure for Discord webhooks
type DiscordPayload struct {
	Content string `json:"content"`
}

// SlackPayload represents the JSON structure for Slack webhooks
type SlackPayload struct {
	Text string `json:"text"`
}

// Formatter handles webhook payload formatting based on the webhook type
type Formatter struct {
	webhookType WebhookType
}

// NewFormatter creates a new webhook formatter based on the webhook URL
func NewFormatter(webhookURL string) *Formatter {
	webhookType := detectWebhookType(webhookURL)
	return &Formatter{
		webhookType: webhookType,
	}
}

// detectWebhookType determines the webhook type based on the URL
func detectWebhookType(webhookURL string) WebhookType {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return WebhookTypeGeneric
	}

	hostname := strings.ToLower(parsedURL.Hostname())

	// Check for Discord webhook
	if strings.Contains(hostname, "discord.com") || strings.Contains(hostname, "discordapp.com") {
		return WebhookTypeDiscord
	}

	// Check for Slack webhook
	if strings.Contains(hostname, "slack.com") || strings.Contains(hostname, "hooks.slack.com") {
		return WebhookTypeSlack
	}

	return WebhookTypeGeneric
}

// FormatMessage formats the raw message content according to the webhook type
func (f *Formatter) FormatMessage(message []byte) ([]byte, error) {
	messageStr := string(message)

	switch f.webhookType {
	case WebhookTypeDiscord:
		payload := DiscordPayload{
			Content: messageStr,
		}
		return json.Marshal(payload)

	case WebhookTypeSlack:
		payload := SlackPayload{
			Text: messageStr,
		}
		return json.Marshal(payload)

	case WebhookTypeGeneric:
		// For generic webhooks, return the raw message
		return message, nil

	default:
		return message, nil
	}
}

// GetWebhookType returns the detected webhook type
func (f *Formatter) GetWebhookType() WebhookType {
	return f.webhookType
}

// GetWebhookTypeName returns a human-readable name for the webhook type
func (f *Formatter) GetWebhookTypeName() string {
	switch f.webhookType {
	case WebhookTypeDiscord:
		return "Discord"
	case WebhookTypeSlack:
		return "Slack"
	case WebhookTypeGeneric:
		return "Generic"
	default:
		return "Unknown"
	}
}

// FormatMessageWithType is a convenience function that creates a formatter and formats a message
func FormatMessageWithType(webhookURL string, message []byte) ([]byte, error) {
	formatter := NewFormatter(webhookURL)
	return formatter.FormatMessage(message)
}
