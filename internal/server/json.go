package server

import "encoding/json/v2"

// marshalJSON encodes v as JSON with encoding/json/v2 defaults.
func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// unmarshalJSON decodes data into v with encoding/json/v2 defaults.
func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
