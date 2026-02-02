package formatter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatYAML  OutputFormat = "yaml"
	FormatCSV   OutputFormat = "csv"
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
	case "csv":
		f.format = FormatCSV
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
	case FormatCSV:
		return f.formatCSV(data)
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

// formatCSV formats data as CSV (flattens structs/maps to one or more rows)
func (f *Formatter) formatCSV(data interface{}) (string, error) {
	rows, err := dataToCSVRows(data)
	if err != nil {
		return "", fmt.Errorf("failed to convert to CSV: %v", err)
	}
	if len(rows) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(rows); err != nil {
		return "", fmt.Errorf("failed to write CSV: %v", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

// dataToCSVRows converts interface{} to [][]string (header + data rows)
func dataToCSVRows(data interface{}) ([][]string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var raw interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, err
	}
	switch v := raw.(type) {
	case []interface{}:
		if len(v) == 0 {
			return nil, nil
		}
		allKeys := make(map[string]bool)
		var maps []map[string]string
		for _, item := range v {
			m, err := flattenToMap(item)
			if err != nil {
				return nil, err
			}
			for k := range m {
				allKeys[k] = true
			}
			maps = append(maps, m)
		}
		headers := sortedKeysFromSet(allKeys)
		rows := [][]string{headers}
		for _, m := range maps {
			row := make([]string, len(headers))
			for i, h := range headers {
				row[i] = m[h]
			}
			rows = append(rows, row)
		}
		return rows, nil
	case map[string]interface{}:
		m := flattenMap("", v)
		headers := sortedKeys(m)
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = m[h]
		}
		return [][]string{headers, row}, nil
	default:
		return nil, fmt.Errorf("unsupported type for CSV: %T", data)
	}
}

func flattenToMap(data interface{}) (map[string]string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, err
	}
	return flattenMap("", raw), nil
}

func sortedKeysFromSet(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func flattenMap(prefix string, m map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			for k2, v2 := range flattenMap(key, val) {
				out[k2] = v2
			}
		case []interface{}:
			parts := make([]string, len(val))
			for i, el := range val {
				parts[i] = fmt.Sprint(el)
			}
			out[key] = strings.Join(parts, ",")
		case nil:
			out[key] = ""
		default:
			out[key] = fmt.Sprint(val)
		}
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

// IsCSV returns true if the format is CSV
func (f *Formatter) IsCSV() bool {
	return f.format == FormatCSV
}
