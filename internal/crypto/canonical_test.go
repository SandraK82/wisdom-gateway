package crypto

import (
	"testing"
)

func TestCanonicalJSON_NoHTMLEscaping(t *testing.T) {
	// Go's json.Marshal escapes & as \u0026 by default.
	// Our canonical JSON must NOT do this, to match JavaScript and Rust behavior.
	fields := map[string]interface{}{
		"content": "R+D 100 Award & more <details>",
		"name":    "test",
	}

	result, err := CanonicalJSON(fields)
	if err != nil {
		t.Fatalf("CanonicalJSON failed: %v", err)
	}

	expected := `{"content":"R+D 100 Award & more <details>","name":"test"}`
	if result != expected {
		t.Errorf("HTML escaping detected.\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestCanonicalJSON_SortedKeys(t *testing.T) {
	fields := map[string]interface{}{
		"zebra": "last",
		"alpha": "first",
		"mid":   "middle",
	}

	result, err := CanonicalJSON(fields)
	if err != nil {
		t.Fatalf("CanonicalJSON failed: %v", err)
	}

	expected := `{"alpha":"first","mid":"middle","zebra":"last"}`
	if result != expected {
		t.Errorf("Keys not sorted.\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestCanonicalJSON_NestedObjects(t *testing.T) {
	fields := map[string]interface{}{
		"creator": map[string]interface{}{
			"server_port": "hub:443",
			"entity":      "uuid-123",
			"domain":      "AGENT",
		},
		"content": "test & verify",
	}

	result, err := CanonicalJSON(fields)
	if err != nil {
		t.Fatalf("CanonicalJSON failed: %v", err)
	}

	expected := `{"content":"test & verify","creator":{"domain":"AGENT","entity":"uuid-123","server_port":"hub:443"}}`
	if result != expected {
		t.Errorf("Nested canonical JSON wrong.\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestCanonicalJSON_Arrays(t *testing.T) {
	fields := map[string]interface{}{
		"tags": []interface{}{
			map[string]interface{}{
				"domain": "TAG",
				"entity": "tag-1",
			},
		},
	}

	result, err := CanonicalJSON(fields)
	if err != nil {
		t.Fatalf("CanonicalJSON failed: %v", err)
	}

	expected := `{"tags":[{"domain":"TAG","entity":"tag-1"}]}`
	if result != expected {
		t.Errorf("Array handling wrong.\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestCanonicalJSON_NullAndBool(t *testing.T) {
	fields := map[string]interface{}{
		"active":    true,
		"transform": nil,
	}

	result, err := CanonicalJSON(fields)
	if err != nil {
		t.Fatalf("CanonicalJSON failed: %v", err)
	}

	expected := `{"active":true,"transform":null}`
	if result != expected {
		t.Errorf("Null/bool handling wrong.\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestCanonicalJSON_FloatRounding(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "simple float",
			input:    map[string]interface{}{"confidence": 0.5},
			expected: `{"confidence":0.5}`,
		},
		{
			name:     "repeating decimal",
			input:    map[string]interface{}{"confidence": 0.5555},
			expected: `{"confidence":0.5555}`,
		},
		{
			name:     "small float",
			input:    map[string]interface{}{"confidence": 0.0005},
			expected: `{"confidence":0.0005}`,
		},
		{
			name:     "integer as float",
			input:    map[string]interface{}{"confidence": 1.0},
			expected: `{"confidence":1}`,
		},
		{
			name:     "zero",
			input:    map[string]interface{}{"confidence": 0.0},
			expected: `{"confidence":0}`,
		},
		{
			name:     "negative float",
			input:    map[string]interface{}{"trust": -0.75},
			expected: `{"trust":-0.75}`,
		},
		{
			name:     "very precise float",
			input:    map[string]interface{}{"score": 0.123456789},
			expected: `{"score":0.123456789}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("CanonicalJSON failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Float rounding issue.\ngot:  %s\nwant: %s", result, tt.expected)
			}
		})
	}
}

func TestCanonicalJSON_EmptyStructures(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "empty object",
			input:    map[string]interface{}{},
			expected: `{}`,
		},
		{
			name:     "empty array",
			input:    map[string]interface{}{"tags": []interface{}{}},
			expected: `{"tags":[]}`,
		},
		{
			name:     "empty string",
			input:    map[string]interface{}{"content": ""},
			expected: `{"content":""}`,
		},
		{
			name:     "nested empty",
			input:    map[string]interface{}{"meta": map[string]interface{}{}},
			expected: `{"meta":{}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("CanonicalJSON failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("wrong.\ngot:  %s\nwant: %s", result, tt.expected)
			}
		})
	}
}

func TestCanonicalJSON_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "unicode",
			input:    map[string]interface{}{"content": "Bérard's Odyssey — «quotes»"},
			expected: `{"content":"Bérard's Odyssey — «quotes»"}`,
		},
		{
			name:     "newlines",
			input:    map[string]interface{}{"content": "line1\nline2"},
			expected: `{"content":"line1\nline2"}`,
		},
		{
			name:     "tabs",
			input:    map[string]interface{}{"content": "col1\tcol2"},
			expected: `{"content":"col1\tcol2"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("CanonicalJSON failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("wrong.\ngot:  %s\nwant: %s", result, tt.expected)
			}
		})
	}
}
