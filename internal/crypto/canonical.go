// Package crypto provides Ed25519 signature verification and canonical JSON generation.
package crypto

import (
	"encoding/json"
	"fmt"
	"sort"
)

// CanonicalJSON creates a deterministic JSON string from a map.
// Keys are sorted alphabetically to match the MCP and Hub implementations.
func CanonicalJSON(fields map[string]interface{}) (string, error) {
	return canonicalValue(fields)
}

// canonicalValue recursively produces canonical JSON for any value.
func canonicalValue(v interface{}) (string, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := "{"
		for i, k := range keys {
			if i > 0 {
				result += ","
			}
			keyJSON, err := json.Marshal(k)
			if err != nil {
				return "", fmt.Errorf("failed to marshal key %q: %w", k, err)
			}
			valJSON, err := canonicalValue(val[k])
			if err != nil {
				return "", fmt.Errorf("failed to canonicalize value for key %q: %w", k, err)
			}
			result += string(keyJSON) + ":" + valJSON
		}
		result += "}"
		return result, nil

	case []interface{}:
		result := "["
		for i, item := range val {
			if i > 0 {
				result += ","
			}
			itemJSON, err := canonicalValue(item)
			if err != nil {
				return "", err
			}
			result += itemJSON
		}
		result += "]"
		return result, nil

	default:
		// For primitives (string, number, bool, null), use standard JSON encoding
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value: %w", err)
		}
		return string(b), nil
	}
}

// AddressToMap converts an Address-like struct to a map for canonical JSON.
// Address JSON format: {"domain":"AGENT","entity":"uuid","server_port":"hub:443"}
func AddressToMap(serverPort, domain, entity string) map[string]interface{} {
	return map[string]interface{}{
		"server_port": serverPort,
		"domain":      domain,
		"entity":      entity,
	}
}
