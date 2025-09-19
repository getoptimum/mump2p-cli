package webhook

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewTemplateFormatter(t *testing.T) {
	tests := []struct {
		name        string
		schema      string
		expectError bool
	}{
		{
			name:        "Empty schema (raw message)",
			schema:      "",
			expectError: false,
		},
		{
			name:        "Valid Discord schema",
			schema:      `{"content":"{{.Message}}"}`,
			expectError: false,
		},
		{
			name:        "Valid Slack schema",
			schema:      `{"text":"{{.Message}}"}`,
			expectError: false,
		},
		{
			name:        "Complex schema with multiple fields",
			schema:      `{"message":"{{.Message}}","timestamp":"{{.Timestamp}}","topic":"{{.Topic}}"}`,
			expectError: false,
		},
		{
			name:        "Invalid JSON template",
			schema:      `{"content":"{{.Message"}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := NewTemplateFormatter(tt.schema)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && formatter == nil {
				t.Error("Expected formatter but got nil")
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		message   []byte
		topic     string
		clientID  string
		messageID string
		expected  string
	}{
		{
			name:      "Empty schema returns raw message",
			schema:    "",
			message:   []byte("Hello World"),
			topic:     "test",
			clientID:  "user123",
			messageID: "msg456",
			expected:  "Hello World",
		},
		{
			name:      "Discord schema",
			schema:    `{"content":"{{.Message}}"}`,
			message:   []byte("Hello Discord!"),
			topic:     "test",
			clientID:  "user123",
			messageID: "msg456",
			expected:  `{"content":"Hello Discord!"}`,
		},
		{
			name:      "Slack schema",
			schema:    `{"text":"{{.Message}}"}`,
			message:   []byte("Hello Slack!"),
			topic:     "test",
			clientID:  "user123",
			messageID: "msg456",
			expected:  `{"text":"Hello Slack!"}`,
		},
		{
			name:      "Complex schema with all fields",
			schema:    `{"message":"{{.Message}}","topic":"{{.Topic}}","client":"{{.ClientID}}","id":"{{.MessageID}}"}`,
			message:   []byte("Test message"),
			topic:     "alerts",
			clientID:  "user123",
			messageID: "msg456",
			expected:  `{"message":"Test message","topic":"alerts","client":"user123","id":"msg456"}`,
		},
		{
			name:      "Message with special characters",
			schema:    `{"content":"{{.Message}}"}`,
			message:   []byte("Hello World with special chars"),
			topic:     "test",
			clientID:  "user123",
			messageID: "msg456",
			expected:  `{"content":"Hello World with special chars"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := NewTemplateFormatter(tt.schema)
			if err != nil {
				t.Fatalf("Failed to create formatter: %v", err)
			}

			result, err := formatter.FormatMessage(tt.message, tt.topic, tt.clientID, tt.messageID)
			if err != nil {
				t.Fatalf("Failed to format message: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}

			// Validate JSON if not empty schema
			if tt.schema != "" {
				var jsonData interface{}
				if err := json.Unmarshal(result, &jsonData); err != nil {
					t.Errorf("Result is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestGetDefaultSchemas(t *testing.T) {
	schemas := GetDefaultSchemas()

	expectedSchemas := []string{"discord", "slack", "generic"}
	for _, schemaName := range expectedSchemas {
		if _, exists := schemas[schemaName]; !exists {
			t.Errorf("Expected schema %s not found", schemaName)
		}
	}

	// Test that each schema is valid JSON
	for name, schema := range schemas {
		formatter, err := NewTemplateFormatter(schema)
		if err != nil {
			t.Errorf("Default schema %s is invalid: %v", name, err)
		}

		// Test with sample data
		_, err = formatter.FormatMessage([]byte("test"), "test-topic", "test-client", "test-msg")
		if err != nil {
			t.Errorf("Default schema %s failed to format: %v", name, err)
		}
	}
}

func TestFormatMessageWithSchema(t *testing.T) {
	message := []byte("Test message")
	topic := "test-topic"
	clientID := "test-client"
	messageID := "test-msg"

	// Test Discord schema
	result, err := FormatMessageWithSchema(`{"content":"{{.Message}}"}`, message, topic, clientID, messageID)
	if err != nil {
		t.Fatalf("Failed to format with Discord schema: %v", err)
	}

	var discordPayload map[string]interface{}
	if err := json.Unmarshal(result, &discordPayload); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	if discordPayload["content"] != "Test message" {
		t.Errorf("Expected content 'Test message', got %v", discordPayload["content"])
	}

	// Test empty schema (raw message)
	result, err = FormatMessageWithSchema("", message, topic, clientID, messageID)
	if err != nil {
		t.Fatalf("Failed to format with empty schema: %v", err)
	}

	if string(result) != "Test message" {
		t.Errorf("Expected raw message 'Test message', got %q", string(result))
	}
}

func TestTemplateWithTimestamp(t *testing.T) {
	schema := `{"message":"{{.Message}}","timestamp":"{{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}"}`
	formatter, err := NewTemplateFormatter(schema)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	result, err := formatter.FormatMessage([]byte("Test"), "test", "client", "msg")
	if err != nil {
		t.Fatalf("Failed to format message: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(result, &payload); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Check that timestamp is present and valid
	if timestamp, exists := payload["timestamp"]; !exists {
		t.Error("Timestamp field not found in result")
	} else {
		// Try to parse the timestamp
		if _, err := time.Parse(time.RFC3339, timestamp.(string)); err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	}
}
