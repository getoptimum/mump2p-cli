package webhook

import (
	"encoding/json"
	"testing"
)

func TestDetectWebhookType(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected WebhookType
	}{
		{
			name:     "Discord webhook",
			url:      "https://discord.com/api/webhooks/123456789/abcdef",
			expected: WebhookTypeDiscord,
		},
		{
			name:     "Discord app webhook",
			url:      "https://discordapp.com/api/webhooks/123456789/abcdef",
			expected: WebhookTypeDiscord,
		},
		{
			name:     "Slack webhook",
			url:      "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expected: WebhookTypeSlack,
		},
		{
			name:     "Slack webhook with custom domain",
			url:      "https://slack.com/api/webhooks/123456789/abcdef",
			expected: WebhookTypeSlack,
		},
		{
			name:     "Generic webhook",
			url:      "https://webhook.site/12345678-1234-1234-1234-123456789abc",
			expected: WebhookTypeGeneric,
		},
		{
			name:     "Custom webhook",
			url:      "https://mycompany.com/webhook",
			expected: WebhookTypeGeneric,
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: WebhookTypeGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.url)
			if formatter.webhookType != tt.expected {
				t.Errorf("Expected webhook type %v, got %v", tt.expected, formatter.webhookType)
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name        string
		webhookURL  string
		message     []byte
		expected    string
		expectError bool
	}{
		{
			name:       "Discord webhook formatting",
			webhookURL: "https://discord.com/api/webhooks/123456789/abcdef",
			message:    []byte("Hello, Discord!"),
			expected:   `{"content":"Hello, Discord!"}`,
		},
		{
			name:       "Slack webhook formatting",
			webhookURL: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			message:    []byte("Hello, Slack!"),
			expected:   `{"text":"Hello, Slack!"}`,
		},
		{
			name:       "Generic webhook formatting",
			webhookURL: "https://webhook.site/12345678-1234-1234-1234-123456789abc",
			message:    []byte("Hello, Generic!"),
			expected:   "Hello, Generic!",
		},
		{
			name:       "Empty message",
			webhookURL: "https://discord.com/api/webhooks/123456789/abcdef",
			message:    []byte(""),
			expected:   `{"content":""}`,
		},
		{
			name:       "Message with special characters",
			webhookURL: "https://discord.com/api/webhooks/123456789/abcdef",
			message:    []byte("Hello \"World\" with quotes and \n newlines"),
			expected:   `{"content":"Hello \"World\" with quotes and \n newlines"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.webhookURL)
			result, err := formatter.FormatMessage(tt.message)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestGetWebhookTypeName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Discord webhook",
			url:      "https://discord.com/api/webhooks/123456789/abcdef",
			expected: "Discord",
		},
		{
			name:     "Slack webhook",
			url:      "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expected: "Slack",
		},
		{
			name:     "Generic webhook",
			url:      "https://webhook.site/12345678-1234-1234-1234-123456789abc",
			expected: "Generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.url)
			result := formatter.GetWebhookTypeName()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatMessageWithType(t *testing.T) {
	message := []byte("Test message")
	
	// Test Discord formatting
	result, err := FormatMessageWithType("https://discord.com/api/webhooks/123456789/abcdef", message)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	var discordPayload DiscordPayload
	if err := json.Unmarshal(result, &discordPayload); err != nil {
		t.Errorf("Failed to unmarshal Discord payload: %v", err)
	}
	
	if discordPayload.Content != "Test message" {
		t.Errorf("Expected content 'Test message', got %q", discordPayload.Content)
	}
	
	// Test Slack formatting
	result, err = FormatMessageWithType("https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX", message)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	var slackPayload SlackPayload
	if err := json.Unmarshal(result, &slackPayload); err != nil {
		t.Errorf("Failed to unmarshal Slack payload: %v", err)
	}
	
	if slackPayload.Text != "Test message" {
		t.Errorf("Expected text 'Test message', got %q", slackPayload.Text)
	}
}
