package repository

import "encoding/json"

// mustJSON marshals a value we constructed ourselves (never user input that
// could fail to marshal) for storage in a jsonb column.
func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}
