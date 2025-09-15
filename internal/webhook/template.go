package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"
)

// WebhookData represents the data available in webhook templates
type WebhookData struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Topic     string    `json:"topic"`
	ClientID  string    `json:"client_id"`
	MessageID string    `json:"message_id"`
}

// TemplateFormatter handles webhook payload formatting using Go templates
type TemplateFormatter struct {
	template *template.Template
}

// NewTemplateFormatter creates a new template formatter
func NewTemplateFormatter(schema string) (*TemplateFormatter, error) {
	if schema == "" {
		// Default: return raw message (no formatting)
		return &TemplateFormatter{}, nil
	}

	// Parse the template
	tmpl, err := template.New("webhook").Parse(schema)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook schema template: %v", err)
	}

	return &TemplateFormatter{template: tmpl}, nil
}

// FormatMessage formats the message using the template
func (tf *TemplateFormatter) FormatMessage(message []byte, topic, clientID, messageID string) ([]byte, error) {
	// If no template, return raw message
	if tf.template == nil {
		return message, nil
	}

	// Prepare template data
	data := WebhookData{
		Message:   string(message),
		Timestamp: time.Now().UTC(),
		Topic:     topic,
		ClientID:  clientID,
		MessageID: messageID,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tf.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute webhook template: %v", err)
	}

	// Try to parse as JSON to validate
	var jsonData interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonData); err != nil {
		return nil, fmt.Errorf("webhook template must produce valid JSON: %v", err)
	}

	return buf.Bytes(), nil
}

// GetDefaultSchemas returns common webhook schemas for services
func GetDefaultSchemas() map[string]string {
	return map[string]string{
		"discord": `{"content":"{{.Message}}"}`,
		"slack":   `{"text":"{{.Message}}"}`,
		"generic": `{"message":"{{.Message}}","timestamp":"{{.Timestamp}}","topic":"{{.Topic}}"}`,
	}
}

// FormatMessageWithSchema is a convenience function for one-time formatting
func FormatMessageWithSchema(schema string, message []byte, topic, clientID, messageID string) ([]byte, error) {
	formatter, err := NewTemplateFormatter(schema)
	if err != nil {
		return nil, err
	}
	return formatter.FormatMessage(message, topic, clientID, messageID)
}
