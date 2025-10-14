package formatter

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatYAML  OutputFormat = "yaml"
)

// Formatter handles output formatting for different formats
type Formatter struct {
	format OutputFormat
}

// New creates a new formatter with the specified format
func New(format string) *Formatter {
	f := &Formatter{
		format: FormatTable, // default
	}

	switch strings.ToLower(format) {
	case "json":
		f.format = FormatJSON
	case "yaml", "yml":
		f.format = FormatYAML
	case "table", "":
		f.format = FormatTable
	}

	return f
}

// Format formats the data according to the configured format
func (f *Formatter) Format(data interface{}) (string, error) {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(data)
	case FormatYAML:
		return f.formatYAML(data)
	case FormatTable:
		// For table format, data should already be formatted as string
		if str, ok := data.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", data), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", f.format)
	}
}

// formatJSON formats data as JSON
func (f *Formatter) formatJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}
	return string(jsonBytes), nil
}

// formatYAML formats data as YAML
func (f *Formatter) formatYAML(data interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %v", err)
	}
	return string(yamlBytes), nil
}

// IsTable returns true if the format is table
func (f *Formatter) IsTable() bool {
	return f.format == FormatTable
}

// IsJSON returns true if the format is JSON
func (f *Formatter) IsJSON() bool {
	return f.format == FormatJSON
}

// IsYAML returns true if the format is YAML
func (f *Formatter) IsYAML() bool {
	return f.format == FormatYAML
}
